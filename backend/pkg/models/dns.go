package models

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
)

type DomainV1Meta struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func (m *DomainV1Meta) Validate(ctx context.Context) error {
	return nil
}

func (m *DomainV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("domain name is required")
	}
	return nil
}

type DomainV1Status struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Domain = Resource[DomainV1Meta, DomainV1Status]

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

func (m *RecordV1Meta) Validate(ctx context.Context) error {
	return nil
}

func (m *RecordV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("record name is required")
	}
	if m.Type == "" {
		return errors.New("record type is required")
	}
	if m.Value == "" {
		return errors.New("record value is required")
	}
	return nil
}

type RecordV1Status struct {
}

type Record = Resource[RecordV1Meta, RecordV1Status]

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
