package unit

import (
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/controllers"
	"homelab/pkg/models"
	"homelab/pkg/services/rbac"
	"homelab/pkg/services/site"
	"homelab/tests"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSiteSecurity(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	common.FS = afero.NewMemMapFs()

	engine := site.NewAnalysisEngine(nil)
	sitePoolService := site.NewSitePoolService(engine)
	siteExportManager := site.NewExportManager(engine)
	sitePoolService.SetExportManager(siteExportManager)
	controllers.InitSiteControllers(sitePoolService, engine, siteExportManager)

	r := chi.NewRouter()
	controllers.SiteRouter(r)

	// Create a Service Account and a Role with limited permissions
	ctx := tests.SetupMockRootContext()
	sa := &models.ServiceAccount{ID: "test-sa-site", Name: "Test SA Site", Enabled: true}
	_, _ = rbac.CreateServiceAccount(ctx, sa)

	// Role 1: Can only list pools
	roleRead := &models.Role{
		ID:   "role-read-site",
		Name: "Read Only Site",
		Rules: []models.PolicyRule{
			{Resource: "network/site", Verbs: []string{"list"}},
		},
	}
	_, _ = rbac.CreateRole(ctx, roleRead)
	_, _ = rbac.CreateRoleBinding(ctx, &models.RoleBinding{
		ID: "rb-read-site", Name: "RB Read Site", RoleIDs: []string{roleRead.ID}, ServiceAccountID: sa.ID, Enabled: true,
	})

	t.Run("SA with Read permissions can list pools", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/network/site/pools", nil)
		authCtx := &commonauth.AuthContext{Type: "sa", ID: sa.ID}
		req = req.WithContext(commonauth.WithAuth(req.Context(), authCtx))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("SA with Read permissions cannot create pool", func(t *testing.T) {
		body := `{"name": "New Site Pool"}`
		req := httptest.NewRequest("POST", "/network/site/pools", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		authCtx := &commonauth.AuthContext{Type: "sa", ID: sa.ID}
		req = req.WithContext(commonauth.WithAuth(req.Context(), authCtx))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("SA with execute permission can trigger site export", func(t *testing.T) {
		// First create an export as root
		exp := &models.SiteExport{Name: "Test Site Export", Rule: "true"}
		_ = sitePoolService.CreateExport(ctx, exp)

		// Grant execute permission to SA
		roleExec := &models.Role{
			ID:   "role-exec-site",
			Name: "Exec Only Site",
			Rules: []models.PolicyRule{
				{Resource: "network/site", Verbs: []string{"execute"}},
			},
		}
		_, _ = rbac.CreateRole(ctx, roleExec)
		_, _ = rbac.CreateRoleBinding(ctx, &models.RoleBinding{
			ID: "rb-exec-site", Name: "RB Exec Site", RoleIDs: []string{roleExec.ID}, ServiceAccountID: sa.ID, Enabled: true,
		})

		req := httptest.NewRequest("POST", "/network/site/exports/"+exp.ID+"/trigger", nil)
		authCtx := &commonauth.AuthContext{Type: "sa", ID: sa.ID}
		req = req.WithContext(commonauth.WithAuth(req.Context(), authCtx))

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Cleanup tasks
		siteExportManager.WaitAll()
	})
}
