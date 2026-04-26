package v1

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"time"
)

type DomainMeta struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	PrimaryNS   string `json:"primaryNs"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
}

type DomainStatus struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Domain struct {
	ID              string       `json:"id"`
	Meta            DomainMeta   `json:"meta"`
	Status          DomainStatus `json:"status"`
	Generation      int64        `json:"generation"`
	ResourceVersion int64        `json:"resourceVersion"`
}

func (d *Domain) Bind(_ *http.Request) error {
	d.Meta.Name = strings.TrimSpace(d.Meta.Name)
	d.Meta.Email = strings.TrimSpace(d.Meta.Email)
	d.Meta.PrimaryNS = strings.TrimSpace(d.Meta.PrimaryNS)
	if d.Meta.Name == "" {
		return errors.New("domain name is required")
	}
	if d.Meta.Email != "" {
		if _, err := mail.ParseAddress(d.Meta.Email); err != nil {
			return errors.New("domain email is invalid")
		}
	}
	return nil
}

type RecordMeta struct {
	DomainID string `json:"domainId"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
	Comments string `json:"comments"`
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

type RecordStatus struct {
	SOA *SOAStatus `json:"soa,omitempty"`
}

type Record struct {
	ID              string       `json:"id"`
	Meta            RecordMeta   `json:"meta"`
	Status          RecordStatus `json:"status"`
	Generation      int64        `json:"generation"`
	ResourceVersion int64        `json:"resourceVersion"`
}

func (r *Record) Bind(_ *http.Request) error {
	r.Meta.Name = strings.TrimSpace(r.Meta.Name)
	r.Meta.Type = strings.TrimSpace(r.Meta.Type)
	r.Meta.Value = strings.TrimSpace(r.Meta.Value)
	if r.Meta.Name == "" {
		return errors.New("record name is required")
	}
	if r.Meta.Type == "" {
		return errors.New("record type is required")
	}
	if r.Meta.Value == "" {
		return errors.New("record value is required")
	}
	return nil
}

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

type ExportResponse struct {
	Domains []ExportDomain `json:"domains"`
}
