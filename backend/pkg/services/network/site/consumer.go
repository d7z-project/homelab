package site

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/site"

	"gopkg.d7z.net/middleware/queue"
)

const siteSyncTopic = "network.site.sync"

type syncJob struct {
	PolicyID string `json:"policyId"`
}

func (s *SitePoolService) StartSyncConsumer(ctx context.Context) error {
	if s.deps.Queue == nil {
		return errors.New("task queue is not configured")
	}
	go s.consumeSyncQueue(s.deps.WithContext(ctx))
	return nil
}

func (s *SitePoolService) consumeSyncQueue(ctx context.Context) {
	for {
		msg, err := s.deps.Queue.Consume(ctx, siteSyncTopic, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("site sync queue consume failed: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		s.handleSyncMessage(ctx, msg)
	}
}

func (s *SitePoolService) handleSyncMessage(ctx context.Context, msg *queue.Message) {
	var job syncJob
	if err := json.Unmarshal([]byte(msg.Body), &job); err != nil {
		log.Printf("site sync queue decode failed for message %s: %v", msg.ID, err)
		_ = msg.Ack(ctx)
		return
	}

	policy, err := repo.GetSyncPolicy(ctx, job.PolicyID)
	if err != nil {
		if errors.Is(err, common.ErrNotFound) {
			_ = msg.Ack(ctx)
			return
		}
		log.Printf("site sync queue load policy %s failed: %v", job.PolicyID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}
	if policy.Status.LastStatus != shared.TaskStatusPending {
		_ = msg.Ack(ctx)
		return
	}

	dispatchedAt := time.Now()
	policy.Status.DispatchedAt = &dispatchedAt
	if err := repo.SaveSyncPolicy(ctx, policy); err != nil {
		log.Printf("site sync queue persist dispatch %s failed: %v", job.PolicyID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}

	sysCtx := commonauth.WithSystemSA(s.deps.WithContext(context.Background()))
	go func() {
		_ = s.doSync(sysCtx, job.PolicyID)
	}()
	_ = msg.Ack(ctx)
}
