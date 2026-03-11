package models

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var siteIDRegex = regexp.MustCompile(`^[a-z0-9_\-]+$`)

// SiteSyncPolicyV1Meta 代表一个域名数据同步策略的配置
type SiteSyncPolicyV1Meta struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	SourceURL     string            `json:"sourceUrl"`
	Format        string            `json:"format"` // "text", "geosite"
	Mode          string            `json:"mode"`   // "overwrite", "append"
	Config        map[string]string `json:"config"` // 格式特定的配置
	TargetGroupID string            `json:"targetGroupId"`
	Cron          string            `json:"cron"`
	Enabled       bool              `json:"enabled"`
}

func (m *SiteSyncPolicyV1Meta) Validate(ctx context.Context) error {
	return nil
}

func (m *SiteSyncPolicyV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("name is required")
	}
	m.SourceURL = strings.TrimSpace(m.SourceURL)
	if m.SourceURL == "" {
		return errors.New("sourceUrl is required")
	}
	if m.TargetGroupID == "" {
		return errors.New("targetGroupId is required")
	}
	m.Cron = strings.TrimSpace(m.Cron)
	if m.Cron != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(m.Cron); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	} else {
		return errors.New("cron expression is required")
	}
	if m.Format == "" {
		m.Format = "text"
	}
	if m.Mode == "" {
		m.Mode = "overwrite"
	}
	if m.Config == nil {
		m.Config = make(map[string]string)
	}
	return nil
}

// SiteSyncPolicyV1Status 代表域名同步策略的状态
type SiteSyncPolicyV1Status struct {
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	LastRunAt    time.Time  `json:"lastRunAt"`
	LastStatus   TaskStatus `json:"lastStatus"` // "success", "failed"
	Progress     float64    `json:"progress"`
	ErrorMessage string     `json:"errorMessage"`
}

// SiteSyncPolicy 代表一个域名同步策略资源
type SiteSyncPolicy = Resource[SiteSyncPolicyV1Meta, SiteSyncPolicyV1Status]

// RuleType 定义
const (
	RuleTypeKeyword uint8 = 0
	RuleTypeRegex   uint8 = 1
	RuleTypeDomain  uint8 = 2
	RuleTypeFull    uint8 = 3
)

// SiteGroupV1Meta 代表一个域名池的配置
type SiteGroupV1Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *SiteGroupV1Meta) Validate(ctx context.Context) error {
	return nil
}

func (m *SiteGroupV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// SiteGroupV1Status 代表一个域名池的状态
type SiteGroupV1Status struct {
	Checksum   string    `json:"checksum"`   // 数据指纹
	EntryCount int64     `json:"entryCount"` // 条目总数
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// SiteGroup 代表一个域名池资源
type SiteGroup = Resource[SiteGroupV1Meta, SiteGroupV1Status]

// SiteExportV1Meta 代表一个动态导出规则的配置
type SiteExportV1Meta struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Rule        string   `json:"rule"`     // go-expr 表达式
	GroupIDs    []string `json:"groupIds"` // 依赖的域名池 ID 列表
}

func (m *SiteExportV1Meta) Validate(ctx context.Context) error {
	return nil
}

func (m *SiteExportV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
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

// SiteExportV1Status 代表一个动态导出规则的状态
type SiteExportV1Status struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SiteExport 代表一个动态导出规则资源
type SiteExport = Resource[SiteExportV1Meta, SiteExportV1Status]

// SiteExportPreviewRequest 动态导出预览请求
type SiteExportPreviewRequest struct {
	Rule     string   `json:"rule"`
	GroupIDs []string `json:"groupIds"`
}

func (req *SiteExportPreviewRequest) Bind(r *http.Request) error {
	if req.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(req.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
	return nil
}

// SitePoolEntry 代表 Site 池中的单条记录
type SitePoolEntry struct {
	Type  uint8    `json:"type"` // 0:Keyword, 1:Regex, 2:Domain, 3:Full
	Value string   `json:"value"`
	Tags  []string `json:"tags"`
}

// SitePoolEntryRequest 用于维护条目的请求
type SitePoolEntryRequest struct {
	Type    uint8    `json:"type"`
	Value   string   `json:"value"`
	OldTags []string `json:"oldTags"` // 被替换的标签（用于编辑场景）
	NewTags []string `json:"newTags"` // 新增或更新后的标签
}

func (req *SitePoolEntryRequest) Bind(r *http.Request) error {
	req.Value = strings.TrimSpace(req.Value)
	if req.Value == "" {
		return errors.New("value is required")
	}
	if req.Type > 3 {
		return errors.New("invalid rule type")
	}

	for i, t := range req.NewTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if strings.HasPrefix(t, "_") {
			return fmt.Errorf("tag '%s' is invalid: tags starting with '_' are reserved for internal use", t)
		}
		req.NewTags[i] = t
	}
	return nil
}

// SitePoolPreviewResponse 游标分页预览响应
type SitePoolPreviewResponse struct {
	Entries    []SitePoolEntry `json:"entries"`
	NextCursor string          `json:"nextCursor"` // 下一个 Byte Offset (作为字符串传递)
	Total      int64           `json:"total"`      // 总条数
}

// SiteAnalysisResult 命中推演结果
type SiteAnalysisResult struct {
	Matched  bool     `json:"matched"`
	RuleType uint8    `json:"ruleType"`
	Pattern  string   `json:"pattern"`
	Tags     []string `json:"tags"`

	// DNS Intelligence
	DNS *SiteDNSAnalysis `json:"dns,omitempty"`
}

type SiteDNSAnalysis struct {
	A     []IPInfoResponse `json:"a,omitempty"`
	AAAA  []IPInfoResponse `json:"aaaa,omitempty"`
	CNAME []string         `json:"cname,omitempty"`
	NS    []IPInfoResponse `json:"ns,omitempty"`
	SOA   []string         `json:"soa,omitempty"`
}

// SiteHitTestRequest 域名命中推演请求
type SiteHitTestRequest struct {
	Domain   string   `json:"domain"`
	GroupIDs []string `json:"groupIds"`
}

// SiteExportTriggerResponse 导出任务触发响应
type SiteExportTriggerResponse struct {
	TaskID string `json:"taskId"`
}
