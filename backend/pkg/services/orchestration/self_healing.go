package orchestration

import (
	"context"
	repo "homelab/pkg/repositories/orchestration"
	"log"
	"strings"

	"github.com/spf13/afero"
)

func BootUpSelfHealing() {
	ctx := context.Background()
	instances, err := repo.ListTaskInstances(ctx)
	if err != nil {
		log.Printf("Self-healing failed to list instances: %v", err)
		return
	}

	for _, instance := range instances {
		if instance.Status == "Running" {
			instance.Status = "Failed"
			instance.Error = "System restarted while task was running"
			_ = repo.SaveTaskInstance(ctx, &instance)
			log.Printf("Self-healing: marked task %s as Failed", instance.ID)
		}
	}
// Clean up temp dirs in orch sub-directory (orchFS is already scoped to 'orch')
matches, err := afero.Glob(orchFS, "*")
if err != nil {
	log.Printf("Self-healing failed to glob orchFS temp dirs: %v", err)
	return
}

for _, match := range matches {
	if info, err := orchFS.Stat(match); err == nil && info.IsDir() {
		if strings.HasPrefix(match, TaskPrefix) {
			// Parse instance ID (e.g., task_12345)
			parts := strings.Split(match, "_")
			if len(parts) >= 2 {
				instanceID := parts[0] + "_" + parts[1]
				inst, err := repo.GetTaskInstance(ctx, instanceID)

				// If task not found or NOT running, it's safe to clean up
				if err != nil || (inst != nil && inst.Status != "Running") {
					_ = orchFS.RemoveAll(match)
					log.Printf("Self-healing: removed legacy task directory %s", match)
				} else {
					log.Printf("Self-healing: skipped active task directory %s", match)
				}
			}
		}
	}
}
}
