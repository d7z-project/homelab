package unit

import (
	"homelab/pkg/common"
	"homelab/tests"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/robfig/cron/v3"
)

func TestDistributedCronJob(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	c := cron.New()

	var executionCount int32

	lockKey := "test_cron_lock"
	spec := "* * * * *" // Every minute

	entryID, err := common.AddDistributedCronJob(c, spec, lockKey, func() {
		atomic.AddInt32(&executionCount, 1)
	})

	if err != nil {
		t.Fatalf("Failed to add distributed cron job: %v", err)
	}

	entry := c.Entry(entryID)
	if entry.Job == nil {
		t.Fatalf("Job was not scheduled properly")
	}

	// Simulate multiple nodes (or processes) trying to run the job at the exact same scheduled time
	var wg sync.WaitGroup
	numNodes := 5
	for i := 0; i < numNodes; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Call the job run method directly to simulate cron firing across distributed nodes
			entry.Job.Run()
		}()
	}

	wg.Wait()

	// Given our distributed lease mechanism, it should only execute exactly once
	count := atomic.LoadInt32(&executionCount)
	if count != 1 {
		t.Errorf("Expected distributed cron task to execute exactly once, but executed %d times", count)
	}
}
