package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"homelab/pkg/models/shared"

	"github.com/robfig/cron/v3"
)

type SyncPolicyMeta struct {
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

type SyncPolicyStatus struct {
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	LastRunAt    time.Time         `json:"lastRunAt"`
	LastStatus   shared.TaskStatus `json:"lastStatus"`
	Progress     float64           `json:"progress"`
	ErrorMessage string            `json:"errorMessage"`
}

type SyncPolicy struct {
	ID              string           `json:"id"`
	Meta            SyncPolicyMeta   `json:"meta"`
	Status          SyncPolicyStatus `json:"status"`
	Generation      int64            `json:"generation"`
	ResourceVersion int64            `json:"resourceVersion"`
}

func (p *SyncPolicy) Bind(_ *http.Request) error {
	p.Meta.Name = strings.TrimSpace(p.Meta.Name)
	p.Meta.SourceURL = strings.TrimSpace(p.Meta.SourceURL)
	p.Meta.Cron = strings.TrimSpace(p.Meta.Cron)
	if p.Meta.Name == "" {
		return errors.New("name is required")
	}
	if p.Meta.SourceURL == "" {
		return errors.New("sourceUrl is required")
	}
	if p.Meta.TargetGroupID == "" {
		return errors.New("targetGroupId is required")
	}
	if p.Meta.Cron == "" {
		return errors.New("cron expression is required")
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(p.Meta.Cron); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	if p.Meta.Format == "" {
		p.Meta.Format = "text"
	}
	if p.Meta.Mode == "" {
		p.Meta.Mode = "overwrite"
	}
	if p.Meta.Config == nil {
		p.Meta.Config = map[string]string{}
	}
	return nil
}

type PoolMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PoolStatus struct {
	Checksum   string    `json:"checksum"`
	EntryCount int64     `json:"entryCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Pool struct {
	ID              string     `json:"id"`
	Meta            PoolMeta   `json:"meta"`
	Status          PoolStatus `json:"status"`
	Generation      int64      `json:"generation"`
	ResourceVersion int64      `json:"resourceVersion"`
}

func (p *Pool) Bind(_ *http.Request) error {
	p.Meta.Name = strings.TrimSpace(p.Meta.Name)
	if p.Meta.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type PoolEntryRequest struct {
	CIDR    string   `json:"cidr"`
	OldTags []string `json:"oldTags,omitempty"`
	NewTags []string `json:"newTags,omitempty"`
}

func (r *PoolEntryRequest) Bind(_ *http.Request) error {
	r.CIDR = strings.TrimSpace(r.CIDR)
	if r.CIDR == "" {
		return errors.New("cidr is required")
	}
	for i, t := range r.NewTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if strings.HasPrefix(t, "_") {
			return fmt.Errorf("tag '%s' is invalid: tags starting with '_' are reserved for internal use", t)
		}
		r.NewTags[i] = t
	}
	for i, t := range r.OldTags {
		r.OldTags[i] = strings.ToLower(strings.TrimSpace(t))
	}
	return nil
}

type ExportMeta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rule        string   `json:"rule"`
	GroupIDs    []string `json:"groupIds"`
}

type ExportStatus struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Export struct {
	ID              string       `json:"id"`
	Meta            ExportMeta   `json:"meta"`
	Status          ExportStatus `json:"status"`
	Generation      int64        `json:"generation"`
	ResourceVersion int64        `json:"resourceVersion"`
}

func (e *Export) Bind(_ *http.Request) error {
	e.Meta.Name = strings.TrimSpace(e.Meta.Name)
	if e.Meta.Name == "" {
		return errors.New("name is required")
	}
	if e.Meta.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(e.Meta.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
	return nil
}

type ExportPreviewRequest struct {
	Rule     string   `json:"rule"`
	GroupIDs []string `json:"groupIds"`
}

func (r *ExportPreviewRequest) Bind(_ *http.Request) error {
	if r.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(r.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
	return nil
}

type PoolEntry struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

type PoolPreviewResponse struct {
	Entries    []PoolEntry `json:"entries"`
	NextCursor string      `json:"nextCursor"`
	Total      int64       `json:"total"`
}

type AnalysisMatch struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

type AnalysisResult struct {
	Matched bool            `json:"matched"`
	Matches []AnalysisMatch `json:"matches"`
}

type HitTestRequest struct {
	IP       string   `json:"ip"`
	GroupIDs []string `json:"groupIds"`
}

type ExportTriggerResponse struct {
	TaskID string `json:"taskId"`
}
