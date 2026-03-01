package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.d7z.net/middleware/kv"
)

func SaveLog(ctx context.Context, log *models.AuditLog) error {
	db := common.DB.Child("system", "audit")

	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.Timestamp == "" {
		log.Timestamp = time.Now().Format(time.RFC3339)
	}

	key := fmt.Sprintf("%s-%s", log.Timestamp, log.ID)
	data, err := json.Marshal(log)
	if err != nil {
		return err
	}

	return db.Put(ctx, key, string(data), kv.TTLKeep)
}

func ListLogs(ctx context.Context, page, pageSize int, search string) ([]models.AuditLog, int, error) {
	db := common.DB.Child("system", "audit")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}

	var logs []models.AuditLog
	searchLower := ""
	if search != "" {
		searchLower = strings.ToLower(search)
	}

	// Iterate backwards for descending order (newest first)
	for i := len(items) - 1; i >= 0; i-- {
		var log models.AuditLog
		if err := json.Unmarshal([]byte(items[i].Value), &log); err != nil {
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
	start := page * pageSize
	if start >= total {
		return []models.AuditLog{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return logs[start:end], total, nil
}
