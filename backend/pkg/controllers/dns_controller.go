package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListDomainsHandler godoc
// @Summary List all domains
// @Tags dns
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.Domain}
// @Security ApiKeyAuth
// @Router /dns/domains [get]
func ListDomainsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := dnsservice.ListDomains(r.Context(), page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateDomainHandler godoc
// @Summary Create a domain
// @Tags dns
// @Accept json
// @Produce json
// @Param domain body models.Domain true "Domain"
// @Success 200 {object} models.Domain
// @Security ApiKeyAuth
// @Router /dns/domains [post]
func CreateDomainHandler(w http.ResponseWriter, r *http.Request) {
	var domain models.Domain
	if err := render.Bind(r, &domain); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := dnsservice.CreateDomain(r.Context(), &domain)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateDomainHandler godoc
// @Summary Update a domain
// @Tags dns
// @Accept json
// @Produce json
// @Param id path string true "Domain ID"
// @Param domain body models.Domain true "Domain"
// @Success 200 {object} models.Domain
// @Security ApiKeyAuth
// @Router /dns/domains/{id} [put]
func UpdateDomainHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var domain models.Domain
	if err := render.Bind(r, &domain); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := dnsservice.UpdateDomain(r.Context(), id, &domain)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteDomainHandler godoc
// @Summary Delete a domain
// @Tags dns
// @Produce json
// @Param id path string true "Domain ID"
// @Success 200 {string} string "success"
// @Security ApiKeyAuth
// @Router /dns/domains/{id} [delete]
func DeleteDomainHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := dnsservice.DeleteDomain(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ListRecordsHandler godoc
// @Summary List all records
// @Tags dns
// @Produce json
// @Param domainId query string false "Filter by domain ID"
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name or value"
// @Success 200 {object} common.PaginatedResponse{items=[]models.Record}
// @Security ApiKeyAuth
// @Router /dns/records [get]
func ListRecordsHandler(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domainId")
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	res, err := dnsservice.ListRecords(r.Context(), domainID, page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateRecordHandler godoc
// @Summary Create a record
// @Tags dns
// @Accept json
// @Produce json
// @Param record body models.Record true "Record"
// @Success 200 {object} models.Record
// @Security ApiKeyAuth
// @Router /dns/records [post]
func CreateRecordHandler(w http.ResponseWriter, r *http.Request) {
	var record models.Record
	if err := render.Bind(r, &record); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := dnsservice.CreateRecord(r.Context(), &record)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// UpdateRecordHandler godoc
// @Summary Update a record
// @Tags dns
// @Accept json
// @Produce json
// @Param id path string true "Record ID"
// @Param record body models.Record true "Record"
// @Success 200 {object} models.Record
// @Security ApiKeyAuth
// @Router /dns/records/{id} [put]
func UpdateRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var record models.Record
	if err := render.Bind(r, &record); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	res, err := dnsservice.UpdateRecord(r.Context(), id, &record)
	if err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DeleteRecordHandler godoc
// @Summary Delete a record
// @Tags dns
// @Produce json
// @Param id path string true "Record ID"
// @Success 200 {string} string "success"
// @Security ApiKeyAuth
// @Router /dns/records/{id} [delete]
func DeleteRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := dnsservice.DeleteRecord(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
}

// ExportHandler godoc
// @Summary Export all DNS configurations
// @Description Returns all enabled DNS domains and records in a structured format.
// @Tags dns
// @Produce json
// @Success 200 {object} models.DnsExportResponse
// @Security ApiKeyAuth
// @Router /dns/export [get]
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	res, err := dnsservice.ExportAll(r.Context())
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// DNSRouter registers the DNS routes
func DNSRouter(r chi.Router) {
	r.Route("/dns", func(r chi.Router) {
		r.Get("/domains", ListDomainsHandler)
		r.Post("/domains", CreateDomainHandler)
		r.Put("/domains/{id}", UpdateDomainHandler)
		r.Delete("/domains/{id}", DeleteDomainHandler)

		r.Get("/records", ListRecordsHandler)
		r.Post("/records", CreateRecordHandler)
		r.Put("/records/{id}", UpdateRecordHandler)
		r.Delete("/records/{id}", DeleteRecordHandler)
	})
}
