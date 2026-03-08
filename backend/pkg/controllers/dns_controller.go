package controllers

import (
	"fmt"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)
// ScanDomainsHandler godoc
// @Summary Scan all DNS domains
// @Tags network/dns
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.Domain}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/domains [get]
func ScanDomainsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := dnsservice.ScanDomains(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateDomainHandler godoc
// @Summary Create a domain
// @Tags network/dns
// @Accept json
// @Produce json
// @Param domain body models.Domain true "Domain"
// @Success 200 {object} models.Domain
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/domains [post]
func CreateDomainHandler(w http.ResponseWriter, r *http.Request) {
	var domain models.Domain
	if err := render.Bind(r, &domain); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := dnsservice.CreateDomain(r.Context(), &domain)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// UpdateDomainHandler godoc
// @Summary Update a domain
// @Tags network/dns
// @Accept json
// @Produce json
// @Param id path string true "Domain ID"
// @Param domain body models.Domain true "Domain"
// @Success 200 {object} models.Domain
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Domain Not Found"
// @Security ApiKeyAuth
// @Router /network/dns/domains/{id} [put]
func UpdateDomainHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	domain, err := dnsservice.GetDomain(r.Context(), id)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns/" + domain.Name) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns") {
		HandleError(w, r, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, domain.Name))
		return
	}
	var updated models.Domain
	if err := render.Bind(r, &updated); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := dnsservice.UpdateDomain(r.Context(), id, &updated)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DeleteDomainHandler godoc
// @Summary Delete a domain
// @Tags network/dns
// @Produce json
// @Param id path string true "Domain ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Domain Not Found"
// @Security ApiKeyAuth
// @Router /network/dns/domains/{id} [delete]
func DeleteDomainHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	domain, err := dnsservice.GetDomain(r.Context(), id)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns/" + domain.Name) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns") {
		HandleError(w, r, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, domain.Name))
		return
	}
	if err := dnsservice.DeleteDomain(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ScanRecordsHandler godoc
// @Summary Scan DNS records
// @Tags network/dns
// @Produce json
// @Param domainId query string false "Domain ID"
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.Record}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/dns/records [get]
func ScanRecordsHandler(w http.ResponseWriter, r *http.Request) {

	domainID := r.URL.Query().Get("domainId")
	if domainID != "" {
		domain, err := dnsservice.GetDomain(r.Context(), domainID)
		if err == nil {
			if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns/"+domain.Name) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns") {
				HandleError(w, r, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, domain.Name))
				return
			}
		}
	}
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := dnsservice.ScanRecords(r.Context(), domainID, cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateRecordHandler godoc
// @Summary Create a record
// @Tags network/dns
// @Accept json
// @Produce json
// @Param record body models.Record true "Record"
// @Success 200 {object} models.Record
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/records [post]
func CreateRecordHandler(w http.ResponseWriter, r *http.Request) {
	var record models.Record
	if err := render.Bind(r, &record); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	domain, err := dnsservice.GetDomain(r.Context(), record.DomainID)
	if err == nil {
		if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns/"+domain.Name) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns") {
			HandleError(w, r, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, domain.Name))
			return
		}
	}
	res, err := dnsservice.CreateRecord(r.Context(), &record)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// UpdateRecordHandler godoc
// @Summary Update a record
// @Tags network/dns
// @Accept json
// @Produce json
// @Param id path string true "Record ID"
// @Param record body models.Record true "Record"
// @Success 200 {object} models.Record
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Record Not Found"
// @Security ApiKeyAuth
// @Router /network/dns/records/{id} [put]
func UpdateRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	record, err := dnsservice.GetRecord(r.Context(), id)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	domain, err := dnsservice.GetDomain(r.Context(), record.DomainID)
	if err == nil {
		if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns/"+domain.Name) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns") {
			HandleError(w, r, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, domain.Name))
			return
		}
	}
	var updated models.Record
	if err := render.Bind(r, &updated); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := dnsservice.UpdateRecord(r.Context(), id, &updated)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DeleteRecordHandler godoc
// @Summary Delete a record
// @Tags network/dns
// @Produce json
// @Param id path string true "Record ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Record Not Found"
// @Security ApiKeyAuth
// @Router /network/dns/records/{id} [delete]
func DeleteRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	record, err := dnsservice.GetRecord(r.Context(), id)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	domain, err := dnsservice.GetDomain(r.Context(), record.DomainID)
	if err == nil {
		if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns/"+domain.Name) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/dns") {
			HandleError(w, r, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, domain.Name))
			return
		}
	}
	if err := dnsservice.DeleteRecord(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ExportHandler godoc
// @Summary Export all DNS configurations
// @Description Returns all enabled DNS domains and records in a structured format.
// @Tags network/dns
// @Produce json
// @Success 200 {object} models.DnsExportResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/export [get]
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	res, err := dnsservice.ExportAll(r.Context())
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DNSRouter registers the DNS routes
func DNSRouter(r chi.Router) {
	r.Route("/network/dns", func(r chi.Router) {
		r.With(middlewares.RequirePermission("get", "network/dns")).Get("/export", ExportHandler)

		r.With(middlewares.RequirePermission("list", "network/dns")).Get("/domains", ScanDomainsHandler)
		r.With(middlewares.RequirePermission("create", "network/dns")).Post("/domains", CreateDomainHandler)
		r.With(middlewares.RequirePermission("update", "network/dns")).Put("/domains/{id}", UpdateDomainHandler)
		r.With(middlewares.RequirePermission("delete", "network/dns")).Delete("/domains/{id}", DeleteDomainHandler)

		r.With(middlewares.RequirePermission("list", "network/dns")).Get("/records", ScanRecordsHandler)
		r.With(middlewares.RequirePermission("create", "network/dns")).Post("/records", CreateRecordHandler)
		r.With(middlewares.RequirePermission("update", "network/dns")).Put("/records/{id}", UpdateRecordHandler)
		r.With(middlewares.RequirePermission("delete", "network/dns")).Delete("/records/{id}", DeleteRecordHandler)
	})
}
