package models

import (
	"errors"
	"net/http"
	"strings"
	"time"
)

type IntelligenceSource struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"` // asn, city, country
	URL           string    `json:"url"`
	Enabled       bool      `json:"enabled"`
	AutoUpdate    bool      `json:"autoUpdate"`
	UpdateCron    string    `json:"cron"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
	Status        string    `json:"status"` // Ready, Downloading, Error
	ErrorMessage  string    `json:"errorMessage"`
}

func (s *IntelligenceSource) Bind(r *http.Request) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.URL == "" {
		return errors.New("url is required")
	}
	validTypes := map[string]bool{"asn": true, "city": true, "country": true}
	if !validTypes[s.Type] {
		return errors.New("invalid type: must be asn, city or country")
	}
	return nil
}
