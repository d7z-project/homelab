package site

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"homelab/pkg/services/actions"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/net/idna"
)

var siteImportMutex sync.Mutex

type ImportProcessor struct {
	service *SitePoolService
}

func RegisterSiteProcessors(service *SitePoolService) {
	actions.Register(&ImportProcessor{service: service})
}

func (p *ImportProcessor) Manifest() actions.StepManifest {
	return actions.StepManifest{
		ID:          "site/pool/import",
		Name:        "域名池导入",
		Description: "将外部域名规则导入到指定池中。支持清洗与 Punycode 转换。",
		Params: []models.ParamDefinition{
			{Name: "targetPool", Description: "目标域名池 ID", Optional: false, LookupCode: "network/site/pools"},
			{Name: "filePath", Description: "待导入的本地文件路径", Optional: false},
			{Name: "format", Description: "输入文件格式 (text, geosite)", Optional: false},
			{Name: "mode", Description: "导入模式 (append, delete)", Optional: true},
			{Name: "defaultTags", Description: "附加的默认 Tags (逗号分隔)", Optional: true},
			{Name: "category", Description: "geosite 格式的指定类别 (为空表示导入全部并将类别作为 Tag)", Optional: true},
		},
	}
}

func (p *ImportProcessor) Execute(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
	siteImportMutex.Lock()
	defer siteImportMutex.Unlock()

	groupID := inputs["targetPool"]
	filePath := inputs["filePath"]
	format := inputs["format"]
	mode := inputs["mode"]
	if mode == "" {
		mode = "append"
	}

	var defaultTags []string
	if inputs["defaultTags"] != "" {
		defaultTags = strings.Split(inputs["defaultTags"], ",")
	}

	group, err := p.service.GetGroup(ctx.Context, groupID)
	if err != nil {
		return nil, err
	}

	ctx.Logger.Logf("Importing site data to pool %s...", group.Name)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var newEntries []models.SitePoolEntry
	if format == "text" {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			entry := parseTextRule(line)
			entry.Tags = defaultTags

			// Normalize
			val, err := idna.ToASCII(strings.ToLower(entry.Value))
			if err == nil {
				entry.Value = val
			}

			newEntries = append(newEntries, entry)
		}
	} else if format == "geosite" || format == "v2ray-dat" {
		data, err := io.ReadAll(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read geosite file: %w", err)
		}

		targetCategory := inputs["category"]
		importAll := targetCategory == "" || targetCategory == "*" || targetCategory == "all"

		parsedEntries, err := ParseV2RayGeoSite(data, targetCategory, importAll)
		if err != nil {
			return nil, fmt.Errorf("failed to parse geosite: %w", err)
		}

		for _, e := range parsedEntries {
			var tags []string
			tags = append(tags, defaultTags...)
			if importAll {
				tags = append(tags, strings.ToLower(e.Category))
			} else {
				if len(tags) == 0 {
					tags = append(tags, strings.ToLower(e.Category))
				}
			}

			val, err := idna.ToASCII(strings.ToLower(e.Value))
			if err != nil {
				val = strings.ToLower(e.Value)
			}

			newEntries = append(newEntries, models.SitePoolEntry{
				Type:  e.Type,
				Value: val,
				Tags:  tags,
			})
		}
	} else {
		return nil, fmt.Errorf("format %s not yet fully implemented", format)
	}

	// Read existing and merge
	var entries []Entry
	var allTags []string
	tagSet := make(map[string]struct{})

	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	if exists, _ := afero.Exists(common.FS, poolPath); exists {
		pf, _ := common.FS.Open(poolPath)
		reader, _ := NewReader(pf)
		for {
			entry, err := reader.Next()
			if err == io.EOF {
				break
			}

			// Simple deduplication for append mode
			duplicate := false
			for _, ne := range newEntries {
				if ne.Type == entry.Type && ne.Value == entry.Value {
					duplicate = true
					break
				}
			}
			if duplicate && mode == "append" {
				continue
			}

			var tagIndices []uint32
			for _, t := range entry.Tags {
				if _, ok := tagSet[t]; !ok {
					tagSet[t] = struct{}{}
					allTags = append(allTags, t)
				}
				tagIndices = append(tagIndices, uint32(slices.Index(allTags, t)))
			}
			entries = append(entries, Entry{Type: entry.Type, Value: entry.Value, TagIndices: tagIndices})
		}
		pf.Close()
	}

	// Add new ones
	for _, ne := range newEntries {
		var tagIndices []uint32
		for _, t := range ne.Tags {
			if t == "" {
				continue
			}
			if _, ok := tagSet[t]; !ok {
				tagSet[t] = struct{}{}
				allTags = append(allTags, t)
			}
			tagIndices = append(tagIndices, uint32(slices.Index(allTags, t)))
		}
		entries = append(entries, Entry{Type: ne.Type, Value: ne.Value, TagIndices: tagIndices})
	}

	_ = common.FS.MkdirAll(PoolsDir, 0755)
	tempFile := poolPath + ".tmp"
	tf, _ := common.FS.Create(tempFile)
	codec := NewCodec()
	_ = codec.WritePool(tf, allTags, entries)
	tf.Close()
	_ = common.FS.Rename(tempFile, poolPath)

	group.EntryCount = int64(len(entries))
	group.UpdatedAt = time.Now()
	hf := sha256.New()
	content, _ := afero.ReadFile(common.FS, poolPath)
	hf.Write(content)
	group.Checksum = hex.EncodeToString(hf.Sum(nil))
	_ = repo.SaveGroup(ctx.Context, group)

	ctx.Logger.Logf("Import finished. Total entries: %d", group.EntryCount)
	return nil, nil
}

func parseTextRule(line string) models.SitePoolEntry {
	if strings.HasPrefix(line, "full:") {
		return models.SitePoolEntry{Type: 3, Value: strings.TrimPrefix(line, "full:")}
	}
	if strings.HasPrefix(line, "domain:") {
		return models.SitePoolEntry{Type: 2, Value: strings.TrimPrefix(line, "domain:")}
	}
	if strings.HasPrefix(line, "keyword:") {
		return models.SitePoolEntry{Type: 0, Value: strings.TrimPrefix(line, "keyword:")}
	}
	if strings.HasPrefix(line, "regexp:") {
		return models.SitePoolEntry{Type: 1, Value: strings.TrimPrefix(line, "regexp:")}
	}
	// Default to domain
	return models.SitePoolEntry{Type: 2, Value: line}
}
