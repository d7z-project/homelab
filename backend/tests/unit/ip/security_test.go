package ip_test

import (
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/controllers"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIPSecurity(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	common.FS = afero.NewMemMapFs()

	mmdbManager := ip.NewMMDBManager()
	ipPoolService := ip.NewIPPoolService(mmdbManager)
	analysisEngine := ip.NewAnalysisEngine(mmdbManager)
	exportManager := ip.NewExportManager(analysisEngine)
	ipPoolService.SetExportManager(exportManager)
	controllers.InitIPControllers(ipPoolService, analysisEngine, exportManager)

	r := chi.NewRouter()
	controllers.IPRouter(r)

	// Create a Service Account and a Role with limited permissions
	ctx := tests.SetupMockRootContext()
	sa := &models.ServiceAccount{ID: "test-sa", Name: "Test SA", Enabled: true}
	_, _ = rbac.CreateServiceAccount(ctx, sa)

	// Role 1: Can only list and get pools
	roleRead := &models.Role{
		ID:   "role-read",
		Name: "Read Only",
		Rules: []models.PolicyRule{
			{Resource: "network/ip", Verbs: []string{"list", "get"}},
		},
	}
	_, _ = rbac.CreateRole(ctx, roleRead)
	_, _ = rbac.CreateRoleBinding(ctx, &models.RoleBinding{
		ID: "rb-read", Name: "RB Read", RoleIDs: []string{roleRead.ID}, ServiceAccountID: sa.ID, Enabled: true,
	})

	t.Run("SA with Read permissions can list pools", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/network/ip/pools", nil)
		// Inject SA identity
		authCtx := &commonauth.AuthContext{Type: "sa", ID: sa.ID}
		req = req.WithContext(commonauth.WithAuth(req.Context(), authCtx))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("SA with Read permissions cannot create pool", func(t *testing.T) {
		body := `{"name": "New Pool"}`
		req := httptest.NewRequest("POST", "/network/ip/pools", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Inject SA identity
		authCtx := &commonauth.AuthContext{Type: "sa", ID: sa.ID}
		req = req.WithContext(commonauth.WithAuth(req.Context(), authCtx))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code) // RequirePermission returns 401 for permission denied
	})

	t.Run("SA with execute permission can trigger export", func(t *testing.T) {
		// First create an export as root
		exp := &models.IPExport{Name: "Test Export", Rule: "true"}
		_ = ipPoolService.CreateExport(ctx, exp)

		// Grant execute permission to SA
		roleExec := &models.Role{
			ID:   "role-exec",
			Name: "Exec Only",
			Rules: []models.PolicyRule{
				{Resource: "network/ip", Verbs: []string{"execute"}},
			},
		}
		_, _ = rbac.CreateRole(ctx, roleExec)
		_, _ = rbac.CreateRoleBinding(ctx, &models.RoleBinding{
			ID: "rb-exec", Name: "RB Exec", RoleIDs: []string{roleExec.ID}, ServiceAccountID: sa.ID, Enabled: true,
		})

		req := httptest.NewRequest("POST", "/network/ip/exports/"+exp.ID+"/trigger", nil)
		// Inject SA identity
		authCtx := &commonauth.AuthContext{Type: "sa", ID: sa.ID}
		req = req.WithContext(commonauth.WithAuth(req.Context(), authCtx))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Cleanup tasks
		exportManager.WaitAll()
	})
}
