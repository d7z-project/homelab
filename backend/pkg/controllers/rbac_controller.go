package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	rbacservice "homelab/pkg/services/rbac"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListServiceAccountsHandler godoc
// @Summary List all service accounts
// @Tags rbac
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name or id"
// @Success 200 {object} common.PaginatedResponse{items=[]models.ServiceAccount}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts [get]
func ListServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ListServiceAccounts(r.Context(), page, pageSize, search)
	if err != nil {
		HandleError(w, r, err)
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
// @Summary List all roles
// @Tags rbac
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name or id"
// @Success 200 {object} common.PaginatedResponse{items=[]models.Role}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/roles [get]
func ListRolesHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ListRoles(r.Context(), page, pageSize, search)
	if err != nil {
		HandleError(w, r, err)
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
// @Summary List all role bindings
// @Tags rbac
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name or id"
// @Success 200 {object} common.PaginatedResponse{items=[]models.RoleBinding}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings [get]
func ListRoleBindingsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ListRoleBindings(r.Context(), page, pageSize, search)
	if err != nil {
		HandleError(w, r, err)
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
	suggestions, err := rbacservice.SuggestResources(r.Context(), prefix)
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
	verbs, err := rbacservice.SuggestVerbs(r.Context(), resource)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, verbs)
}

// RBACRouter registers the RBAC routes
func RBACRouter(r chi.Router) {
	r.Route("/rbac", func(r chi.Router) {
		r.Get("/resources/suggest", SuggestResourcesHandler)
		r.Get("/verbs/suggest", SuggestVerbsHandler)
		r.With(middlewares.RequirePermission("simulate", "rbac")).Post("/simulate", SimulatePermissionsHandler)

		r.Get("/serviceaccounts", ListServiceAccountsHandler)
		r.With(middlewares.RequirePermission("create", "rbac")).Post("/serviceaccounts", CreateServiceAccountHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Put("/serviceaccounts/{id}", UpdateServiceAccountHandler)
		r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/serviceaccounts/{id}", DeleteServiceAccountHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Post("/serviceaccounts/{id}/reset", ResetServiceAccountTokenHandler)

		r.Get("/roles", ListRolesHandler)
		r.With(middlewares.RequirePermission("create", "rbac")).Post("/roles", CreateRoleHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Put("/roles/{id}", UpdateRoleHandler)
		r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/roles/{id}", DeleteRoleHandler)

		r.Get("/rolebindings", ListRoleBindingsHandler)
		r.With(middlewares.RequirePermission("create", "rbac")).Post("/rolebindings", CreateRoleBindingHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Put("/rolebindings/{id}", UpdateRoleBindingHandler)
		r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/rolebindings/{id}", DeleteRoleBindingHandler)
	})
}
