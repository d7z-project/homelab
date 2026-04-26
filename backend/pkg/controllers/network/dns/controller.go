package dns

import (
	apiv1 "homelab/pkg/apis/network/dns/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	dnsservice "homelab/pkg/services/network/dns"
	"net/http"
)

// ScanDomainsHandler godoc
// @Summary Scan all DNS domains
// @Tags network/dns
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]apiv1.Domain}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/domains [get]
func ScanDomainsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit, search := controllercommon.GetSearchCursorParams(r)

	res, err := dnsservice.ScanDomains(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapDomains(res))
}

// CreateDomainHandler godoc
// @Summary Create a domain
// @Tags network/dns
// @Accept json
// @Produce json
// @Param domain body apiv1.Domain true "Domain"
// @Success 200 {object} apiv1.Domain
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/domains [post]
func CreateDomainHandler(w http.ResponseWriter, r *http.Request) {
	domain, ok := controllercommon.BindRequest[apiv1.Domain](w, r)
	if !ok {
		return
	}
	model := toModelDomain(domain)
	res, err := dnsservice.CreateDomain(r.Context(), &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIDomain(*res))
}

// UpdateDomainHandler godoc
// @Summary Update a domain
// @Tags network/dns
// @Accept json
// @Produce json
// @Param id path string true "Domain ID"
// @Param domain body apiv1.Domain true "Domain"
// @Success 200 {object} apiv1.Domain
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Domain Not Found"
// @Security ApiKeyAuth
// @Router /network/dns/domains/{id} [put]
func UpdateDomainHandler(w http.ResponseWriter, r *http.Request) {
	id := controllercommon.PathID(r, "id")
	updated, ok := controllercommon.BindRequest[apiv1.Domain](w, r)
	if !ok {
		return
	}
	model := toModelDomain(updated)
	res, err := dnsservice.UpdateDomain(r.Context(), id, &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIDomain(*res))
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
	id := controllercommon.PathID(r, "id")
	if err := dnsservice.DeleteDomain(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
// @Success 200 {object} common.CursorResponse{items=[]apiv1.Record}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/dns/records [get]
func ScanRecordsHandler(w http.ResponseWriter, r *http.Request) {
	domainID := r.URL.Query().Get("domainId")
	cursor, limit, search := controllercommon.GetSearchCursorParams(r)

	res, err := dnsservice.ScanRecords(r.Context(), domainID, cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapRecords(res))
}

// CreateRecordHandler godoc
// @Summary Create a record
// @Tags network/dns
// @Accept json
// @Produce json
// @Param record body apiv1.Record true "Record"
// @Success 200 {object} apiv1.Record
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/records [post]
func CreateRecordHandler(w http.ResponseWriter, r *http.Request) {
	record, ok := controllercommon.BindRequest[apiv1.Record](w, r)
	if !ok {
		return
	}
	model := toModelRecord(record)
	res, err := dnsservice.CreateRecord(r.Context(), &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIRecord(*res))
}

// UpdateRecordHandler godoc
// @Summary Update a record
// @Tags network/dns
// @Accept json
// @Produce json
// @Param id path string true "Record ID"
// @Param record body apiv1.Record true "Record"
// @Success 200 {object} apiv1.Record
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Record Not Found"
// @Security ApiKeyAuth
// @Router /network/dns/records/{id} [put]
func UpdateRecordHandler(w http.ResponseWriter, r *http.Request) {
	id := controllercommon.PathID(r, "id")
	updated, ok := controllercommon.BindRequest[apiv1.Record](w, r)
	if !ok {
		return
	}
	model := toModelRecord(updated)
	res, err := dnsservice.UpdateRecord(r.Context(), id, &model)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIRecord(*res))
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
	id := controllercommon.PathID(r, "id")
	if err := dnsservice.DeleteRecord(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ExportHandler godoc
// @Summary Export all DNS configurations
// @Description Returns all enabled DNS domains and records in a structured format.
// @Tags network/dns
// @Produce json
// @Success 200 {object} apiv1.ExportResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/dns/export [get]
func ExportHandler(w http.ResponseWriter, r *http.Request) {
	res, err := dnsservice.ExportAll(r.Context())
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIExportResponse(res))
}
