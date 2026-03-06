package ip

import (
	"fmt"
	"net/netip"
	"strings"

	"google.golang.org/protobuf/encoding/protowire"
)

// 本文件通过 protowire 手动解析 V2Ray GeoIP 二进制数据，
// 彻底解决官方库导致的 protobuf 扩展冲突 panic 问题。

type parsedV2RayEntry struct {
	Prefix      netip.Prefix
	CountryCode string
}

// ParseV2RayGeoIP 解析 geoip.dat 文件并提取指定 code 的 CIDR
func ParseV2RayGeoIP(data []byte, targetCode string, importAll bool) ([]parsedV2RayEntry, error) {
	var entries []parsedV2RayEntry
	
	// 外层是 GeoIPList (Field 1: repeated GeoIP)
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
			
			// 解析单个 GeoIP 条目
			categoryEntries, err := parseSingleGeoIP(v, targetCode, importAll)
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

func parseSingleGeoIP(data []byte, targetCode string, importAll bool) ([]parsedV2RayEntry, error) {
	var countryCode string
	var cidrData [][]byte
	
	temp := data
	for len(temp) > 0 {
		num, typ, n := protowire.ConsumeTag(temp)
		if n < 0 { break }
		temp = temp[n:]
		
		switch num {
		case 1: // country_code
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeString(temp)
				if n >= 0 {
					countryCode = v
					temp = temp[n:]
				}
			}
		case 2: // repeated CIDR
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeBytes(temp)
				if n >= 0 {
					cidrData = append(cidrData, v)
					temp = temp[n:]
				}
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, temp)
			if n < 0 { break }
			temp = temp[n:]
		}
	}

	if !importAll && !strings.EqualFold(countryCode, targetCode) {
		return nil, nil
	}

	var results []parsedV2RayEntry
	for _, b := range cidrData {
		prefix, err := parseV2RayCIDR(b)
		if err == nil {
			results = append(results, parsedV2RayEntry{
				Prefix:      prefix,
				CountryCode: countryCode,
			})
		}
	}
	
	return results, nil
}

func parseV2RayCIDR(data []byte) (netip.Prefix, error) {
	var ip []byte
	var prefix uint32
	
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 { break }
		data = data[n:]
		
		switch num {
		case 1: // ip
			if typ == protowire.BytesType {
				v, n := protowire.ConsumeBytes(data)
				if n >= 0 {
					ip = v
					data = data[n:]
				}
			}
		case 2: // prefix
			if typ == protowire.VarintType {
				v, n := protowire.ConsumeVarint(data)
				if n >= 0 {
					prefix = uint32(v)
					data = data[n:]
				}
			}
		default:
			n := protowire.ConsumeFieldValue(num, typ, data)
			if n < 0 { break }
			data = data[n:]
		}
	}

	if len(ip) == 0 {
		return netip.Prefix{}, fmt.Errorf("no ip")
	}
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return netip.Prefix{}, fmt.Errorf("invalid ip")
	}
	return netip.PrefixFrom(addr, int(prefix)), nil
}
