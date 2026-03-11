package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.d7z.net/middleware/kv"
)

func getAuditDB(t time.Time) kv.KV {
	db := common.DB
	if db == nil {
		return nil
	}
	year := t.Format("2006")
	month := t.Format("01")
	return db.Child("system", "audit", "data", year, month)
}

func SaveLog(ctx context.Context, log *models.AuditLog) error {
	db := common.DB
	if db == nil {
		return nil
	}
	var t time.Time
	var err error

	if log.Timestamp == "" {
		t = time.Now()
		log.Timestamp = t.Format(time.RFC3339)
	} else {
		t, err = time.Parse(time.RFC3339, log.Timestamp)
		if err != nil {
			t = time.Now()
		}
	}

	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Update index for this year-month
	yearMonth := t.Format("2006-01")
	indexDB := db.Child("system", "audit", "index")
	_, _ = indexDB.PutIfNotExists(ctx, yearMonth, "1", kv.TTLKeep)

	auditDB := getAuditDB(t)
	if auditDB == nil {
		return nil
	}
	key := fmt.Sprintf("%s-%s", log.Timestamp, log.ID)
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}

	return auditDB.Put(ctx, key, string(data), kv.TTLKeep)
}

func ScanLogs(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.AuditLog], error) {
	db := common.DB
	if db == nil {
		return nil, nil
	}
	indexDB := db.Child("system", "audit", "index")
	indexItems, _ := indexDB.List(ctx, "")

	var yearMonths []string
	for _, item := range indexItems {
		yearMonths = append(yearMonths, item.Key)
	}
	sort.Strings(yearMonths)

	// 游标格式: YYYY-MM|InternalKey
	targetYearMonth := ""
	internalCursor := ""
	if cursor != "" {
		parts := strings.SplitN(cursor, "|", 2)
		targetYearMonth = parts[0]
		if len(parts) > 1 {
			internalCursor = parts[1]
		}
	}

	logs := make([]models.AuditLog, 0)
	searchLower := strings.ToLower(search)
	nextCursor := ""
	hasMore := false

	total := int64(0)
	for _, ym := range yearMonths {
		parts := strings.Split(ym, "-")
		if len(parts) == 2 {
			dataDB := db.Child("system", "audit", "data", parts[0], parts[1])
			c, _ := dataDB.Count(ctx)
			total += int64(c)
		}
	}

	// 从最新月份开始逆序查找
	for i := len(yearMonths) - 1; i >= 0; i-- {
		ym := yearMonths[i]
		if targetYearMonth != "" && ym > targetYearMonth {
			continue // 跳过比游标月份更新的数据
		}

		parts := strings.Split(ym, "-")
		if len(parts) != 2 {
			continue
		}

		dataDB := db.Child("system", "audit", "data", parts[0], parts[1])

		// 注意：Audit 存入时 Key 是 "Timestamp-UUID"，List 默认升序。
		// 但审计日志通常需要倒序（最新在前）。
		// 因为 KV.ListCursor 暂不支持逆序，我们仍然需要获取该月数据并内存倒序。
		// 优化：如果该月数据巨大，这仍然有压力。但由于是按月分片，通常可控。
		items, _ := dataDB.List(ctx, "")

		// 内存倒序处理
		for j := len(items) - 1; j >= 0; j-- {
			item := items[j]
			if ym == targetYearMonth && internalCursor != "" && item.Key >= internalCursor {
				continue // 跳过已读数据（Key 是 Timestamp-UUID，倒序时 Key 越小越旧）
			}

			var log models.AuditLog
			if err := json.Unmarshal([]byte(item.Value), &log); err != nil {
				continue
			}

			if search != "" {
				match := strings.Contains(strings.ToLower(log.Subject), searchLower) ||
					strings.Contains(strings.ToLower(log.Action), searchLower) ||
					strings.Contains(strings.ToLower(log.Resource), searchLower) ||
					strings.Contains(strings.ToLower(log.TargetID), searchLower) ||
					strings.Contains(strings.ToLower(log.Message), searchLower)
				if !match {
					continue
				}
			}

			logs = append(logs, log)
			if len(logs) >= limit {
				nextCursor = fmt.Sprintf("%s|%s", ym, item.Key)
				hasMore = true
				break
			}
		}

		if len(logs) >= limit {
			break
		}
		// 如果还没满，重置 targetYearMonth 以便继续扫描下一个（更旧的）月份
		targetYearMonth = ""
	}

	return &models.PaginationResponse[models.AuditLog]{
		Items:      logs,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      total,
	}, nil
}

func CleanupLogs(ctx context.Context, days int) (int, error) {
	db := common.DB
	if db == nil {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	indexDB := db.Child("system", "audit", "index")
	indexItems, _ := indexDB.List(ctx, "")

	deletedCount := 0
	for _, item := range indexItems {
		parts := strings.Split(item.Key, "-")
		if len(parts) == 2 {
			dataDB := db.Child("system", "audit", "data", parts[0], parts[1])
			logs, _ := dataDB.List(ctx, "")

			monthHasRecords := false
			for _, logItem := range logs {
				var log models.AuditLog
				if err := json.Unmarshal([]byte(logItem.Value), &log); err == nil {
					t, err := time.Parse(time.RFC3339, log.Timestamp)
					if err == nil && t.Before(cutoff) {
						dataDB.Delete(ctx, logItem.Key)
						deletedCount++
					} else {
						monthHasRecords = true
					}
				}
			}

			if !monthHasRecords {
				indexDB.Delete(ctx, item.Key)
			}
		}
	}
	return deletedCount, nil
}
