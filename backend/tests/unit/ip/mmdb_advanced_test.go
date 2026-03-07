package ip_test

import (
	"homelab/pkg/common"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"net"
	"testing"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestMMDBAdvanced(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	common.FS = afero.NewMemMapFs()

	// 1. Create a dummy ASN database
	tree, _ := mmdbwriter.New(mmdbwriter.Options{DatabaseType: "GeoLite2-ASN"})
	_, ipnet, _ := net.ParseCIDR("8.8.8.0/24")
	_ = tree.Insert(ipnet, mmdbtype.Map{
		"autonomous_system_number":       mmdbtype.Uint32(15169),
		"autonomous_system_organization": mmdbtype.String("Google LLC"),
	})

	_ = common.FS.MkdirAll("network/ip/mmdb", 0755)
	asnFile, _ := common.FS.Create("network/ip/mmdb/GeoLite2-ASN.mmdb")
	_, _ = tree.WriteTo(asnFile)
	asnFile.Close()

	// 2. Load MMDB
	manager := ip.NewMMDBManager()
	err := manager.Reload() // Should load ASN
	assert.NoError(t, err)

	// 3. Lookup public IP
	res, err := manager.Lookup("8.8.8.8")
	assert.NoError(t, err)
	assert.Equal(t, uint32(15169), uint32(res.ASN))
	assert.Equal(t, "Google LLC", res.Org)
}
