package rbac

import (
	apiv1 "homelab/pkg/apis/core/rbac/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	registryruntime "homelab/pkg/runtime/registry"
	rbacservice "homelab/pkg/services/core/rbac"
	"net/http"

	discoverymodel "homelab/pkg/models/core/discovery"

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
// @Success 200 {object} common.CursorResponse{items=[]apiv1.ServiceAccount}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts [get]
func ScanServiceAccountsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ScanServiceAccounts(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapServiceAccounts(res))
}

// CreateServiceAccountHandler godoc
// @Summary Create a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param sa body apiv1.ServiceAccount true "Service Account"
// @Success 200 {object} apiv1.ServiceAccount
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts [post]
func CreateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	var sa apiv1.ServiceAccount
	if err := render.Bind(r, &sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelServiceAccount(sa)
	res, err := rbacservice.CreateServiceAccount(r.Context(), &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIServiceAccount(*res))
}

// UpdateServiceAccountHandler godoc
// @Summary Update a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param id path string true "Service Account ID"
// @Param sa body apiv1.ServiceAccount true "Service Account"
// @Success 200 {object} apiv1.ServiceAccount
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Service Account Not Found"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts/{id} [put]
func UpdateServiceAccountHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var sa apiv1.ServiceAccount
	if err := render.Bind(r, &sa); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelServiceAccount(sa)
	res, err := rbacservice.UpdateServiceAccount(r.Context(), id, &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIServiceAccount(*res))
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
		controllercommon.HandleError(w, r, err)
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
// @Success 200 {object} common.CursorResponse{items=[]apiv1.Role}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/roles [get]
func ScanRolesHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ScanRoles(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapRoles(res))
}

// CreateRoleHandler godoc
// @Summary Create a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param role body apiv1.Role true "Role"
// @Success 200 {object} apiv1.Role
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/roles [post]
func CreateRoleHandler(w http.ResponseWriter, r *http.Request) {
	var role apiv1.Role
	if err := render.Bind(r, &role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelRole(role)
	res, err := rbacservice.CreateRole(r.Context(), &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIRole(*res))
}

// UpdateRoleHandler godoc
// @Summary Update a role
// @Tags rbac
// @Accept json
// @Produce json
// @Param id path string true "Role ID"
// @Param role body apiv1.Role true "Role"
// @Success 200 {object} apiv1.Role
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Role Not Found"
// @Security ApiKeyAuth
// @Router /rbac/roles/{id} [put]
func UpdateRoleHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var role apiv1.Role
	if err := render.Bind(r, &role); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelRole(role)
	res, err := rbacservice.UpdateRole(r.Context(), id, &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIRole(*res))
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
		controllercommon.HandleError(w, r, err)
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
// @Success 200 {object} common.CursorResponse{items=[]apiv1.RoleBinding}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings [get]
func ScanRoleBindingsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := rbacservice.ScanRoleBindings(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapRoleBindings(res))
}

// CreateRoleBindingHandler godoc
// @Summary Create a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param rb body apiv1.RoleBinding true "Role Binding"
// @Success 200 {object} apiv1.RoleBinding
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings [post]
func CreateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	var rb apiv1.RoleBinding
	if err := render.Bind(r, &rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelRoleBinding(rb)
	res, err := rbacservice.CreateRoleBinding(r.Context(), &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIRoleBinding(*res))
}

// UpdateRoleBindingHandler godoc
// @Summary Update a role binding
// @Tags rbac
// @Accept json
// @Produce json
// @Param id path string true "Role Binding ID"
// @Param rb body apiv1.RoleBinding true "Role Binding"
// @Success 200 {object} apiv1.RoleBinding
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Role Binding Not Found"
// @Security ApiKeyAuth
// @Router /rbac/rolebindings/{id} [put]
func UpdateRoleBindingHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var rb apiv1.RoleBinding
	if err := render.Bind(r, &rb); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	model := toModelRoleBinding(rb)
	res, err := rbacservice.UpdateRoleBinding(r.Context(), id, &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIRoleBinding(*res))
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
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ResetServiceAccountTokenHandler godoc
// @Summary Reset a service account token
// @Tags rbac
// @Produce json
// @Param id path string true "Service Account ID"
// @Success 200 {object} apiv1.ServiceAccount
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Service Account Not Found"
// @Security ApiKeyAuth
// @Router /rbac/serviceaccounts/{id}/reset [post]
func ResetServiceAccountTokenHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	res, err := rbacservice.ResetServiceAccountToken(r.Context(), id)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIServiceAccount(*res))
}

// SimulatePermissionsHandler godoc
// @Summary Simulate permissions for a service account
// @Tags rbac
// @Accept json
// @Produce json
// @Param request body apiv1.SimulatePermissionsRequest true "Simulation Request"
// @Success 200 {object} apiv1.ResourcePermissions
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /rbac/simulate [post]
func SimulatePermissionsHandler(w http.ResponseWriter, r *http.Request) {
	var req apiv1.SimulatePermissionsRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := rbacservice.SimulatePermissions(r.Context(), req.ServiceAccountID, req.Verb, req.Resource)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIResourcePermissions(res))
}

// SuggestResourcesHandler godoc
// @Summary Suggest RBAC resources
// @Tags rbac
// @Produce json
// @Param prefix query string false "Prefix to filter resources"
// @Success 200 {array} discoverymodel.DiscoverResult
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /rbac/resources/suggest [get]
func SuggestResourcesHandler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")
	suggestions, err := registryruntime.Default().SuggestResources(r.Context(), prefix)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, []discoverymodel.DiscoverResult(suggestions))
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
	verbs, err := registryruntime.Default().SuggestVerbs(r.Context(), resource)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, verbs)
}
