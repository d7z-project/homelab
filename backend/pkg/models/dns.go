package models

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var domainRegex = regexp.MustCompile(`^([\-a-zA-Z0-9]+([\-a-zA-Z0-9]+)*\.)+[a-zA-Z]{2,}$`)

// Domain 代表一个 DNS 域名
type Domain struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`      // 域名名称 (e.g., example.com)
	Enabled   bool      `json:"enabled"`   // 是否启用
	Comments  string    `json:"comments"`  // 备注说明
	CreatedAt time.Time `json:"createdAt"` // 创建时间
	UpdatedAt time.Time `json:"updatedAt"` // 更新时间
}

func (d *Domain) Bind(r *http.Request) error {
	d.Name = strings.TrimSpace(strings.ToLower(d.Name))
	if d.Name == "" {
		return errors.New("domain name is required")
	}
	if !domainRegex.MatchString(d.Name) {
		return fmt.Errorf("invalid domain format: %s", d.Name)
	}
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
	Enabled  bool   `json:"enabled"`  // 是否启用
	Comments string `json:"comments"` // 备注说明
}

func (rc *Record) Bind(r *http.Request) error {
	rc.Name = strings.TrimSpace(rc.Name)
	rc.Type = strings.ToUpper(strings.TrimSpace(rc.Type))
	rc.Value = strings.TrimSpace(rc.Value)

	if rc.Name == "" {
		return errors.New("record name is required (use @ for root)")
	}
	if rc.Type == "" {
		return errors.New("record type is required")
	}
	if rc.Value == "" {
		return errors.New("record value is required")
	}

	validTypes := map[string]bool{
		"A": true, "AAAA": true, "CNAME": true, "MX": true,
		"TXT": true, "NS": true, "SRV": true, "CAA": true, "SOA": true,
	}
	if !validTypes[rc.Type] {
		return fmt.Errorf("unsupported record type: %s", rc.Type)
	}

	if rc.TTL <= 0 {
		rc.TTL = 600 // Default TTL
	}

	return nil
}

// DnsExportResponse 包含所有 DNS 配置的导出结果
type DnsExportResponse struct {
	Domains []ExportDomain `json:"domains"`
}

// ExportDomain 导出的域名信息，不含 ID 和业务状态
type ExportDomain struct {
	Name    string                            `json:"name"`
	Records map[string]map[string]interface{} `json:"records"`
}

type ExportRecordA struct {
	Address string `json:"address"`
	TTL     int    `json:"ttl"`
}

type ExportRecordAAAA struct {
	Address string `json:"address"`
	TTL     int    `json:"ttl"`
}

type ExportRecordCNAME struct {
	Target string `json:"target"`
	TTL    int    `json:"ttl"`
}

type ExportRecordNS struct {
	Target string `json:"target"`
	TTL    int    `json:"ttl"`
}

type ExportRecordMX struct {
	Host     string `json:"host"`
	Priority int    `json:"priority"`
	TTL      int    `json:"ttl"`
}

type ExportRecordTXT struct {
	Text string `json:"text"`
	TTL  int    `json:"ttl"`
}

type ExportRecordSRV struct {
	Priority int    `json:"priority"`
	Weight   int    `json:"weight"`
	Port     int    `json:"port"`
	Target   string `json:"target"`
	TTL      int    `json:"ttl"`
}

type ExportRecordCAA struct {
	Flags int    `json:"flags"`
	Tag   string `json:"tag"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

type ExportRecordSOA struct {
	Mname   string `json:"mname"`
	Rname   string `json:"rname"`
	Serial  int64  `json:"serial"`
	Refresh int    `json:"refresh"`
	Retry   int    `json:"retry"`
	Expire  int    `json:"expire"`
	Minimum int    `json:"minimum"`
	TTL     int    `json:"ttl"`
}
