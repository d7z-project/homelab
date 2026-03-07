package ip_test

import (
	"bytes"
	"homelab/pkg/services/ip"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestV2RayGeoIP(t *testing.T) {
	groups := map[string][]netip.Prefix{
		"CN": {
			netip.MustParsePrefix("1.0.1.0/24"),
			netip.MustParsePrefix("1.0.2.0/24"),
		},
		"US": {
			netip.MustParsePrefix("8.8.8.8/32"),
		},
	}

	// 1. 测试构建
	var buf bytes.Buffer
	err := ip.BuildV2RayGeoIP(&buf, groups)
	assert.NoError(t, err)
	assert.True(t, buf.Len() > 0)

	// 2. 测试解析 - 指定 CN
	entries, err := ip.ParseV2RayGeoIP(buf.Bytes(), "CN", false)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "CN", entries[0].CountryCode)

	prefixes := []string{entries[0].Prefix.String(), entries[1].Prefix.String()}
	assert.Contains(t, prefixes, "1.0.1.0/24")
	assert.Contains(t, prefixes, "1.0.2.0/24")

	// 3. 测试解析 - 全部加载
	allEntries, err := ip.ParseV2RayGeoIP(buf.Bytes(), "", true)
	assert.NoError(t, err)
	assert.Len(t, allEntries, 3)

	// 4. 测试解析 - 不匹配的 code
	noEntries, err := ip.ParseV2RayGeoIP(buf.Bytes(), "JP", false)
	assert.NoError(t, err)
	assert.Len(t, noEntries, 0)
}
