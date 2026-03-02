package models

import (
	"net/http"
	"time"
)

// Domain 代表一个 DNS 域名
type Domain struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`      // 域名名称 (e.g., example.com)
	Status    string    `json:"status"`    // 状态 (active/inactive)
	Comments  string    `json:"comments"`  // 备注说明
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	UpdatedAt time.Time `json:"updatedAt"` // 更新时间
}

func (d *Domain) Bind(r *http.Request) error {
	return nil
}

// Record 代表一个 DNS 解析记录
type Record struct {
	ID       string `json:"id"`
	DomainID string `json:"domainId"` // 关联的域名 ID
	Name     string `json:"name"`     // 记录名 (e.g., @, www, api)
	Type     string `json:"type"`     // 记录类型 (A, AAAA, CNAME, MX, TXT, NS, SRV, CAA)
	Value    string `json:"value"`    // 记录值 (e.g., 192.168.1.1)
	TTL      int    `json:"ttl"`      // 生存时间 (秒)
	Priority int    `json:"priority"` // 优先级 (仅用于 MX 和 SRV)
	Status   string `json:"status"`   // 状态 (active/inactive)
	Comments string `json:"comments"` // 备注说明
}

func (rc *Record) Bind(r *http.Request) error {
	return nil
}

// DnsExportResponse 包含所有 DNS 配置的导出结果
type DnsExportResponse struct {
	Domains []ExportDomain `json:"domains"`
}

// ExportDomain 导出的域名信息，不含 ID 和业务状态
type ExportDomain struct {
	Name    string         `json:"name"`
	Records []ExportRecord `json:"records"`
}

// ExportRecord 导出的解析记录信息，不含 ID 和业务状态
type ExportRecord struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    string `json:"value"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}
