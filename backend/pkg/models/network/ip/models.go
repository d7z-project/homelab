package ip

import (
	"context"
	"errors"
	"time"

	"homelab/pkg/models/shared"
)

type IPSyncPolicyV1Meta struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	SourceURL     string            `json:"sourceUrl"`
	Format        string            `json:"format"`
	Mode          string            `json:"mode"`
	Config        map[string]string `json:"config"`
	TargetGroupID string            `json:"targetGroupId"`
	Cron          string            `json:"cron"`
	Enabled       bool              `json:"enabled"`
}

func (p *IPSyncPolicyV1Meta) Validate(_ context.Context) error {
	return nil
}

type IPSyncPolicyV1Status struct {
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	LastRunAt    time.Time         `json:"lastRunAt"`
	LastStatus   shared.TaskStatus `json:"lastStatus"`
	Progress     float64           `json:"progress"`
	ErrorMessage string            `json:"errorMessage"`
}

type IPSyncPolicy = shared.Resource[IPSyncPolicyV1Meta, IPSyncPolicyV1Status]

type IPPoolV1Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *IPPoolV1Meta) Validate(_ context.Context) error {
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type IPPoolV1Status struct {
	Checksum   string    `json:"checksum"`
	EntryCount int64     `json:"entryCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type IPPool = shared.Resource[IPPoolV1Meta, IPPoolV1Status]

type IPPoolEntryRequest struct {
	CIDR    string   `json:"cidr"`
	OldTags []string `json:"oldTags,omitempty"`
	NewTags []string `json:"newTags,omitempty"`
}

type IPExportV1Meta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rule        string   `json:"rule"`
	GroupIDs    []string `json:"groupIds"`
}

func (e *IPExportV1Meta) Validate(_ context.Context) error {
	return nil
}

type IPExportV1Status struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type IPExport = shared.Resource[IPExportV1Meta, IPExportV1Status]

type IPExportPreviewRequest struct {
	Rule     string   `json:"rule"`
	GroupIDs []string `json:"groupIds"`
}

type IPPoolEntry struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

type IPPoolPreviewResponse struct {
	Entries    []IPPoolEntry `json:"entries"`
	NextCursor string        `json:"nextCursor"`
	Total      int64         `json:"total"`
}

type IPAnalysisMatch struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

type IPAnalysisResult struct {
	Matched bool              `json:"matched"`
	Matches []IPAnalysisMatch `json:"matches"`
}

type IPHitTestRequest struct {
	IP       string   `json:"ip"`
	GroupIDs []string `json:"groupIds"`
}

type IPExportTriggerResponse struct {
	TaskID string `json:"taskId"`
}
