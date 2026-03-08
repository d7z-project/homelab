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

type SourceProvider interface {
	ListSources(ctx context.Context) ([]models.IntelligenceSource, error)
	GetSource(ctx context.Context, id string) (*models.IntelligenceSource, error)
}

type MMDBManager struct {
	mu       sync.RWMutex
	asn      map[string]*geoip2.Reader
	city     map[string]*geoip2.Reader
	country  map[string]*geoip2.Reader
	provider SourceProvider
}

func NewMMDBManager(provider SourceProvider) *MMDBManager {
	m := &MMDBManager{
		provider: provider,
		asn:      make(map[string]*geoip2.Reader),
		city:     make(map[string]*geoip2.Reader),
		country:  make(map[string]*geoip2.Reader),
	}
	_ = m.ReloadAll(context.Background()) // 首次全量加载

	// 注册集群事件: 当任意节点更新了 MMDB 文件时，特定数据库增量重新加载
	common.RegisterEventHandler(common.EventMMDBUpdate, func(ctx context.Context, payload string) {
		// 判断 payload 是类型还是 ID
		switch payload {
		case "asn", "city", "country":
			_ = m.ReloadType(ctx, payload)
		default:
			_ = m.ReloadID(ctx, payload)
		}
	})

	return m
}

// ReloadAll 全量刷新所有库，并清理已删除的库
func (m *MMDBManager) ReloadAll(ctx context.Context) error {
	if m.provider == nil {
		return nil
	}
	sources, err := m.provider.ListSources(ctx)
	if err != nil {
		log.Printf("[MMDB] failed to list sources: %v", err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 记录当前存在的 ID，用于后续清理
	existingIDs := make(map[string]bool)
	for _, src := range sources {
		existingIDs[src.ID] = true
		if src.Enabled {
			m.reloadOne(src)
		} else {
			m.remove(src.ID)
		}
	}

	// 清理已在数据库中消失的库
	for id, r := range m.asn {
		if !existingIDs[id] {
			_ = r.Close()
			delete(m.asn, id)
		}
	}
	for id, r := range m.city {
		if !existingIDs[id] {
			_ = r.Close()
			delete(m.city, id)
		}
	}
	for id, r := range m.country {
		if !existingIDs[id] {
			_ = r.Close()
			delete(m.country, id)
		}
	}

	return nil
}

// ReloadType 刷新特定类型的所有库 (asn, city, country)
func (m *MMDBManager) ReloadType(ctx context.Context, tp string) error {
	if m.provider == nil {
		return nil
	}
	sources, err := m.provider.ListSources(ctx)
	if err != nil {
		log.Printf("[MMDB] failed to list sources: %v", err)
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, src := range sources {
		if src.Type == tp {
			if src.Enabled {
				m.reloadOne(src)
			} else {
				m.remove(src.ID)
			}
		}
	}
	return nil
}

// ReloadID 精准刷新特定 ID 的库
func (m *MMDBManager) ReloadID(ctx context.Context, id string) error {
	if m.provider == nil {
		return nil
	}
	src, err := m.provider.GetSource(ctx, id)
	if err != nil {
		log.Printf("[MMDB] failed to get source %q: %v", id, err)
		return err
	}
	if src == nil {
		m.mu.Lock()
		m.remove(id)
		m.mu.Unlock()
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if src.Enabled {
		m.reloadOne(*src)
	} else {
		m.remove(src.ID)
	}
	return nil
}

func (m *MMDBManager) reloadOne(src models.IntelligenceSource) {
	path := fmt.Sprintf("%s/%s.mmdb", MMDBDir, src.ID)
	reader := m.loadReader(path)
	if reader != nil {
		m.remove(src.ID) // 先确保移除并关闭旧的
		switch src.Type {
		case "asn":
			m.asn[src.ID] = reader
		case "city":
			m.city[src.ID] = reader
		case "country":
			m.country[src.ID] = reader
		}
	}
}

func (m *MMDBManager) remove(id string) {
	if r, ok := m.asn[id]; ok {
		_ = r.Close()
		delete(m.asn, id)
	}
	if r, ok := m.city[id]; ok {
		_ = r.Close()
		delete(m.city, id)
	}
	if r, ok := m.country[id]; ok {
		_ = r.Close()
		delete(m.country, id)
	}
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

	// 检查是否为私有 IP 或回环 IP
	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		res.ASN = 0
		res.Org = "Private Network"
		res.Country = "内网"
		res.City = "私有地址"
		res.Location = "0.000000,0.000000"
		return res, nil
	}

	// 1. ASN 库查询
	for _, db := range m.asn {
		if asn, err := db.ASN(ip); err == nil && asn.AutonomousSystemNumber > 0 {
			res.ASN = asn.AutonomousSystemNumber
			res.Org = asn.AutonomousSystemOrganization
			break
		}
	}

	// 2. City 库查询 (优先级最高, 提供 Country 信息)
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
			return res, nil
		}
	}

	// 3. Country 库查询 (备选)
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
