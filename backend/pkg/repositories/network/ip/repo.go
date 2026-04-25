package ip

import (
	"context"
	"homelab/pkg/common"
	"time"

	ipmodel "homelab/pkg/models/network/ip"

	"gopkg.d7z.net/middleware/kv"
)

var PoolRepo = common.NewBaseRepository[ipmodel.IPPoolV1Meta, ipmodel.IPPoolV1Status]("network", "IPPool")
var ExportRepo = common.NewBaseRepository[ipmodel.IPExportV1Meta, ipmodel.IPExportV1Status]("network", "IPExport")
var SyncPolicyRepo = common.NewBaseRepository[ipmodel.IPSyncPolicyV1Meta, ipmodel.IPSyncPolicyV1Status]("network", "IPSyncPolicy")

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

func GetSyncPolicy(ctx context.Context, id string) (*ipmodel.IPSyncPolicy, error) {
	return SyncPolicyRepo.Get(ctx, id)
}

func SaveSyncPolicy(ctx context.Context, p *ipmodel.IPSyncPolicy) error {
	return SyncPolicyRepo.Cow(ctx, p.ID, func(res *ipmodel.IPSyncPolicy) error {
		res.Meta = p.Meta
		res.Status = p.Status
		return nil
	})
}
