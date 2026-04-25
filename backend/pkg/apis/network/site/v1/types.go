package v1

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	networkcommon "homelab/pkg/models/network/common"
	"homelab/pkg/models/shared"

	"github.com/robfig/cron/v3"
)

const (
	RuleTypeKeyword uint8 = 0
	RuleTypeRegex   uint8 = 1
	RuleTypeDomain  uint8 = 2
	RuleTypeFull    uint8 = 3
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

type GroupMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type GroupStatus struct {
	Checksum   string    `json:"checksum"`
	EntryCount int64     `json:"entryCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Group struct {
	ID              string      `json:"id"`
	Meta            GroupMeta   `json:"meta"`
	Status          GroupStatus `json:"status"`
	Generation      int64       `json:"generation"`
	ResourceVersion int64       `json:"resourceVersion"`
}

func (g *Group) Bind(_ *http.Request) error {
	g.Meta.Name = strings.TrimSpace(g.Meta.Name)
	if g.Meta.Name == "" {
		return errors.New("name is required")
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
	Type  uint8    `json:"type"`
	Value string   `json:"value"`
	Tags  []string `json:"tags"`
}

type PoolEntryRequest struct {
	Type    uint8    `json:"type"`
	Value   string   `json:"value"`
	OldTags []string `json:"oldTags"`
	NewTags []string `json:"newTags"`
}

func (r *PoolEntryRequest) Bind(_ *http.Request) error {
	r.Value = strings.TrimSpace(r.Value)
	if r.Value == "" {
		return errors.New("value is required")
	}
	if r.Type > RuleTypeFull {
		return errors.New("invalid rule type")
	}
	for i, t := range r.NewTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if strings.HasPrefix(t, "_") {
			return fmt.Errorf("tag '%s' is invalid: tags starting with '_' are reserved for internal use", t)
		}
		r.NewTags[i] = t
	}
	return nil
}

type PoolPreviewResponse struct {
	Entries    []PoolEntry `json:"entries"`
	NextCursor string      `json:"nextCursor"`
	Total      int64       `json:"total"`
}

type AnalysisResult struct {
	Matched  bool         `json:"matched"`
	RuleType uint8        `json:"ruleType"`
	Pattern  string       `json:"pattern"`
	Tags     []string     `json:"tags"`
	DNS      *DNSAnalysis `json:"dns,omitempty"`
}

type DNSAnalysis struct {
	A     []networkcommon.IPInfoResponse `json:"a,omitempty"`
	AAAA  []networkcommon.IPInfoResponse `json:"aaaa,omitempty"`
	CNAME []string                       `json:"cname,omitempty"`
	NS    []networkcommon.IPInfoResponse `json:"ns,omitempty"`
	SOA   []string                       `json:"soa,omitempty"`
}

type HitTestRequest struct {
	Domain   string   `json:"domain"`
	GroupIDs []string `json:"groupIds"`
}

type ExportTriggerResponse struct {
	TaskID string `json:"taskId"`
}
