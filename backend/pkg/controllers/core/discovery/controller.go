package discovery

import (
	apiv1 "homelab/pkg/apis/core/discovery/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	registryruntime "homelab/pkg/runtime/registry"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// DiscoveryController handles lookups for frontend dropdowns/selectors
func DiscoveryController(r chi.Router) {
	r.Get("/lookup", lookupHandler)
	r.Get("/codes", ScanCodesHandler)
}

// @Summary Discovery lookup
// @Description Search for items in a specific discovery code (e.g. network/dns/domains)
// @Tags discovery
// @Accept json
// @Produce json
// @Param code query string true "Discovery code"
// @Param search query string false "Search string"
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Success 200 {object} common.CursorResponse{items=[]models.LookupItem}
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Code Not Found"
// @Router /discovery/lookup [get]
// @Security ApiKeyAuth
func lookupHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	search := r.URL.Query().Get("search")
	cursor, limit := controllercommon.GetCursorParams(r)

	req := apiv1.LookupRequest{
		Code:   code,
		Search: search,
		Cursor: cursor,
		Limit:  limit,
	}

	if err := req.Bind(r); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := registryruntime.Default().Lookup(r.Context(), toModelLookupRequest(req))
	if err != nil {
		if err == registryruntime.ErrCodeNotFound {
			common.Error(w, r, http.StatusNotFound, http.StatusNotFound, err.Error())
			return
		}
		controllercommon.HandleError(w, r, err)
		return
	}

	common.CursorSuccess(w, r, mapLookupItems(res))
}

// @Summary Scan discovery codes
// @Description Returns all registered discovery codes
// @Tags discovery
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Failure 401 {object} common.Response "Unauthorized"
// @Router /discovery/codes [get]
// @Security ApiKeyAuth
func ScanCodesHandler(w http.ResponseWriter, r *http.Request) {
	codes := registryruntime.Default().ScanCodes()
	common.Success(w, r, codes)
}
