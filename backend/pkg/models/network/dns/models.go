package dns

import (
	"context"
	"time"

	"homelab/pkg/models/shared"
)

type DomainV1Meta struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

func (m *DomainV1Meta) Validate(_ context.Context) error {
	return nil
}

type DomainV1Status struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Domain = shared.Resource[DomainV1Meta, DomainV1Status]

type RecordV1Meta struct {
	DomainID string `json:"domainId"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
	Comments string `json:"comments"`
}

func (m *RecordV1Meta) Validate(_ context.Context) error {
	return nil
}

type RecordV1Status struct{}

type Record = shared.Resource[RecordV1Meta, RecordV1Status]

type ExportRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

type ExportDomain struct {
	Name    string         `json:"name"`
	Records []ExportRecord `json:"records"`
}

type DnsExportResponse struct {
	Domains []ExportDomain `json:"domains"`
}
