package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"net/netip"
	"sync"

	"github.com/oschwald/geoip2-golang/v2"
	"github.com/spf13/afero"
)

const (
	MMDBPathASN     = "network/ip/mmdb/GeoLite2-ASN.mmdb"
	MMDBPathCity    = "network/ip/mmdb/GeoLite2-City.mmdb"
	MMDBPathCountry = "network/ip/mmdb/GeoLite2-Country.mmdb"
)

type MMDBManager struct {
	mu      sync.RWMutex
	asn     *geoip2.Reader
	city    *geoip2.Reader
	country *geoip2.Reader
}

func NewMMDBManager() *MMDBManager {
	m := &MMDBManager{}
	_ = m.Reload() // 尝试初始加载

	// 注册集群事件: 当任意节点更新了 MMDB 文件时，所有节点重新加载
	common.RegisterEventHandler("mmdb_update", func(ctx context.Context, payload string) {
		_ = m.Reload()
	})

	return m
}

// Reload 重新从 VFS 加载 MMDB 文件
func (m *MMDBManager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭旧的 Reader
	if m.asn != nil {
		_ = m.asn.Close()
	}
	if m.city != nil {
		_ = m.city.Close()
	}
	if m.country != nil {
		_ = m.country.Close()
	}

	// 加载新的 Reader
	m.asn = m.loadReader(MMDBPathASN)
	m.city = m.loadReader(MMDBPathCity)
	m.country = m.loadReader(MMDBPathCountry)

	return nil
}

func (m *MMDBManager) loadReader(path string) *geoip2.Reader {
	data, err := afero.ReadFile(common.FS, path)
	if err != nil {
		return nil
	}
	reader, err := geoip2.OpenBytes(data)
	if err != nil {
		return nil
	}
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

	if m.asn != nil {
		if asn, err := m.asn.ASN(ip); err == nil {
			res.ASN = asn.AutonomousSystemNumber
			res.Org = asn.AutonomousSystemOrganization
		}
	}

	if m.city != nil {
		if city, err := m.city.City(ip); err == nil {
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
		}
	} else if m.country != nil {
		if country, err := m.country.Country(ip); err == nil {
			res.Country = country.Country.Names.SimplifiedChinese
			if res.Country == "" {
				res.Country = country.Country.Names.English
			}
		}
	}

	return res, nil
}
