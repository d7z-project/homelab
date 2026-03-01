package controllers

import (
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	rbacservice "homelab/pkg/services/rbac"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
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
// @Success 200 {object} common.PaginatedResponse{items=[]models.ServiceAccount}
// @Router /rbac/serviceaccounts [get]
func ListServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ListServiceAccounts(r.Context(), page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateServiceAccountHandler godoc
// @Summary Create a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param sa body models.ServiceAccount true "Service Account"
// @Success 200 {object} models.ServiceAccount
// @Router /rbac/serviceaccounts [post]
func CreateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	var sa models.ServiceAccount
	if err := json.NewDecoder(r.Body).Decode(&sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.CreateServiceAccount(r.Context(), &sa)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateServiceAccountHandler godoc
// @Summary Update a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param name path string true "Service Account Name"
// @Param sa body models.ServiceAccount true "Service Account"
// @Success 200 {object} models.ServiceAccount
// @Router /rbac/serviceaccounts/{name} [put]
func UpdateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var sa models.ServiceAccount
	if err := json.NewDecoder(r.Body).Decode(&sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.UpdateServiceAccount(r.Context(), name, &sa)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteServiceAccountHandler godoc
// @Summary Delete a service account
// @Tags rbac
// @Param name path string true "Service Account Name"
// @Success 200 {string} string "success"
// @Router /rbac/serviceaccounts/{name} [delete]
func DeleteServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := rbacservice.DeleteServiceAccount(r.Context(), name); err != nil {
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
// @Success 200 {object} common.PaginatedResponse{items=[]models.Role}
// @Router /rbac/roles [get]
func ListRolesHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ListRoles(r.Context(), page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateRoleHandler godoc
// @Summary Create a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param role body models.Role true "Role"
// @Success 200 {object} models.Role
// @Router /rbac/roles [post]
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	var role models.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.CreateRole(r.Context(), &role)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateRoleHandler godoc
// @Summary Update a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param name path string true "Role Name"
// @Param role body models.Role true "Role"
// @Success 200 {object} models.Role
// @Router /rbac/roles/{name} [put]
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var role models.Role
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.UpdateRole(r.Context(), name, &role)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteRoleHandler godoc
// @Summary Delete a role
// @Tags rbac
// @Param name path string true "Role Name"
// @Success 200 {string} string "success"
// @Router /rbac/roles/{name} [delete]
func DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := rbacservice.DeleteRole(r.Context(), name); err != nil {
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
// @Success 200 {object} common.PaginatedResponse{items=[]models.RoleBinding}
// @Router /rbac/rolebindings [get]
func ListRoleBindingsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ListRoleBindings(r.Context(), page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateRoleBindingHandler godoc
// @Summary Create a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param rb body models.RoleBinding true "Role Binding"
// @Success 200 {object} models.RoleBinding
// @Router /rbac/rolebindings [post]
func CreateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	var rb models.RoleBinding
	if err := json.NewDecoder(r.Body).Decode(&rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.CreateRoleBinding(r.Context(), &rb)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateRoleBindingHandler godoc
// @Summary Update a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param name path string true "Role Binding Name"
// @Param rb body models.RoleBinding true "Role Binding"
// @Success 200 {object} models.RoleBinding
// @Router /rbac/rolebindings/{name} [put]
func UpdateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var rb models.RoleBinding
	if err := json.NewDecoder(r.Body).Decode(&rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.UpdateRoleBinding(r.Context(), name, &rb)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteRoleBindingHandler godoc
// @Summary Delete a role binding
// @Tags rbac
// @Param name path string true "Role Binding Name"
// @Success 200 {string} string "success"
// @Router /rbac/rolebindings/{name} [delete]
func DeleteRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := rbacservice.DeleteRoleBinding(r.Context(), name); err != nil {
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
// @Success 200 {object} models.ServiceAccount
// @Router /rbac/serviceaccounts/{name}/reset [post]
func ResetServiceAccountTokenHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	res, err := rbacservice.ResetServiceAccountToken(r.Context(), name)
	if err != nil {
		common.BadRequestError(w, r, http.StatusNotFound, err.Error())
		return
	}
	common.Success(w, r, res)
}

type SimulatePermissionsRequest struct {
	ServiceAccountName string `json:"serviceAccountName"`
	Verb               string `json:"verb"`
	Resource           string `json:"resource"`
}

// SimulatePermissionsHandler godoc
// @Summary Simulate permissions for a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param request body SimulatePermissionsRequest true "Simulation Request"
// @Success 200 {object} models.ResourcePermissions
// @Router /rbac/simulate [post]
func SimulatePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	var req SimulatePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.SimulatePermissions(r.Context(), req.ServiceAccountName, req.Verb, req.Resource)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// RBACRouter registers the RBAC routes
func RBACRouter(r chi.Router) {
	r.Route("/rbac", func(r chi.Router) {
		r.Post("/simulate", SimulatePermissionsHandler)
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
