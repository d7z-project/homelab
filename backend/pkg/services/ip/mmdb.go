package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"log"
	"net/netip"
	"sync"

	"github.com/oschwald/geoip2-golang/v2"
	"github.com/spf13/afero"
)

const (
	MMDBDir = "network/ip/mmdb"
)

type SourceProvider func() ([]models.IntelligenceSource, error)

type MMDBManager struct {
	mu       sync.RWMutex
	asn      []*geoip2.Reader
	city     []*geoip2.Reader
	country  []*geoip2.Reader
	provider SourceProvider
}

func NewMMDBManager(provider SourceProvider) *MMDBManager {
	m := &MMDBManager{
		provider: provider,
	}
	_ = m.Reload() // 尝试初始加载

	// 注册集群事件: 当任意节点更新了 MMDB 文件时，所有节点重新加载
	common.RegisterEventHandler(common.EventMMDBUpdate, func(ctx context.Context, payload string) {
		_ = m.Reload()
	})

	return m
}

// Reload 重新从 VFS 加载 MMDB 文件，支持多库回滚机制
func (m *MMDBManager) Reload() error {
	if m.provider == nil {
		return nil
	}

	sources, err := m.provider()
	if err != nil {
		log.Printf("[MMDB] failed to get sources from provider: %v", err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭旧的 Reader
	for _, r := range m.asn {
		_ = r.Close()
	}
	for _, r := range m.city {
		_ = r.Close()
	}
	for _, r := range m.country {
		_ = r.Close()
	}

	m.asn = nil
	m.city = nil
	m.country = nil

	for _, src := range sources {
		if !src.Enabled {
			continue
		}
		path := fmt.Sprintf("%s/%s.mmdb", MMDBDir, src.ID)
		reader := m.loadReader(path)
		if reader == nil {
			continue
		}

		switch src.Type {
		case "asn":
			m.asn = append(m.asn, reader)
		case "city":
			m.city = append(m.city, reader)
		case "country":
			m.country = append(m.country, reader)
		}
	}

	return nil
}

func (m *MMDBManager) loadReader(path string) *geoip2.Reader {
	data, err := afero.ReadFile(common.FS, path)
	if err != nil {
		log.Printf("[MMDB] failed to read file %q: %v", path, err)
		return nil
	}
	reader, err := geoip2.OpenBytes(data)
	if err != nil {
		log.Printf("[MMDB] failed to parse %q: %v", path, err)
		return nil
	}
	log.Printf("[MMDB] loaded %q successfully", path)
	return reader
}

// Lookup 查询 IP 情报
func (m *MMDBManager) Lookup(ipStr string) (*models.IPInfoResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ip: %s", ipStr)
	}

	res := &models.IPInfoResponse{
		IP: ipStr,
	}

	// 检查是否为私有 IP 或回环 IP (RFC 1918, RFC 4193, RFC 1122 等)
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		res.ASN = 0
		res.Org = "Private Network"
		res.Country = "内网"
		res.City = "私有地址"
		res.Location = "0.000000,0.000000"
		return res, nil
	}

	for _, db := range m.asn {
		if asn, err := db.ASN(ip); err == nil && asn.AutonomousSystemNumber > 0 {
			res.ASN = asn.AutonomousSystemNumber
			res.Org = asn.AutonomousSystemOrganization
			break
		}
	}

	for _, db := range m.city {
		if city, err := db.City(ip); err == nil && city.Country.GeoNameID > 0 {
			res.City = city.City.Names.SimplifiedChinese
			if res.City == "" {
				res.City = city.City.Names.English
			}
			res.Country = city.Country.Names.SimplifiedChinese
			if res.Country == "" {
				res.Country = city.Country.Names.English
			}
			if city.Location.Latitude != nil && city.Location.Longitude != nil {
				res.Location = fmt.Sprintf("%f,%f", *city.Location.Latitude, *city.Location.Longitude)
			}
			return res, nil // 优先使用 City 库的结果
		}
	}

	for _, db := range m.country {
		if country, err := db.Country(ip); err == nil && country.Country.GeoNameID > 0 {
			res.Country = country.Country.Names.SimplifiedChinese
			if res.Country == "" {
				res.Country = country.Country.Names.English
			}
			break
		}
	}

	return res, nil
}
