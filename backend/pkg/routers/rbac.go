package routers

import (
	"encoding/json"
	"homelab/pkg/auth"
	"homelab/pkg/common"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func getPaginationParams(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize < 1 {
		pageSize = 15
	}
	return page, pageSize
}

// ListServiceAccountsHandler godoc
// @Summary List all service accounts
// @Tags rbac
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]auth.ServiceAccount}
// @Router /rbac/serviceaccounts [get]
func ListServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")
	sas, total, err := auth.ListServiceAccounts(r.Context(), uint64(page-1), uint(pageSize), search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.PaginatedSuccess(w, r, sas, int(total), page, pageSize)
}

// CreateServiceAccountHandler godoc
// @Summary Create a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param sa body auth.ServiceAccount true "Service Account"
// @Success 200 {object} auth.ServiceAccount
// @Failure 400 {object} common.Response
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

	// Check if already exists
	existing, _ := auth.GetServiceAccount(r.Context(), sa.Name)
	if existing != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, "ServiceAccount already exists")
		return
	}

	if sa.Token == "" {
		sa.Token = uuid.New().String()
	}

	if err := auth.SaveServiceAccount(r.Context(), &sa); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, sa)
}

// UpdateServiceAccountHandler godoc
// @Summary Update a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param name path string true "Service Account Name"
// @Param sa body auth.ServiceAccount true "Service Account"
// @Success 200 {object} auth.ServiceAccount
// @Router /rbac/serviceaccounts/{name} [put]
func UpdateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var sa auth.ServiceAccount
	if err := json.NewDecoder(r.Body).Decode(&sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if sa.Name != name {
		common.BadRequestError(w, r, http.StatusBadRequest, "name in body does not match path")
		return
	}

	existing, err := auth.GetServiceAccount(r.Context(), name)
	if err != nil {
		common.BadRequestError(w, r, http.StatusNotFound, "ServiceAccount not found")
		return
	}

	if sa.Token == "" {
		sa.Token = existing.Token
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
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]auth.Role}
// @Router /rbac/roles [get]
func ListRolesHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")
	roles, total, err := auth.ListRoles(r.Context(), uint64(page-1), uint(pageSize), search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.PaginatedSuccess(w, r, roles, int(total), page, pageSize)
}

// CreateRoleHandler godoc
// @Summary Create a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param role body auth.Role true "Role"
// @Success 200 {object} auth.Role
// @Failure 400 {object} common.Response
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

	// Check if already exists
	existing, _ := auth.GetRole(r.Context(), role.Name)
	if existing != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, "Role already exists")
		return
	}

	if err := auth.SaveRole(r.Context(), &role); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, role)
}

// UpdateRoleHandler godoc
// @Summary Update a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param name path string true "Role Name"
// @Param role body auth.Role true "Role"
// @Success 200 {object} auth.Role
// @Router /rbac/roles/{name} [put]
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var role auth.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if role.Name != name {
		common.BadRequestError(w, r, http.StatusBadRequest, "name in body does not match path")
		return
	}

	_, err := auth.GetRole(r.Context(), name)
	if err != nil {
		common.BadRequestError(w, r, http.StatusNotFound, "Role not found")
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
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]auth.RoleBinding}
// @Router /rbac/rolebindings [get]
func ListRoleBindingsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")
	rbs, total, err := auth.ListRoleBindings(r.Context(), uint64(page-1), uint(pageSize), search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.PaginatedSuccess(w, r, rbs, int(total), page, pageSize)
}

// CreateRoleBindingHandler godoc
// @Summary Create a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param rb body auth.RoleBinding true "Role Binding"
// @Success 200 {object} auth.RoleBinding
// @Failure 400 {object} common.Response
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

	// Check if already exists
	existing, _ := auth.GetRoleBinding(r.Context(), rb.Name)
	if existing != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, "RoleBinding already exists")
		return
	}

	if err := auth.SaveRoleBinding(r.Context(), &rb); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, rb)
}

// UpdateRoleBindingHandler godoc
// @Summary Update a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param name path string true "Role Binding Name"
// @Param rb body auth.RoleBinding true "Role Binding"
// @Success 200 {object} auth.RoleBinding
// @Router /rbac/rolebindings/{name} [put]
func UpdateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var rb auth.RoleBinding
	if err := json.NewDecoder(r.Body).Decode(&rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if rb.Name != name {
		common.BadRequestError(w, r, http.StatusBadRequest, "name in body does not match path")
		return
	}

	_, err := auth.GetRoleBinding(r.Context(), name)
	if err != nil {
		common.BadRequestError(w, r, http.StatusNotFound, "RoleBinding not found")
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
		r.Put("/serviceaccounts/{name}", UpdateServiceAccountHandler)
		r.Delete("/serviceaccounts/{name}", DeleteServiceAccountHandler)
		r.Post("/serviceaccounts/{name}/reset", ResetServiceAccountTokenHandler)
		r.Get("/roles", ListRolesHandler)
		r.Post("/roles", CreateRoleHandler)
		r.Put("/roles/{name}", UpdateRoleHandler)
		r.Delete("/roles/{name}", DeleteRoleHandler)
		r.Get("/rolebindings", ListRoleBindingsHandler)
		r.Post("/rolebindings", CreateRoleBindingHandler)
		r.Put("/rolebindings/{name}", UpdateRoleBindingHandler)
		r.Delete("/rolebindings/{name}", DeleteRoleBindingHandler)
	})
}
