package intelligence

import (
	"context"
	"homelab/pkg/common"
	intelligencemodel "homelab/pkg/models/network/intelligence"
	"log"
)

func (s *IntelligenceService) addCronJob(src intelligencemodel.IntelligenceSource) {
	id := src.ID
	lockKey := "intelligence_sync_" + src.ID
	entryID, err := common.AddDistributedCronJob(s.cron, src.Meta.UpdateCron, lockKey, func() {
		log.Printf("IntelligenceService: running scheduled update for %s (%s)", src.Meta.Name, src.ID)

		// The original flow did not populate this task in the tasks engine, let's trigger it properly
		// so it goes through the proper locking and state tracking mechanism
		s.SyncSource(context.Background(), id)
	})
	if err != nil {
		log.Printf("IntelligenceService: failed to schedule job for %s: %v", src.Meta.Name, err)
		return
	}
	s.entries[id] = entryID
}

func (s *IntelligenceService) updateCronJob(src intelligencemodel.IntelligenceSource) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove existing if any
	if entryID, ok := s.entries[src.ID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, src.ID)
	}

	// Add new if enabled
	if src.Meta.AutoUpdate && src.Meta.UpdateCron != "" {
		s.addCronJob(src)
	}
}

func (s *IntelligenceService) removeCronJob(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
}
