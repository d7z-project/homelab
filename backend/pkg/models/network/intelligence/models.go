package intelligence

import (
	"context"
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
	return nil
}

type IntelligenceSourceV1Status struct {
	LastUpdatedAt time.Time         `json:"lastUpdatedAt"`
	Status        shared.TaskStatus `json:"status"`
	Progress      float64           `json:"progress"`
	ErrorMessage  string            `json:"errorMessage"`
}

type IntelligenceSource = shared.Resource[IntelligenceSourceV1Meta, IntelligenceSourceV1Status]

type MMDBUpdatePayload struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
