package ip_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"net"
	"testing"

	"context"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	sources []models.IntelligenceSource
}

func createASNMMDB(t *testing.T, id string, asn uint32, org string) {
	tree, _ := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-ASN"})
	_, ipnet, _ := net.ParseCIDR("8.8.8.0/24")
	_ = tree.Insert(ipnet, mmdbtype.Map{
		"autonomous_system_number":       mmdbtype.Uint32(asn),
		"autonomous_system_organization": mmdbtype.String(org),
	})
	_ = common.FS.MkdirAll("network/ip/mmdb", 0755)
	file, _ := common.FS.Create("network/ip/mmdb/" + id + ".mmdb")
	_, _ = tree.WriteTo(file)
	file.Close()
}

func createCityMMDB(t *testing.T, id string, cityName string) {
	tree, _ := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-City"})
	_, ipnet, _ := net.ParseCIDR("8.8.8.0/24")
	_ = tree.Insert(ipnet, mmdbtype.Map{
		"city": mmdbtype.Map{
			"names": mmdbtype.Map{
				"zh-CN": mmdbtype.String(cityName),
			},
		},
		"country": mmdbtype.Map{
			"names": mmdbtype.Map{
				"zh-CN": mmdbtype.String("中国"),
			},
			"geoname_id": mmdbtype.Uint32(1814991),
		},
	})
	file, _ := common.FS.Create("network/ip/mmdb/" + id + ".mmdb")
	_, _ = tree.WriteTo(file)
	file.Close()
}

func TestMMDBIncrementalReload(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	common.FS = afero.NewMemMapFs()

	// 1. 准备初始库：ASN1=100(Google), ASN2=300(Cloudflare), City=London
	createASNMMDB(t, "src-asn-1", 100, "Google")
	createASNMMDB(t, "src-asn-2", 300, "Cloudflare")
	createCityMMDB(t, "src-city", "伦敦")

	provider := &mockProvider{
		sources: []models.IntelligenceSource{
			{ID: "src-asn-1", Type: "asn", Enabled: true},
			{ID: "src-asn-2", Type: "asn", Enabled: true},
			{ID: "src-city", Type: "city", Enabled: true},
		},
	}

	manager := ip.NewMMDBManager(provider)
	ctx := context.Background()

	// 验证初始加载
	res, _ := manager.Lookup("8.8.8.8")
	assert.Contains(t, []uint32{100, 300}, uint32(res.ASN))
	assert.Equal(t, "伦敦", res.City)

	// 2. 更新文件
	createASNMMDB(t, "src-asn-1", 200, "Meta")
	createASNMMDB(t, "src-asn-2", 400, "Fastly")
	createCityMMDB(t, "src-city", "巴黎")

	// 3. 仅触发 src-asn-1 (ID) 重载
	err := manager.ReloadID(ctx, "src-asn-1")
	assert.NoError(t, err)

	// 验证
	res, _ = manager.Lookup("8.8.8.8")
	assert.Equal(t, "伦敦", res.City, "City should NOT be reloaded")

	// 4. 触发类型重载 "city"
	err = manager.ReloadType(ctx, "city")
	assert.NoError(t, err)
	res, _ = manager.Lookup("8.8.8.8")
	assert.Equal(t, "巴黎", res.City, "City should now be updated")
}

func (m *mockProvider) ListSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	return m.sources, nil
}

func (m *mockProvider) GetSource(ctx context.Context, id string) (*models.IntelligenceSource, error) {
	for _, s := range m.sources {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}
