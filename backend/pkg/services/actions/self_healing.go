package actions

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"log"
	"strings"

	"github.com/spf13/afero"
)

func BootUpSelfHealing() {
	ctx := context.Background()
	instances, err := repo.ScanAllTaskInstances(ctx)
	if err != nil {
		log.Printf("Self-healing failed to list instances: %v", err)
		return
	}

	for _, instance := range instances {
		if instance.Status == models.TaskStatusRunning || instance.Status == models.TaskStatusPending {
			// 健壮性：仅当该任务对应的分布式锁未被占有时才重置
			lockKey := "action:task:" + instance.ID
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				instance.Status = models.TaskStatusFailed
				instance.Error = "System restarted while task was running or node failure"
				_ = repo.SaveTaskInstance(ctx, &instance)
				log.Printf("Self-healing: marked zombie task %s as Failed", instance.ID)
				release()
			}
		}
	}
	// Clean up temp dirs in actions sub-directory (actionsFS is already scoped to 'orch')
	matches, err := afero.Glob(actionsFS, "*")
	if err != nil {
		log.Printf("Self-healing failed to glob actionsFS temp dirs: %v", err)
		return
	}

	for _, match := range matches {
		if info, err := actionsFS.Stat(match); err == nil && info.IsDir() {
			if strings.HasPrefix(match, TaskPrefix) {
				// Parse instance ID (e.g., task_12345)
				parts := strings.Split(match, "_")
				if len(parts) >= 2 {
					instanceID := parts[0] + "_" + parts[1]
					inst, err := repo.GetTaskInstance(ctx, instanceID)

					// If task not found or NOT running, it's safe to clean up
					if err != nil || (inst != nil && inst.Status != models.TaskStatusRunning) {
						_ = actionsFS.RemoveAll(match)
						log.Printf("Self-healing: removed legacy task directory %s", match)
					} else {
						log.Printf("Self-healing: skipped active task directory %s", match)
					}
				}
			}
		}
	}
}
