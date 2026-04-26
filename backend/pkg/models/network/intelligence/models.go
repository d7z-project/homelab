package intelligence

import (
	"context"
	"errors"
	"strings"
	"time"

	"homelab/pkg/models/shared"
)

type IntelligenceSourceV1Meta struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	URL        string            `json:"url"`
	Enabled    bool              `json:"enabled"`
	AutoUpdate bool              `json:"autoUpdate"`
	UpdateCron string            `json:"cron"`
	Config     map[string]string `json:"config"`
}

func (m *IntelligenceSourceV1Meta) Validate(_ context.Context) error {
	m.Name = strings.TrimSpace(m.Name)
	m.URL = strings.TrimSpace(m.URL)
	m.UpdateCron = strings.TrimSpace(m.UpdateCron)
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.URL == "" {
		return errors.New("url is required")
	}
	switch m.Type {
	case "asn", "city", "country":
	default:
		return errors.New("invalid type: must be asn, city or country")
	}
	if m.AutoUpdate && m.UpdateCron == "" {
		return errors.New("cron expression is required when auto update is enabled")
	}
	if m.Config == nil {
		m.Config = map[string]string{}
	}
	return nil
}

type IntelligenceSourceV1Status struct {
	LastUpdatedAt  time.Time         `json:"lastUpdatedAt"`
	Status         shared.TaskStatus `json:"status"`
	Progress       float64           `json:"progress"`
	ErrorMessage   string            `json:"errorMessage"`
	QueueTopic     string            `json:"queueTopic,omitempty"`
	QueueMessageID string            `json:"queueMessageId,omitempty"`
	QueuedAt       *time.Time        `json:"queuedAt,omitempty"`
	DispatchedAt   *time.Time        `json:"dispatchedAt,omitempty"`
}

type IntelligenceSource = shared.Resource[IntelligenceSourceV1Meta, IntelligenceSourceV1Status]

type MMDBUpdatePayload struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
