package common

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/robfig/cron/v3"
)

// AddDistributedCronJob adds a job to the cron with lease-based distributed scheduling.
// TTL is set to 50% of the next interval to avoid missing tasks due to node downtime.
// Returns the scheduled cron.EntryID or an error if scheduling fails.
func AddDistributedCronJob(c *cron.Cron, spec, lockKey string, fn func()) (cron.EntryID, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(spec)
	if err != nil {
		return 0, fmt.Errorf("invalid cron expression: %w", err)
	}

	return c.Schedule(schedule, cron.FuncJob(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		now := time.Now()
		next := schedule.Next(now)
		ttl := next.Sub(now) / 2

		// 确保 TTL 至少为 10 秒，防止短暂任务的租期过短
		if ttl < 10*time.Second {
			ttl = 10 * time.Second
		}
		// 但也不能超过下次执行的时间
		if maxAllowed := next.Sub(now) - 1*time.Second; ttl > maxAllowed && maxAllowed > 0 {
			ttl = maxAllowed
		}

		if infrastructure.db == nil {
			fmt.Printf("cron_lease: db not configured for %s\n", lockKey)
			return
		}
		db := infrastructure.db

		val := fmt.Sprintf("%d", now.UnixNano())
		acquired, err := db.Child("system", "cron", "lease").PutIfNotExists(ctx, lockKey, val, ttl)
		if err == nil && !acquired {
			// 时钟容错: 抢锁失败的节点会进入一个极短的随机等待期 (50-200ms) 后再次尝试
			jitter := time.Duration(rand.Intn(150)+50) * time.Millisecond
			time.Sleep(jitter)
			acquired, err = db.Child("system", "cron", "lease").PutIfNotExists(ctx, lockKey, val, ttl)
		}

		if err != nil {
			fmt.Printf("cron_lease: failed to acquire lock %s: %v\n", lockKey, err)
			return
		}

		if !acquired {
			// 未拿到租约，说明其他节点已开始执行
			return
		}

		// 拿到租约，执行任务
		fn()
	})), nil
}
