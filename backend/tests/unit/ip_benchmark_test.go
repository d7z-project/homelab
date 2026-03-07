package unit

import (
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/services/ip"
	"net/netip"
	"testing"

	"github.com/spf13/afero"
)

// BenchmarkIPCodecRead 测试流式读取海量数据的性能和内存（防毛刺）
func BenchmarkIPCodecRead(b *testing.B) {
	// Prepare test data
	codec := ip.NewCodec()
	tags := []string{"test_tag_1", "test_tag_2"}

	// Create a dummy pool with 1 million entries
	entryCount := 1000000
	entries := make([]ip.Entry, 0, entryCount)

	baseIP := netip.MustParseAddr("10.0.0.0")
	baseIPBytes := baseIP.As4()

	for i := 0; i < entryCount; i++ {
		// generate some sequential IPs
		ipBytes := make([]byte, 4)
		copy(ipBytes, baseIPBytes[:])
		ipBytes[2] = byte((i >> 8) & 0xFF)
		ipBytes[3] = byte(i & 0xFF)
		addr := netip.AddrFrom4(*(*[4]byte)(ipBytes))

		entries = append(entries, ip.Entry{
			Prefix:     netip.PrefixFrom(addr, 32),
			TagIndices: []uint32{0},
		})
	}

	testFs := afero.NewMemMapFs()
	common.FS = testFs

	f, _ := testFs.Create("benchmark_pool.bin")
	_ = codec.WritePool(f, tags, entries)
	f.Close()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rf, _ := testFs.Open("benchmark_pool.bin")
		reader, _ := ip.NewReader(rf)
		b.StartTimer()

		count := 0
		for {
			_, _, err := reader.Next()
			if err != nil {
				break
			}
			count++
		}

		b.StopTimer()
		rf.Close()
		if count != entryCount {
			b.Fatalf("expected %d entries, got %d", entryCount, count)
		}
		b.StartTimer()
	}
}

// BenchmarkIPTrieLookup 测试基数树的查询性能
func BenchmarkIPTrieLookup(b *testing.B) {
	trie := ip.NewIPPoolTrie()
	// Insert some prefixes
	trie.Insert(netip.MustParsePrefix("10.0.0.0/8"), []string{"private"})
	trie.Insert(netip.MustParsePrefix("192.168.0.0/16"), []string{"lan"})
	trie.Insert(netip.MustParsePrefix("172.16.0.0/12"), []string{"lan"})

	// Add more to increase tree depth/width
	for i := 0; i < 1000; i++ {
		ipStr := fmt.Sprintf("100.64.%d.0/24", i%256)
		trie.Insert(netip.MustParsePrefix(ipStr), []string{"cgnat"})
	}

	testIP := netip.MustParseAddr("192.168.1.100")
	testIPMiss := netip.MustParseAddr("8.8.8.8")

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("Hit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			trie.Lookup(testIP)
		}
	})

	b.Run("Miss", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			trie.Lookup(testIPMiss)
		}
	})
}
