package site

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

type SiteSyncPolicyV1Meta struct {
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

func (m *SiteSyncPolicyV1Meta) Validate(_ context.Context) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Description = strings.TrimSpace(m.Description)
	m.SourceURL = strings.TrimSpace(m.SourceURL)
	m.TargetGroupID = strings.TrimSpace(m.TargetGroupID)
	m.Cron = strings.TrimSpace(m.Cron)
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.SourceURL == "" {
		return errors.New("sourceUrl is required")
	}
	if m.TargetGroupID == "" {
		return errors.New("targetGroupId is required")
	}
	if m.Cron == "" {
		return errors.New("cron expression is required")
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(m.Cron); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	if m.Format == "" {
		m.Format = "text"
	}
	if m.Mode == "" {
		m.Mode = "overwrite"
	}
	if m.Config == nil {
		m.Config = map[string]string{}
	}
	return nil
}

type SiteSyncPolicyV1Status struct {
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

type SiteSyncPolicy = shared.Resource[SiteSyncPolicyV1Meta, SiteSyncPolicyV1Status]

const (
	RuleTypeKeyword uint8 = 0
	RuleTypeRegex   uint8 = 1
	RuleTypeDomain  uint8 = 2
	RuleTypeFull    uint8 = 3
)

type SiteGroupV1Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *SiteGroupV1Meta) Validate(_ context.Context) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Description = strings.TrimSpace(m.Description)
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type SiteGroupV1Status struct {
	Checksum   string    `json:"checksum"`
	EntryCount int64     `json:"entryCount"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type SiteGroup = shared.Resource[SiteGroupV1Meta, SiteGroupV1Status]

type SiteExportV1Meta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rule        string   `json:"rule"`
	GroupIDs    []string `json:"groupIds"`
}

func (m *SiteExportV1Meta) Validate(_ context.Context) error {
	m.Name = strings.TrimSpace(m.Name)
	m.Description = strings.TrimSpace(m.Description)
	m.Rule = strings.TrimSpace(m.Rule)
	if m.Name == "" {
		return errors.New("name is required")
	}
	if m.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(m.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
	return nil
}

type SiteExportV1Status struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type SiteExport = shared.Resource[SiteExportV1Meta, SiteExportV1Status]

type SiteExportPreviewRequest struct {
	Rule     string   `json:"rule"`
	GroupIDs []string `json:"groupIds"`
}

type SitePoolEntry struct {
	Type  uint8    `json:"type"`
	Value string   `json:"value"`
	Tags  []string `json:"tags"`
}

type SitePoolEntryRequest struct {
	Type    uint8    `json:"type"`
	Value   string   `json:"value"`
	OldTags []string `json:"oldTags"`
	NewTags []string `json:"newTags"`
}

type SitePoolPreviewResponse struct {
	Entries    []SitePoolEntry `json:"entries"`
	NextCursor string          `json:"nextCursor"`
	Total      int64           `json:"total"`
}

type SiteAnalysisResult struct {
	Matched  bool             `json:"matched"`
	RuleType uint8            `json:"ruleType"`
	Pattern  string           `json:"pattern"`
	Tags     []string         `json:"tags"`
	DNS      *SiteDNSAnalysis `json:"dns,omitempty"`
}

type SiteDNSAnalysis struct {
	A     []networkcommon.IPInfoResponse `json:"a,omitempty"`
	AAAA  []networkcommon.IPInfoResponse `json:"aaaa,omitempty"`
	CNAME []string                       `json:"cname,omitempty"`
	NS    []networkcommon.IPInfoResponse `json:"ns,omitempty"`
	SOA   []string                       `json:"soa,omitempty"`
}

type SiteHitTestRequest struct {
	Domain   string   `json:"domain"`
	GroupIDs []string `json:"groupIds"`
}

type SiteExportTriggerResponse struct {
	TaskID string `json:"taskId"`
}
