package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/discovery"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// DiscoveryController handles lookups for frontend dropdowns/selectors
func DiscoveryController(r chi.Router) {
	r.Get("/lookup", lookupHandler)
	r.Get("/codes", listCodesHandler)
}

// @Summary Discovery lookup
// @Description Search for items in a specific discovery code (e.g. dns/domains)
// @Tags discovery
// @Accept json
// @Produce json
// @Param code query string true "Discovery code"
// @Param search query string false "Search string"
// @Param offset query int false "Offset"
// @Param limit query int false "Limit"
// @Success 200 {object} models.LookupResponse
// @Router /discovery/lookup [get]
// @Security ApiKeyAuth
func lookupHandler(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	search := r.URL.Query().Get("search")
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	req := models.LookupRequest{
		Code:   code,
		Search: search,
		Offset: offset,
		Limit:  limit,
	}

	if err := req.Bind(r); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	items, total, err := discovery.Lookup(r.Context(), req)
	if err != nil {
		if err == discovery.ErrCodeNotFound {
			common.Error(w, r, http.StatusNotFound, http.StatusNotFound, err.Error())
			return
		}
		HandleError(w, r, err)
		return
	}

	common.Success(w, r, &models.LookupResponse{
		Items: items,
		Total: total,
	})
}

// @Summary List discovery codes
// @Description Returns all registered discovery codes
// @Tags discovery
// @Accept json
// @Produce json
// @Success 200 {array} string
// @Router /discovery/codes [get]
// @Security ApiKeyAuth
func listCodesHandler(w http.ResponseWriter, r *http.Request) {
	codes := discovery.GetRegisteredCodes()
	common.Success(w, r, codes)
}
