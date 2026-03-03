package orchestration

import (
	"context"
	repo "homelab/pkg/repositories/orchestration"
	"log"
	"os"
	"path/filepath"
	"strings"
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

	// Clean up temp dirs
	tmpDir := os.TempDir()
	matches, err := filepath.Glob(filepath.Join(tmpDir, "task_*"))
	if err != nil {
		log.Printf("Self-healing failed to glob temp dirs: %v", err)
		return
	}

	for _, match := range matches {
		if info, err := os.Stat(match); err == nil && info.IsDir() {
			// Check if it's really our task directory
			if strings.HasPrefix(filepath.Base(match), "task_") {
				_ = os.RemoveAll(match)
				log.Printf("Self-healing: physically removed legacy directory %s", match)
			}
		}
	}
}
