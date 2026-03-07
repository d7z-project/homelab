package site

import (
	"context"
	"homelab/pkg/common"
)

func (s *SitePoolService) lockPool(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "network:site:pool:"+id, 0)
}
