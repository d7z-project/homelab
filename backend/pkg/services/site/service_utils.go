package site

import (
	"context"
	"crypto/rand"
	"homelab/pkg/common"
)

func generatePolicyID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 10)
	b[0] = '_'
	rb := make([]byte, 9)
	_, _ = rand.Read(rb)
	for i := 0; i < 9; i++ {
		b[i+1] = letters[rb[i]%uint8(len(letters))]
	}
	return string(b)
}

func (s *SitePoolService) lockPool(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "network:site:pool:"+id, 0)
}
