package v1

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"homelab/pkg/models/shared"
)

type SourceMeta struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	URL        string            `json:"url"`
	Enabled    bool              `json:"enabled"`
	AutoUpdate bool              `json:"autoUpdate"`
	UpdateCron string            `json:"cron"`
	Config     map[string]string `json:"config"`
}

type SourceStatus struct {
	LastUpdatedAt time.Time         `json:"lastUpdatedAt"`
	Status        shared.TaskStatus `json:"status"`
	Progress      float64           `json:"progress"`
	ErrorMessage  string            `json:"errorMessage"`
}

type Source struct {
	ID              string       `json:"id"`
	Meta            SourceMeta   `json:"meta"`
	Status          SourceStatus `json:"status"`
	Generation      int64        `json:"generation"`
	ResourceVersion int64        `json:"resourceVersion"`
}

func (s *Source) Bind(_ *http.Request) error {
	s.Meta.Name = strings.TrimSpace(s.Meta.Name)
	s.Meta.URL = strings.TrimSpace(s.Meta.URL)
	if s.Meta.Name == "" {
		return errors.New("name is required")
	}
	if s.Meta.URL == "" {
		return errors.New("url is required")
	}
	switch s.Meta.Type {
	case "asn", "city", "country":
	default:
		return errors.New("invalid type: must be asn, city or country")
	}
	if s.Meta.AutoUpdate && s.Meta.UpdateCron == "" {
		return errors.New("cron expression is required when auto update is enabled")
	}
	return nil
}
