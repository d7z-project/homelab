package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
	"time"
	"sort"

	"github.com/google/uuid"
	"gopkg.d7z.net/middleware/kv"
)

func getAuditDB(t time.Time) kv.KV {
	year := t.Format("2006")
	month := t.Format("01")
	return common.DB.Child("system", "audit", "data", year, month)
}

func SaveLog(ctx context.Context, log *models.AuditLog) error {
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
	indexDB := common.DB.Child("system", "audit", "index")
	_ = indexDB.Put(ctx, yearMonth, "1", kv.TTLKeep)

	db := getAuditDB(t)
	key := fmt.Sprintf("%s-%s", log.Timestamp, log.ID)
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}

	return db.Put(ctx, key, string(data), kv.TTLKeep)
}

func ListLogs(ctx context.Context, page, pageSize int, search string) ([]models.AuditLog, int, error) {
	indexDB := common.DB.Child("system", "audit", "index")
	indexItems, _ := indexDB.List(ctx, "")
	
	var yearMonths []string
	for _, item := range indexItems {
		yearMonths = append(yearMonths, item.Key)
	}
	sort.Strings(yearMonths)

	var allItems []kv.Pair
	// Iterate year-months descending
	for i := len(yearMonths) - 1; i >= 0; i-- {
		parts := strings.Split(yearMonths[i], "-")
		if len(parts) == 2 {
			db := common.DB.Child("system", "audit", "data", parts[0], parts[1])
			items, _ := db.List(ctx, "")
			allItems = append(allItems, items...)
		}
	}

	var logs []models.AuditLog
	searchLower := strings.ToLower(search)

	// Iterate backwards through gathered items for newest first
	// Note: inside a single month, items are sorted ascending by timestamp key.
	// We need to reverse them. Since we append months descending, and items within month ascending,
	// we actually need to reverse the items *within* the month before appending, or just collect all 
	// and reverse the whole list if we want strictly descending by time across all.
	// Actually, a simpler way is just to collect all, then sort or reverse.
	// Let's just collect all, and since we need them descending, we will reverse the whole slice.
	// Wait, allItems is constructed by appending month by month (newest month first).
	// Within a month, items are ascending. So we must reverse items WITHIN the month.
	
	var properlySortedItems []kv.Pair
	for i := len(yearMonths) - 1; i >= 0; i-- {
		parts := strings.Split(yearMonths[i], "-")
		if len(parts) == 2 {
			db := common.DB.Child("system", "audit", "data", parts[0], parts[1])
			items, _ := db.List(ctx, "")
			for j := len(items) - 1; j >= 0; j-- {
				properlySortedItems = append(properlySortedItems, items[j])
			}
		}
	}

	for _, item := range properlySortedItems {
		var log models.AuditLog
		if err := json.Unmarshal([]byte(item.Value), &log); err != nil {
			continue
		}

		if search != "" {
			match := strings.Contains(strings.ToLower(log.Subject), searchLower) ||
				strings.Contains(strings.ToLower(log.Action), searchLower) ||
				strings.Contains(strings.ToLower(log.Resource), searchLower) ||
				strings.Contains(strings.ToLower(log.TargetID), searchLower)
			if !match {
				continue
			}
		}
		logs = append(logs, log)
	}

	total := len(logs)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.AuditLog{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return logs[start:end], total, nil
}

func CleanupLogs(ctx context.Context, days int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -days)
	indexDB := common.DB.Child("system", "audit", "index")
	indexItems, _ := indexDB.List(ctx, "")
	
	deletedCount := 0
	for _, item := range indexItems {
		parts := strings.Split(item.Key, "-")
		if len(parts) == 2 {
			db := common.DB.Child("system", "audit", "data", parts[0], parts[1])
			logs, _ := db.List(ctx, "")
			
			monthHasRecords := false
			for _, logItem := range logs {
				var log models.AuditLog
				if err := json.Unmarshal([]byte(logItem.Value), &log); err == nil {
					t, err := time.Parse(time.RFC3339, log.Timestamp)
					if err == nil && t.Before(cutoff) {
						db.Delete(ctx, logItem.Key)
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
