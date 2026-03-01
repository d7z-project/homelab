package controllers

import (
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ListDomainsHandler godoc
// @Summary List all DNS domains
// @Tags dns
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.Domain}
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
// @Summary Create a DNS domain
// @Tags dns
// @Accept json
// @Produce json
// @Param domain body models.Domain true "Domain"
// @Success 200 {object} models.Domain
// @Router /dns/domains [post]
func CreateDomainHandler(w http.ResponseWriter, r *http.Request) {
	var domain models.Domain
	if err := json.NewDecoder(r.Body).Decode(&domain); err != nil {
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
// @Summary Update a DNS domain
// @Tags dns
// @Accept json
// @Produce json
// @Param id path string true "Domain ID"
// @Param domain body models.Domain true "Domain"
// @Success 200 {object} models.Domain
// @Router /dns/domains/{id} [put]
func UpdateDomainHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var domain models.Domain
	if err := json.NewDecoder(r.Body).Decode(&domain); err != nil {
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
// @Summary Delete a DNS domain
// @Tags dns
// @Param id path string true "Domain ID"
// @Success 200 {string} string "success"
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
// @Summary List all DNS records
// @Tags dns
// @Produce json
// @Param domainId query string false "Domain ID filter"
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name or value"
// @Success 200 {object} common.PaginatedResponse{items=[]models.Record}
// @Router /dns/records [get]
func ListRecordsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")
	domainID := r.URL.Query().Get("domainId")

	res, err := dnsservice.ListRecords(r.Context(), domainID, page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, res)
}

// CreateRecordHandler godoc
// @Summary Create a DNS record
// @Tags dns
// @Accept json
// @Produce json
// @Param record body models.Record true "Record"
// @Success 200 {object} models.Record
// @Router /dns/records [post]
func CreateRecordHandler(w http.ResponseWriter, r *http.Request) {
	var record models.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
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
// @Summary Update a DNS record
// @Tags dns
// @Accept json
// @Produce json
// @Param id path string true "Record ID"
// @Param record body models.Record true "Record"
// @Success 200 {object} models.Record
// @Router /dns/records/{id} [put]
func UpdateRecordRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var record models.Record
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
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
// @Summary Delete a DNS record
// @Tags dns
// @Param id path string true "Record ID"
// @Success 200 {string} string "success"
// @Router /dns/records/{id} [delete]
func DeleteRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := dnsservice.DeleteRecord(r.Context(), id); err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, "success")
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
		r.Put("/records/{id}", UpdateRecordRecordHandler)
		r.Delete("/records/{id}", DeleteRecordHandler)
	})
}
