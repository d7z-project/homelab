package dns

import (
	apiv1 "homelab/pkg/apis/network/dns/v1"
	dnsmodel "homelab/pkg/models/network/dns"
	"homelab/pkg/models/shared"
)

func toModelDomain(api apiv1.Domain) dnsmodel.Domain {
	return dnsmodel.Domain{
		ID: api.ID,
		Meta: dnsmodel.DomainV1Meta{
			Name:        api.Meta.Name,
			Enabled:     api.Meta.Enabled,
			Description: api.Meta.Description,
		},
		Status: dnsmodel.DomainV1Status{
			CreatedAt: api.Status.CreatedAt,
			UpdatedAt: api.Status.UpdatedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIDomain(model dnsmodel.Domain) apiv1.Domain {
	return apiv1.Domain{
		ID: model.ID,
		Meta: apiv1.DomainMeta{
			Name:        model.Meta.Name,
			Enabled:     model.Meta.Enabled,
			Description: model.Meta.Description,
		},
		Status: apiv1.DomainStatus{
			CreatedAt: model.Status.CreatedAt,
			UpdatedAt: model.Status.UpdatedAt,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelRecord(api apiv1.Record) dnsmodel.Record {
	return dnsmodel.Record{
		ID: api.ID,
		Meta: dnsmodel.RecordV1Meta{
			DomainID: api.Meta.DomainID,
			Name:     api.Meta.Name,
			Type:     api.Meta.Type,
			Value:    api.Meta.Value,
			TTL:      api.Meta.TTL,
			Priority: api.Meta.Priority,
			Enabled:  api.Meta.Enabled,
			Comments: api.Meta.Comments,
		},
		Status:          dnsmodel.RecordV1Status{},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIRecord(model dnsmodel.Record) apiv1.Record {
	return apiv1.Record{
		ID: model.ID,
		Meta: apiv1.RecordMeta{
			DomainID: model.Meta.DomainID,
			Name:     model.Meta.Name,
			Type:     model.Meta.Type,
			Value:    model.Meta.Value,
			TTL:      model.Meta.TTL,
			Priority: model.Meta.Priority,
			Enabled:  model.Meta.Enabled,
			Comments: model.Meta.Comments,
		},
		Status:          apiv1.RecordStatus{},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func mapDomains(res *shared.PaginationResponse[dnsmodel.Domain]) *shared.PaginationResponse[apiv1.Domain] {
	items := make([]apiv1.Domain, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIDomain(item))
	}
	return &shared.PaginationResponse[apiv1.Domain]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapRecords(res *shared.PaginationResponse[dnsmodel.Record]) *shared.PaginationResponse[apiv1.Record] {
	items := make([]apiv1.Record, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIRecord(item))
	}
	return &shared.PaginationResponse[apiv1.Record]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func toAPIExportResponse(model *dnsmodel.DnsExportResponse) *apiv1.ExportResponse {
	if model == nil {
		return nil
	}
	res := &apiv1.ExportResponse{Domains: make([]apiv1.ExportDomain, 0, len(model.Domains))}
	for _, domain := range model.Domains {
		apiDomain := apiv1.ExportDomain{Name: domain.Name, Records: make([]apiv1.ExportRecord, 0, len(domain.Records))}
		for _, record := range domain.Records {
			apiDomain.Records = append(apiDomain.Records, apiv1.ExportRecord{
				Name:     record.Name,
				Type:     record.Type,
				Value:    record.Value,
				TTL:      record.TTL,
				Priority: record.Priority,
			})
		}
		res.Domains = append(res.Domains, apiDomain)
	}
	return res
}
