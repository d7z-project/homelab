package site_test

import (
	"bytes"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"testing"
)

func TestV2RayGeoSite(t *testing.T) {
	groups := map[string][]site.ParsedGeoSiteEntry{
		"CN": {
			{Type: models.RuleTypeDomain, Value: "baidu.com"},
			{Type: models.RuleTypeFull, Value: "www.taobao.com"},
		},
		"GLOBAL": {
			{Type: models.RuleTypeKeyword, Value: "google"},
			{Type: models.RuleTypeRegex, Value: "^.*\\.apple\\.com$"},
		},
	}

	var buf bytes.Buffer
	err := site.BuildV2RayGeoSite(&buf, groups)
	if err != nil {
		t.Fatalf("BuildV2RayGeoSite failed: %v", err)
	}

	entries, err := site.ParseV2RayGeoSite(buf.Bytes(), "CN", false)
	if err != nil {
		t.Fatalf("ParseV2RayGeoSite failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("Expected 2 entries for CN, got %d", len(entries))
	}

	allEntries, err := site.ParseV2RayGeoSite(buf.Bytes(), "", true)
	if err != nil {
		t.Fatalf("ParseV2RayGeoSite (all) failed: %v", err)
	}

	if len(allEntries) != 4 {
		t.Fatalf("Expected 4 entries total, got %d", len(allEntries))
	}

	noEntries, err := site.ParseV2RayGeoSite(buf.Bytes(), "JP", false)
	if err != nil {
		t.Fatalf("ParseV2RayGeoSite failed: %v", err)
	}
	if len(noEntries) != 0 {
		t.Fatalf("Expected 0 entries for JP, got %d", len(noEntries))
	}
}
