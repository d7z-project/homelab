package site

import (
	"fmt"
	"io"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// 本文件通过 protowire 手动解析和构建 V2Ray GeoSite 二进制数据
// 解决由于官方库导致的版本冲突等问题。

type ParsedGeoSiteEntry struct {
	Type     uint8
	Value    string
	Category string
}

// BuildV2RayGeoSite 构建 V2Ray geosite.dat 格式
// groups 为按 Category 分组的域名列表
func BuildV2RayGeoSite(w io.Writer, groups map[string][]ParsedGeoSiteEntry) error {
	var listData []byte

	for code, domains := range groups {
		var geoSiteData []byte

		// country_code = 1
		geoSiteData = protowire.AppendTag(geoSiteData, 1, protowire.BytesType)
		geoSiteData = protowire.AppendString(geoSiteData, strings.ToUpper(code))

		// repeated domain = 2
		for _, d := range domains {
			var domainData []byte

			// type = 1 (enum)
			domainData = protowire.AppendTag(domainData, 1, protowire.VarintType)
			domainData = protowire.AppendVarint(domainData, uint64(d.Type))

			// value = 2
			domainData = protowire.AppendTag(domainData, 2, protowire.BytesType)
			domainData = protowire.AppendString(domainData, d.Value)

			geoSiteData = protowire.AppendTag(geoSiteData, 2, protowire.BytesType)
			geoSiteData = protowire.AppendBytes(geoSiteData, domainData)
		}

		// entry = 1 in GeoSiteList
		listData = protowire.AppendTag(listData, 1, protowire.BytesType)
		listData = protowire.AppendBytes(listData, geoSiteData)
	}

	_, err := w.Write(listData)
	return err
}

// ParseV2RayGeoSite 解析 geosite.dat 文件
func ParseV2RayGeoSite(data []byte, targetCategory string, importAll bool) ([]ParsedGeoSiteEntry, error) {
	var entries []ParsedGeoSiteEntry

	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("invalid tag")
		}
		data = data[n:]

		if num == 1 && typ == protowire.BytesType {
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, fmt.Errorf("invalid bytes")
			}
			data = data[n:]

			categoryEntries, err := parseSingleGeoSite(v, targetCategory, importAll)
			if err == nil {
				entries = append(entries, categoryEntries...)
			}
		} else {
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				return nil, fmt.Errorf("invalid field value")
			}
			data = data[n:]
		}
	}
	return entries, nil
}

func parseSingleGeoSite(data []byte, targetCategory string, importAll bool) ([]ParsedGeoSiteEntry, error) {
	var category string
	var domainData [][]byte

	temp := data
	for len(temp) > 0 {
		num, typ, n := protowire.ConsumeTag(temp)
		if n < 0 {
			break
		}
		temp = temp[n:]

		switch num {
		case 1: // country_code
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeString(temp)
				if n >= 0 {
					category = v
					temp = temp[n:]
				}
			}
		case 2: // repeated domain
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeBytes(temp)
				if n >= 0 {
					domainData = append(domainData, v)
					temp = temp[n:]
				}
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, temp)
			if n < 0 {
				break
			}
			temp = temp[n:]
		}
	}

	if !importAll && !strings.EqualFold(category, targetCategory) {
		return nil, nil
	}

	var results []ParsedGeoSiteEntry
	for _, b := range domainData {
		dType, dValue, err := parseDomain(b)
		if err == nil {
			results = append(results, ParsedGeoSiteEntry{
				Type:     dType,
				Value:    dValue,
				Category: category,
			})
		}
	}

	return results, nil
}

func parseDomain(data []byte) (uint8, string, error) {
	var dType uint64
	var value string

	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			break
		}
		data = data[n:]

		switch num {
		case 1: // type
			if typ == protowire.VarintType {
				v, n := protowire.ConsumeVarint(data)
				if n >= 0 {
					dType = v
					data = data[n:]
				}
			}
		case 2: // value
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeString(data)
				if n >= 0 {
					value = v
					data = data[n:]
				}
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 {
				break
			}
			data = data[n:]
		}
	}

	if value == "" {
		return 0, "", fmt.Errorf("no value")
	}
	return uint8(dType), value, nil
}
