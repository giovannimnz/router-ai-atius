package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
)

var codexCatalogTaskOnce sync.Once

func nextCodexCatalogSyncDelay(now time.Time) time.Duration {
	location := now.Location()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 4, 0, 0, 0, location)
	if !nextRun.After(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}
	return nextRun.Sub(now)
}

func StartCodexCatalogSyncTask() {
	codexCatalogTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			for {
				delay := nextCodexCatalogSyncDelay(time.Now())
				common.SysLog(fmt.Sprintf("codex catalog sync task sleeping until next 04:00 run in %s", delay))
				time.Sleep(delay)

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				result, err := SyncCodexCatalog(ctx, codexCatalogDefaultChannelID)
				cancel()
				if err != nil {
					common.SysLog(fmt.Sprintf("codex catalog sync task failed: %v", err))
					continue
				}
				common.SysLog(fmt.Sprintf("codex catalog sync task complete: changed=%t discovered=%d promoted=%d validated=%d", result.Changed, len(result.Discovered), len(result.Promoted), result.ValidatedCount))
			}
		})
	})
}
