package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	networkcommon "homelab/pkg/models/network/common"
	"log"
	"net/netip"
	"sync"

	intelligencemodel "homelab/pkg/models/network/intelligence"
	intrepo "homelab/pkg/repositories/network/intelligence"
	runtimepkg "homelab/pkg/runtime"

	"github.com/oschwald/geoip2-golang/v2"
	"github.com/spf13/afero"
)

const (
	MMDBDir = "network/ip/mmdb"
)

type MMDBManager struct {
	mu       sync.RWMutex
	initOnce sync.Once
	initErr  error
	fs       afero.Fs
	asn      map[string]*geoip2.Reader
	city     map[string]*geoip2.Reader
	country  map[string]*geoip2.Reader
}

func NewMMDBManager(deps runtimepkg.ModuleDeps) *MMDBManager {
	m := &MMDBManager{
		fs:      deps.FS,
		asn:     make(map[string]*geoip2.Reader),
		city:    make(map[string]*geoip2.Reader),
		country: make(map[string]*geoip2.Reader),
	}

	// 注册集群事件: 当任意节点更新了 MMDB 文件时，增量重新加载
	common.RegisterEventHandler(common.EventMMDBUpdate, func(ctx context.Context, payload intelligencemodel.MMDBUpdatePayload) {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.reloadOne(intelligencemodel.IntelligenceSource{ID: payload.ID, Meta: intelligencemodel.IntelligenceSourceV1Meta{Type: payload.Type}})
	})

	return m
}

func (m *MMDBManager) Init(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.initOnce.Do(func() {
		sources, err := intrepo.ScanAllSources(ctx)
		if err != nil {
			m.initErr = err
			return
		}

		m.mu.Lock()
		defer m.mu.Unlock()
		for _, src := range sources {
			if src.Meta.Enabled {
				m.reloadOne(src)
			}
		}
	})
	return m.initErr
}

func (m *MMDBManager) reloadOne(src intelligencemodel.IntelligenceSource) {
	m.remove(src.ID)
	path := fmt.Sprintf("%s/%s.mmdb", MMDBDir, src.ID)
	reader := m.loadReader(path)
	if reader != nil {
		switch src.Meta.Type {
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
	data, err := afero.ReadFile(m.fs, path)
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
func (m *MMDBManager) Lookup(ipStr string) (*networkcommon.IPInfoResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ip: %s", ipStr)
	}

	res := &networkcommon.IPInfoResponse{
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
