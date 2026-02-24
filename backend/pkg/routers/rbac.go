package routers

import (
	"encoding/json"
	"homelab/pkg/auth"
	"homelab/pkg/common"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ListServiceAccountsHandler godoc
// @Summary List all service accounts
// @Tags rbac
// @Produce json
// @Success 200 {array} auth.ServiceAccount
// @Router /rbac/serviceaccounts [get]
func ListServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	sas, err := auth.ListServiceAccounts(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, sas)
}

// CreateServiceAccountHandler godoc
// @Summary Create a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param sa body auth.ServiceAccount true "Service Account"
// @Success 200 {object} auth.ServiceAccount
// @Router /rbac/serviceaccounts [post]
func CreateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	var sa auth.ServiceAccount
	if err := json.NewDecoder(r.Body).Decode(&sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if sa.Name == "" {
		common.BadRequestError(w, r, http.StatusBadRequest, "name is required")
		return
	}

	// If it's an update, preserve existing token if not provided
	existing, _ := auth.GetServiceAccount(r.Context(), sa.Name)
	if existing != nil {
		if sa.Token == "" {
			sa.Token = existing.Token
		}
	} else {
		if sa.Token == "" {
			sa.Token = uuid.New().String()
		}
	}

	if err := auth.SaveServiceAccount(r.Context(), &sa); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, sa)
}

// DeleteServiceAccountHandler godoc
// @Summary Delete a service account
// @Tags rbac
// @Param name path string true "Service Account Name"
// @Success 200 {string} string "success"
// @Router /rbac/serviceaccounts/{name} [delete]
func DeleteServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := auth.DeleteServiceAccount(r.Context(), name); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListRolesHandler godoc
// @Summary List all roles
// @Tags rbac
// @Produce json
// @Success 200 {array} auth.Role
// @Router /rbac/roles [get]
func ListRolesHandler(w http.ResponseWriter, r *http.Request) {
	roles, err := auth.ListRoles(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, roles)
}

// CreateRoleHandler godoc
// @Summary Create a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param role body auth.Role true "Role"
// @Success 200 {object} auth.Role
// @Router /rbac/roles [post]
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	var role auth.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if role.Name == "" {
		common.BadRequestError(w, r, http.StatusBadRequest, "name is required")
		return
	}
	if err := auth.SaveRole(r.Context(), &role); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, role)
}

// DeleteRoleHandler godoc
// @Summary Delete a role
// @Tags rbac
// @Param name path string true "Role Name"
// @Success 200 {string} string "success"
// @Router /rbac/roles/{name} [delete]
func DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := auth.DeleteRole(r.Context(), name); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListRoleBindingsHandler godoc
// @Summary List all role bindings
// @Tags rbac
// @Produce json
// @Success 200 {array} auth.RoleBinding
// @Router /rbac/rolebindings [get]
func ListRoleBindingsHandler(w http.ResponseWriter, r *http.Request) {
	rbs, err := auth.ListRoleBindings(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, rbs)
}

// CreateRoleBindingHandler godoc
// @Summary Create a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param rb body auth.RoleBinding true "Role Binding"
// @Success 200 {object} auth.RoleBinding
// @Router /rbac/rolebindings [post]
func CreateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	var rb auth.RoleBinding
	if err := json.NewDecoder(r.Body).Decode(&rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if rb.Name == "" {
		common.BadRequestError(w, r, http.StatusBadRequest, "name is required")
		return
	}
	if err := auth.SaveRoleBinding(r.Context(), &rb); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, rb)
}

// DeleteRoleBindingHandler godoc
// @Summary Delete a role binding
// @Tags rbac
// @Param name path string true "Role Binding Name"
// @Success 200 {string} string "success"
// @Router /rbac/rolebindings/{name} [delete]
func DeleteRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := auth.DeleteRoleBinding(r.Context(), name); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ResetServiceAccountTokenHandler godoc
// @Summary Reset a service account token
// @Tags rbac
// @Produce json
// @Param name path string true "Service Account Name"
// @Success 200 {object} auth.ServiceAccount
// @Router /rbac/serviceaccounts/{name}/reset [post]
func ResetServiceAccountTokenHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	sa, err := auth.GetServiceAccount(r.Context(), name)
	if err != nil {
		common.BadRequestError(w, r, http.StatusNotFound, "service account not found")
		return
	}

	sa.Token = uuid.New().String()
	if err := auth.SaveServiceAccount(r.Context(), sa); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, sa)
}

// RBACRouter registers the RBAC routes
func RBACRouter(r chi.Router) {
	r.Route("/rbac", func(r chi.Router) {
		r.Get("/serviceaccounts", ListServiceAccountsHandler)
		r.Post("/serviceaccounts", CreateServiceAccountHandler)
		r.Delete("/serviceaccounts/{name}", DeleteServiceAccountHandler)
		r.Post("/serviceaccounts/{name}/reset", ResetServiceAccountTokenHandler)
		r.Get("/roles", ListRolesHandler)
		r.Post("/roles", CreateRoleHandler)
		r.Delete("/roles/{name}", DeleteRoleHandler)
		r.Get("/rolebindings", ListRoleBindingsHandler)
		r.Post("/rolebindings", CreateRoleBindingHandler)
		r.Delete("/rolebindings/{name}", DeleteRoleBindingHandler)
	})
}
