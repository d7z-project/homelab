package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	"homelab/pkg/services/discovery"
	rbacservice "homelab/pkg/services/rbac"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListServiceAccountsHandler godoc
// @Summary Scan all service accounts
// @Tags rbac
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name or id"
// @Success 200 {object} common.CursorResponse{items=[]models.ServiceAccount}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts [get]
func ScanServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ScanServiceAccounts(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateServiceAccountHandler godoc
// @Summary Create a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param sa body models.ServiceAccount true "Service Account"
// @Success 200 {object} models.ServiceAccount
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts [post]
func CreateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	var sa models.ServiceAccount
	if err := render.Bind(r, &sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.CreateServiceAccount(r.Context(), &sa)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// UpdateServiceAccountHandler godoc
// @Summary Update a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param id path string true "Service Account ID"
// @Param sa body models.ServiceAccount true "Service Account"
// @Success 200 {object} models.ServiceAccount
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Service Account Not Found"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts/{id} [put]
func UpdateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sa models.ServiceAccount
	if err := render.Bind(r, &sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.UpdateServiceAccount(r.Context(), id, &sa)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DeleteServiceAccountHandler godoc
// @Summary Delete a service account
// @Tags rbac
// @Produce json
// @Param id path string true "Service Account ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Service Account Not Found"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts/{id} [delete]
func DeleteServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := rbacservice.DeleteServiceAccount(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ListRolesHandler godoc
// @Summary Scan all roles
// @Tags rbac
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name or id"
// @Success 200 {object} common.CursorResponse{items=[]models.Role}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/roles [get]
func ScanRolesHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ScanRoles(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateRoleHandler godoc
// @Summary Create a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param role body models.Role true "Role"
// @Success 200 {object} models.Role
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/roles [post]
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	var role models.Role
	if err := render.Bind(r, &role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.CreateRole(r.Context(), &role)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// UpdateRoleHandler godoc
// @Summary Update a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param role body models.Role true "Role"
// @Success 200 {object} models.Role
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Role Not Found"
// @Security ApiKeyAuth
// @Router /rbac/roles/{id} [put]
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var role models.Role
	if err := render.Bind(r, &role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.UpdateRole(r.Context(), id, &role)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DeleteRoleHandler godoc
// @Summary Delete a role
// @Tags rbac
// @Produce json
// @Param id path string true "Role ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Role Not Found"
// @Security ApiKeyAuth
// @Router /rbac/roles/{id} [delete]
func DeleteRoleHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := rbacservice.DeleteRole(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ListRoleBindingsHandler godoc
// @Summary Scan all role bindings
// @Tags rbac
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name or id"
// @Success 200 {object} common.CursorResponse{items=[]models.RoleBinding}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings [get]
func ScanRoleBindingsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ScanRoleBindings(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateRoleBindingHandler godoc
// @Summary Create a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param rb body models.RoleBinding true "Role Binding"
// @Success 200 {object} models.RoleBinding
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings [post]
func CreateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	var rb models.RoleBinding
	if err := render.Bind(r, &rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.CreateRoleBinding(r.Context(), &rb)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// UpdateRoleBindingHandler godoc
// @Summary Update a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param id path string true "Role Binding ID"
// @Param rb body models.RoleBinding true "Role Binding"
// @Success 200 {object} models.RoleBinding
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Role Binding Not Found"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings/{id} [put]
func UpdateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var rb models.RoleBinding
	if err := render.Bind(r, &rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.UpdateRoleBinding(r.Context(), id, &rb)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DeleteRoleBindingHandler godoc
// @Summary Delete a role binding
// @Tags rbac
// @Produce json
// @Param id path string true "Role Binding ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Role Binding Not Found"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings/{id} [delete]
func DeleteRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := rbacservice.DeleteRoleBinding(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ResetServiceAccountTokenHandler godoc
// @Summary Reset a service account token
// @Tags rbac
// @Produce json
// @Param id path string true "Service Account ID"
// @Success 200 {object} models.ServiceAccount
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Service Account Not Found"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts/{id}/reset [post]
func ResetServiceAccountTokenHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := rbacservice.ResetServiceAccountToken(r.Context(), id)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// SimulatePermissionsHandler godoc
// @Summary Simulate permissions for a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param request body models.SimulatePermissionsRequest true "Simulation Request"
// @Success 200 {object} models.ResourcePermissions
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/simulate [post]
func SimulatePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	var req models.SimulatePermissionsRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.SimulatePermissions(r.Context(), req.ServiceAccountID, req.Verb, req.Resource)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// SuggestResourcesHandler godoc
// @Summary Suggest RBAC resources
// @Tags rbac
// @Produce json
// @Param prefix query string false "Prefix to filter resources"
// @Success 200 {array} models.DiscoverResult
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /rbac/resources/suggest [get]
func SuggestResourcesHandler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	suggestions, err := discovery.SuggestResources(r.Context(), prefix)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, suggestions)
}

// SuggestVerbsHandler godoc
// @Summary Suggest RBAC verbs for a resource
// @Tags rbac
// @Produce json
// @Param resource query string false "Resource prefix"
// @Success 200 {array} string
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /rbac/verbs/suggest [get]
func SuggestVerbsHandler(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	verbs, err := discovery.SuggestVerbs(r.Context(), resource)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, verbs)
}

// RBACRouter registers the RBAC routes
func RBACRouter(r chi.Router) {
	r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/resources/suggest", SuggestResourcesHandler)
	r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/verbs/suggest", SuggestVerbsHandler)
	r.With(middlewares.RequirePermission("simulate", "rbac")).Post("/api/v1/rbac/simulate", SimulatePermissionsHandler)

	r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/serviceaccounts", ScanServiceAccountsHandler)
	r.With(middlewares.RequirePermission("create", "rbac")).Post("/api/v1/rbac/serviceaccounts", CreateServiceAccountHandler)
	r.With(middlewares.RequirePermission("update", "rbac")).Put("/api/v1/rbac/serviceaccounts/{id}", UpdateServiceAccountHandler)
	r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/api/v1/rbac/serviceaccounts/{id}", DeleteServiceAccountHandler)
	r.With(middlewares.RequirePermission("update", "rbac")).Post("/api/v1/rbac/serviceaccounts/{id}/reset", ResetServiceAccountTokenHandler)

	r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/roles", ScanRolesHandler)
	r.With(middlewares.RequirePermission("create", "rbac")).Post("/api/v1/rbac/roles", CreateRoleHandler)
	r.With(middlewares.RequirePermission("update", "rbac")).Put("/api/v1/rbac/roles/{id}", UpdateRoleHandler)
	r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/api/v1/rbac/roles/{id}", DeleteRoleHandler)

	r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/rolebindings", ScanRoleBindingsHandler)
	r.With(middlewares.RequirePermission("create", "rbac")).Post("/api/v1/rbac/rolebindings", CreateRoleBindingHandler)
	r.With(middlewares.RequirePermission("update", "rbac")).Put("/api/v1/rbac/rolebindings/{id}", UpdateRoleBindingHandler)
	r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/api/v1/rbac/rolebindings/{id}", DeleteRoleBindingHandler)
}
