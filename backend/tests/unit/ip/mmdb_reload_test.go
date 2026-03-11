package ip_test

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"net"
	"testing"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

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

	// 1. 准备初始库：ASN1=100(Google), ASN2=300(Cloudflare), City=伦敦
	createASNMMDB(t, "src-asn-1", 100, "Google")
	createASNMMDB(t, "src-asn-2", 300, "Cloudflare")
	createCityMMDB(t, "src-city", "伦敦")

	manager := ip.NewMMDBManager([]models.IntelligenceSource{
		{ID: "src-asn-1", Meta: models.IntelligenceSourceV1Meta{Type: "asn", Enabled: true}},
		{ID: "src-asn-2", Meta: models.IntelligenceSourceV1Meta{Type: "asn", Enabled: true}},
		{ID: "src-city", Meta: models.IntelligenceSourceV1Meta{Type: "city", Enabled: true}},
	})
	ctx := context.Background()

	// 验证初始加载
	res, _ := manager.Lookup("8.8.8.8")
	assert.Contains(t, []uint32{100, 300}, uint32(res.ASN))
	assert.Equal(t, "伦敦", res.City)

	// 2. 更新 ASN 文件并通过集群事件触发重载
	createASNMMDB(t, "src-asn-1", 200, "Meta")
	payloadJSON, _ := json.Marshal(models.MMDBUpdatePayload{ID: "src-asn-1", Type: "asn"})
	common.TriggerEvent(ctx, common.EventMMDBUpdate, string(payloadJSON))

	res, _ = manager.Lookup("8.8.8.8")
	assert.Equal(t, "伦敦", res.City, "City should NOT be reloaded")

	// 3. 更新 City 文件并通过集群事件触发重载
	createCityMMDB(t, "src-city", "巴黎")
	payloadJSON, _ = json.Marshal(models.MMDBUpdatePayload{ID: "src-city", Type: "city"})
	common.TriggerEvent(ctx, common.EventMMDBUpdate, string(payloadJSON))

	res, _ = manager.Lookup("8.8.8.8")
	assert.Equal(t, "巴黎", res.City, "City should now be updated via event")

	// 4. 再次更新并验证
	createCityMMDB(t, "src-city", "北京")
	payloadJSON, _ = json.Marshal(models.MMDBUpdatePayload{ID: "src-city", Type: "city"})
	common.TriggerEvent(ctx, common.EventMMDBUpdate, string(payloadJSON))

	res, _ = manager.Lookup("8.8.8.8")
	assert.Equal(t, "北京", res.City, "City should be updated via cluster event")
}
