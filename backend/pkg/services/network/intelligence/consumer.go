package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/intelligence"

	"gopkg.d7z.net/middleware/queue"
)

const intelligenceSyncTopic = "network.intelligence.sync"

type syncJob struct {
	SourceID string `json:"sourceId"`
}

func (s *IntelligenceService) StartSyncConsumer(ctx context.Context) error {
	if s.deps.Queue == nil {
		return errors.New("task queue is not configured")
	}
	go s.consumeSyncQueue(s.deps.WithContext(ctx))
	return nil
}

func (s *IntelligenceService) consumeSyncQueue(ctx context.Context) {
	for {
		msg, err := s.deps.Queue.Consume(ctx, intelligenceSyncTopic, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("intelligence sync queue consume failed: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		s.handleSyncMessage(ctx, msg)
	}
}

func (s *IntelligenceService) handleSyncMessage(ctx context.Context, msg *queue.Message) {
	var job syncJob
	if err := json.Unmarshal([]byte(msg.Body), &job); err != nil {
		log.Printf("intelligence sync queue decode failed for message %s: %v", msg.ID, err)
		_ = msg.Ack(ctx)
		return
	}

	source, err := repo.GetSource(ctx, job.SourceID)
	if err != nil {
		if errors.Is(err, common.ErrNotFound) {
			_ = msg.Ack(ctx)
			return
		}
		log.Printf("intelligence sync queue load source %s failed: %v", job.SourceID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}
	if source.Status.Status != shared.TaskStatusPending {
		_ = msg.Ack(ctx)
		return
	}

	dispatchedAt := time.Now()
	source.Status.DispatchedAt = &dispatchedAt
	if err := repo.SaveSource(ctx, source); err != nil {
		log.Printf("intelligence sync queue persist dispatch %s failed: %v", job.SourceID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}

	sysCtx := commonauth.WithSystemSA(s.deps.WithContext(context.Background()))
	go s.runDownload(sysCtx, job.SourceID)
	_ = msg.Ack(ctx)
}
