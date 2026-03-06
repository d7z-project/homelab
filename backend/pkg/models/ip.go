package models

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var idRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

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

// IPPoolEntryRequest 用于新增、修改、删除 IP/CIDR 条目的请求体
type IPPoolEntryRequest struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

func (req *IPPoolEntryRequest) Bind(r *http.Request) error {
	req.CIDR = strings.TrimSpace(req.CIDR)
	if req.CIDR == "" {
		return errors.New("cidr is required")
	}
	// 简单的清理
	for i, t := range req.Tags {
		req.Tags[i] = strings.ToLower(strings.TrimSpace(t))
	}
	return nil
}

// IPExport 代表一个动态导出规则
type IPExport struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Rule        string    `json:"rule"`      // go-expr 表达式
	GroupIDs    []string  `json:"groupIds"`  // 依赖的 IP 池 ID 列表
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

// IPPoolEntry 代表 IP 池中的单条记录（用于预览和导入）
type IPPoolEntry struct {
	CIDR string   `json:"cidr"`
	Tags []string `json:"tags"`
}

// IPPoolPreviewResponse 游标分页预览响应
type IPPoolPreviewResponse struct {
	Entries    []IPPoolEntry `json:"entries"`
	NextCursor int64         `json:"nextCursor"` // 下一个 Byte Offset
	Total      int64         `json:"total"`      // 总条数
}

// IPAnalysisResult IP 命中推演结果
type IPAnalysisResult struct {
	Matched bool     `json:"matched"`
	CIDR    string   `json:"cidr"` // 命中的具体网段
	Tags    []string `json:"tags"` // 命中的 Tags
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
