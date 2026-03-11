package ip

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"time"

	"gopkg.d7z.net/middleware/kv"
)

var PoolRepo = common.NewBaseRepository[models.IPPoolV1Meta, models.IPPoolV1Status]("network", "IPPool")
var ExportRepo = common.NewBaseRepository[models.IPExportV1Meta, models.IPExportV1Status]("network", "IPExport")
var SyncPolicyRepo = common.NewBaseRepository[models.IPSyncPolicyV1Meta, models.IPSyncPolicyV1Status]("network", "IPSyncPolicy")

func updateLastModified() {
	now := time.Now().Format(time.RFC3339)
	_ = common.DB.Child("network", "ip").Put(context.Background(), "last_modified", now, kv.TTLKeep)
}

func GetLastModified() time.Time {
	val, err := common.DB.Child("network", "ip").Get(context.Background(), "last_modified")
	if err == nil && val != "" {
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func GetSyncPolicy(ctx context.Context, id string) (*models.IPSyncPolicy, error) {
	return SyncPolicyRepo.Get(ctx, id)
}

func SaveSyncPolicy(ctx context.Context, p *models.IPSyncPolicy) error {
	return SyncPolicyRepo.Cow(ctx, p.ID, func(res *models.IPSyncPolicy) error {
		res.Meta = p.Meta
		res.Status = p.Status
		return nil
	})
}
