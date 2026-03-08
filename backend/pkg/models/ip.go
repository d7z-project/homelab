package models

import (
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
type IPSyncPolicy struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	SourceURL     string            `json:"sourceUrl"`
	Format        string            `json:"format"` // "text", "geoip"
	Mode          string            `json:"mode"`   // "overwrite", "append"
	Config        map[string]string `json:"config"` // 格式特定的配置
	TargetGroupID string            `json:"targetGroupId"`
	Cron          string            `json:"cron"`
	Enabled       bool              `json:"enabled"`
	CreatedAt     time.Time         `json:"createdAt"`
	UpdatedAt     time.Time         `json:"updatedAt"`
	LastRunAt     time.Time         `json:"lastRunAt"`
	LastStatus    TaskStatus        `json:"lastStatus"` // "success", "failed"
	Progress      float64           `json:"progress"`
	ErrorMessage  string            `json:"errorMessage"`
}

func (p *IPSyncPolicy) Bind(r *http.Request) error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("name is required")
	}
	if p.ID != "" && !idRegex.MatchString(p.ID) {
		return fmt.Errorf("invalid id format: %s", p.ID)
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

// IPGroup 代表一个 IP 池的元数据
type IPGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Checksum    string    `json:"checksum"`   // 数据指纹，用于缓存失效
	EntryCount  int64     `json:"entryCount"` // 池中条目总数
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (g *IPGroup) Bind(r *http.Request) error {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" {
		return errors.New("name is required")
	}
	if g.ID != "" && !idRegex.MatchString(g.ID) {
		return fmt.Errorf("invalid id format: %s", g.ID)
	}
	return nil
}

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
type IPExport struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rule        string    `json:"rule"`     // go-expr 表达式
	GroupIDs    []string  `json:"groupIds"` // 依赖的 IP 池 ID 列表
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (e *IPExport) Bind(r *http.Request) error {
	e.Name = strings.TrimSpace(e.Name)
	if e.Name == "" {
		return errors.New("name is required")
	}
	if e.ID != "" && !idRegex.MatchString(e.ID) {
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

// IPAnalysisResult IP 命中推演结果
type IPAnalysisResult struct {
	Matched bool     `json:"matched"`
	CIDR    string   `json:"cidr"` // 命中的具体网段
	Tags    []string `json:"tags"` // 命中的 Tags
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
	ASN      uint   `json:"asn"`
	Org      string `json:"org"`
	Country  string `json:"country"`
	City     string `json:"city"`
	Location string `json:"location"` // "lat,lon"
}
