package models

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var siteIDRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

// RuleType 定义
const (
	RuleTypeKeyword uint8 = 0
	RuleTypeRegex   uint8 = 1
	RuleTypeDomain  uint8 = 2
	RuleTypeFull    uint8 = 3
)

// SiteGroup 代表一个域名池的元数据
type SiteGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Checksum    string    `json:"checksum"`   // 数据指纹
	EntryCount  int64     `json:"entryCount"` // 条目总数
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (g *SiteGroup) Bind(r *http.Request) error {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" {
		return errors.New("name is required")
	}
	if g.ID != "" && !siteIDRegex.MatchString(g.ID) {
		return fmt.Errorf("invalid id format: %s", g.ID)
	}
	return nil
}

// SiteExport 代表一个动态导出规则
type SiteExport struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rule        string    `json:"rule"`     // go-expr 表达式
	GroupIDs    []string  `json:"groupIds"` // 依赖的域名池 ID 列表
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (e *SiteExport) Bind(r *http.Request) error {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return errors.New("name is required")
	}
	if e.ID != "" && !siteIDRegex.MatchString(e.ID) {
		return fmt.Errorf("invalid id format: %s", e.ID)
	}
	if e.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(e.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
	return nil
}

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
	Tags    []string `json:"tags"`    // 已废弃，由 NewTags 代替
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

	// 兼容旧 Tags 字段
	if len(req.Tags) > 0 && len(req.NewTags) == 0 {
		req.NewTags = req.Tags
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
