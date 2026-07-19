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

var codexCatalogScheduleLocation = func() *time.Location {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		return location
	}
	return time.FixedZone("UTC-3", -3*60*60)
}()

func nextCodexCatalogSyncDelay(now time.Time) time.Duration {
	localNow := now.In(codexCatalogScheduleLocation)
	nextRun := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 4, 0, 0, 0, codexCatalogScheduleLocation)
	if !nextRun.After(localNow) {
		nextRun = nextRun.AddDate(0, 0, 1)
	}
	return nextRun.Sub(localNow)
}

func runCodexCatalogRefreshCycle() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	result, err := SyncCodexCatalog(ctx, codexCatalogDefaultChannelID)
	cancel()
	if err != nil {
		common.SysLog(fmt.Sprintf("codex catalog sync task failed: %v", err))
	} else {
		common.SysLog(fmt.Sprintf("codex catalog sync task complete: changed=%t discovered=%d promoted=%d validated=%d", result.Changed, len(result.Discovered), len(result.Promoted), result.ValidatedCount))
	}

	pricingCtx, pricingCancel := context.WithTimeout(context.Background(), 30*time.Second)
	pricingResult, pricingErr := RefreshCodexOpenAIReferencePricing(pricingCtx)
	pricingCancel()
	if pricingErr != nil {
		common.SysLog(fmt.Sprintf("codex pricing refresh failed; keeping last known snapshot: %v", pricingErr))
		return
	}
	common.SysLog(fmt.Sprintf("codex pricing reconciliation complete: changed=%t updated=%d registered=%d not_modified=%t", pricingResult.PriceChanged, pricingResult.UpdatedModels, pricingResult.RegisteredModels, pricingResult.NotModified))
}

func StartCodexCatalogSyncTask() {
	codexCatalogTaskOnce.Do(func() {
		codexOpenAIReferencePricingLoadOnce.Do(loadPersistedCodexOpenAIReferencePricing)
		if !common.IsMasterNode {
			return
		}

		gopool.Go(func() {
			runCodexCatalogRefreshCycle()

			for {
				delay := nextCodexCatalogSyncDelay(time.Now())
				common.SysLog(fmt.Sprintf("codex catalog sync task sleeping until next 04:00 America/Sao_Paulo run in %s", delay))
				time.Sleep(delay)

				runCodexCatalogRefreshCycle()
			}
		})
	})
}
