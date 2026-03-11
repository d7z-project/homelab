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

var idRegex = regexp.MustCompile(`^[a-z0-9_\-]+$`)

// IPSyncPolicy 代表一个 IP 数据同步策略
type IPSyncPolicyV1Meta struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	SourceURL     string            `json:"sourceUrl"`
	Format        string            `json:"format"` // "text", "geoip"
	Mode          string            `json:"mode"`   // "overwrite", "append"
	Config        map[string]string `json:"config"` // 格式特定的配置
	TargetGroupID string            `json:"targetGroupId"`
	Cron          string            `json:"cron"`
	Enabled       bool              `json:"enabled"`
}

func (p *IPSyncPolicyV1Meta) Validate(ctx context.Context) error {
	return nil
}

type IPSyncPolicyV1Status struct {
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	LastRunAt     time.Time         `json:"lastRunAt"`
	LastStatus    TaskStatus        `json:"lastStatus"` // "success", "failed"
	Progress      float64           `json:"progress"`
	ErrorMessage  string            `json:"errorMessage"`
}

type IPSyncPolicy = Resource[IPSyncPolicyV1Meta, IPSyncPolicyV1Status]

func (p *IPSyncPolicyV1Meta) Bind(r *http.Request) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("name is required")
	}
	p.SourceURL = strings.TrimSpace(p.SourceURL)
	if p.SourceURL == "" {
		return errors.New("sourceUrl is required")
	}
	if p.TargetGroupID == "" {
		return errors.New("targetGroupId is required")
	}
	p.Cron = strings.TrimSpace(p.Cron)
	if p.Cron != "" {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(p.Cron); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	} else {
		return errors.New("cron expression is required")
	}
	if p.Format == "" {
		p.Format = "text"
	}
	if p.Mode == "" {
		p.Mode = "overwrite"
	}
	if p.Config == nil {
		p.Config = make(map[string]string)
	}
	return nil
}

// IPPoolV1Meta IP池的配置数据
type IPPoolV1Meta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (m *IPPoolV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func (m *IPPoolV1Meta) Validate(ctx context.Context) error {
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// IPPoolV1Status IP池的状态数据
type IPPoolV1Status struct {
	Checksum   string    `json:"checksum"`   // 数据指纹，用于缓存失效
	EntryCount int64     `json:"entryCount"` // 池中条目总数
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// IPPool 代表一个 IP 池资源
type IPPool = Resource[IPPoolV1Meta, IPPoolV1Status]

// IPPoolEntryRequest 用于新增、修改、删除 IP/CIDR 标签的请求体
type IPPoolEntryRequest struct {
	CIDR    string   `json:"cidr"`
	OldTags []string `json:"oldTags,omitempty"`
	NewTags []string `json:"newTags,omitempty"`
}

func (req *IPPoolEntryRequest) Bind(r *http.Request) error {
	req.CIDR = strings.TrimSpace(req.CIDR)
	if req.CIDR == "" {
		return errors.New("cidr is required")
	}
	for i, t := range req.NewTags {
		t = strings.ToLower(strings.TrimSpace(t))
		if strings.HasPrefix(t, "_") {
			return fmt.Errorf("tag '%s' is invalid: tags starting with '_' are reserved for internal use", t)
		}
		req.NewTags[i] = t
	}
	for i, t := range req.OldTags {
		req.OldTags[i] = strings.ToLower(strings.TrimSpace(t))
	}
	return nil
}

// IPExport 代表一个动态导出规则
type IPExportV1Meta struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rule        string    `json:"rule"`     // go-expr 表达式
	GroupIDs    []string  `json:"groupIds"` // 依赖的 IP 池 ID 列表
}

func (e *IPExportV1Meta) Validate(ctx context.Context) error {
	return nil
}

type IPExportV1Status struct {
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type IPExport = Resource[IPExportV1Meta, IPExportV1Status]

func (e *IPExportV1Meta) Bind(r *http.Request) error {
	e.Name = strings.TrimSpace(e.Name)
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

// IPExportPreviewRequest 动态导出预览请求
type IPExportPreviewRequest struct {
	Rule     string   `json:"rule"`
	GroupIDs []string `json:"groupIds"`
}

func (req *IPExportPreviewRequest) Bind(r *http.Request) error {
	if req.Rule == "" {
		return errors.New("rule expression is required")
	}
	if len(req.GroupIDs) == 0 {
		return errors.New("at least one source group is required")
	}
	return nil
}

// IPPoolEntry 代表 IP 池中的单条记录（用于预览和导入）
type IPPoolEntry struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

// IPPoolPreviewResponse 游标分页预览响应
type IPPoolPreviewResponse struct {
	Entries    []IPPoolEntry `json:"entries"`
	NextCursor string        `json:"nextCursor"` // 下一个 Byte Offset (作为字符串传递)
	Total      int64         `json:"total"`      // 总条数
}

// IPAnalysisMatch IP 命中条目
type IPAnalysisMatch struct {
	CIDR string   `json:"cidr"` // 命中的具体网段
	Tags []string `json:"tags"` // 命中的 Tags
}

// IPAnalysisResult IP 命中推演结果
type IPAnalysisResult struct {
	Matched bool              `json:"matched"`
	Matches []IPAnalysisMatch `json:"matches"` // 所有命中的网段及其标签
}

// IPHitTestRequest IP 命中推演请求
type IPHitTestRequest struct {
	IP       string   `json:"ip"`
	GroupIDs []string `json:"groupIds"`
}

// IPExportTriggerResponse 导出任务触发响应
type IPExportTriggerResponse struct {
	TaskID string `json:"taskId"`
}

// IPInfoResponse IP 情报查询结果
type IPInfoResponse struct {
	IP       string `json:"ip"`
	Label    string `json:"label,omitempty"` // 附加标识 (如 NS 主机名)
	ASN      uint   `json:"asn"`
	Org      string `json:"org"`
	Country  string `json:"country"`
	City     string `json:"city"`
	Location string `json:"location"` // "lat,lon"
}
