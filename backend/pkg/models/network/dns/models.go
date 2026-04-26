package dns

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"homelab/pkg/models/shared"
)

type DomainV1Meta struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PrimaryNS   string `json:"primaryNs"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

func (m *DomainV1Meta) Validate(_ context.Context) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Email = strings.TrimSpace(m.Email)
	m.PrimaryNS = strings.TrimSpace(m.PrimaryNS)
	m.Description = strings.TrimSpace(m.Description)
	if m.Name == "" {
		return errors.New("domain name is required")
	}
	if m.Email != "" {
		if _, err := mail.ParseAddress(m.Email); err != nil {
			return errors.New("domain email is invalid")
		}
	}
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
	m.DomainID = strings.TrimSpace(m.DomainID)
	m.Name = strings.TrimSpace(m.Name)
	m.Type = strings.TrimSpace(m.Type)
	m.Value = strings.TrimSpace(m.Value)
	m.Comments = strings.TrimSpace(m.Comments)
	if m.DomainID == "" {
		return errors.New("domainId is required")
	}
	if m.Name == "" {
		return errors.New("record name is required")
	}
	if m.Type == "" {
		return errors.New("record type is required")
	}
	if m.Type != "SOA" && m.Value == "" {
		return errors.New("record value is required")
	}
	return nil
}

type SOAStatus struct {
	MName   string `json:"mName,omitempty"`
	RName   string `json:"rName,omitempty"`
	Serial  string `json:"serial,omitempty"`
	Refresh int    `json:"refresh,omitempty"`
	Retry   int    `json:"retry,omitempty"`
	Expire  int    `json:"expire,omitempty"`
	Minimum int    `json:"minimum,omitempty"`
}

type RecordV1Status struct {
	SOA *SOAStatus `json:"soa,omitempty"`
}

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
