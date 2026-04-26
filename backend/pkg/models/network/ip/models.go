package ip

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	networkcommon "homelab/pkg/models/network/common"
	"homelab/pkg/models/shared"

	"github.com/robfig/cron/v3"
)

type IPSyncPolicyV1Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceURL   string `json:"sourceUrl"`
	Format      string `json:"format"`
	Mode        string `json:"mode"`
	// Config stores non-sensitive parser and import options. Credentials must not be embedded here.
	Config        map[string]string `json:"config"`
	TargetGroupID string            `json:"targetGroupId"`
	Cron          string            `json:"cron"`
	Enabled       bool              `json:"enabled"`
}

func (p *IPSyncPolicyV1Meta) Validate(_ context.Context) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.SourceURL = strings.TrimSpace(p.SourceURL)
	p.TargetGroupID = strings.TrimSpace(p.TargetGroupID)
	p.Cron = strings.TrimSpace(p.Cron)
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.SourceURL == "" {
		return errors.New("sourceUrl is required")
	}
	if p.TargetGroupID == "" {
		return errors.New("targetGroupId is required")
	}
	if p.Cron == "" {
		return errors.New("cron expression is required")
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(p.Cron); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	if p.Format == "" {
		p.Format = "text"
	}
	if p.Mode == "" {
		p.Mode = "merge"
	}
	if p.Config == nil {
		p.Config = map[string]string{}
	}
	for key := range p.Config {
		if networkcommon.LooksSensitiveConfigKey(key) {
			return fmt.Errorf("config key %q is reserved for secret data and must not be stored in policy config", key)
		}
	}
	return nil
}

type IPSyncPolicyV1Status struct {
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	LastRunAt      time.Time         `json:"lastRunAt"`
	LastStatus     shared.TaskStatus `json:"lastStatus"`
	Progress       float64           `json:"progress"`
	ErrorMessage   string            `json:"errorMessage"`
	QueueTopic     string            `json:"queueTopic,omitempty"`
	QueueMessageID string            `json:"queueMessageId,omitempty"`
	QueuedAt       *time.Time        `json:"queuedAt,omitempty"`
	DispatchedAt   *time.Time        `json:"dispatchedAt,omitempty"`
}

type IPSyncPolicy = shared.Resource[IPSyncPolicyV1Meta, IPSyncPolicyV1Status]

type IPPoolV1Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *IPPoolV1Meta) Validate(_ context.Context) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Description = strings.TrimSpace(m.Description)
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
	e.Name = strings.TrimSpace(e.Name)
	e.Description = strings.TrimSpace(e.Description)
	e.Rule = strings.TrimSpace(e.Rule)
	if e.Name == "" {
		return errors.New("name is required")
	}
	if e.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(e.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
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
