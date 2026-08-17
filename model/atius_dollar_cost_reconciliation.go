package model

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	atiusDollarCostReconciliationV1Option = "AtiusDollarCostReconciliationV1"
	atiusDollarCostQuotaPerUSD            = int64(500_000)
	atiusDollarCostPerMillionDivisor      = int64(1_000_000)
	atiusDollarCostScanBatchSize          = 500
	atiusDollarCostUpdateBatchSize        = 250
)

type atiusDollarCostLogMetadata struct {
	UseDollarCost         bool    `json:"use_dollar_cost"`
	BillingSource         string  `json:"billing_source"`
	BillingMode           string  `json:"billing_mode"`
	InputPrice            float64 `json:"input_price"`
	OutputPrice           float64 `json:"output_price"`
	GroupRatio            float64 `json:"group_ratio"`
	CacheTokens           int64   `json:"cache_tokens"`
	CacheRatio            float64 `json:"cache_ratio"`
	CacheCreationTokens   int64   `json:"cache_creation_tokens"`
	CacheCreationRatio    float64 `json:"cache_creation_ratio"`
	CacheCreationTokens5m int64   `json:"cache_creation_tokens_5m"`
	CacheCreationRatio5m  float64 `json:"cache_creation_ratio_5m"`
	CacheCreationTokens1h int64   `json:"cache_creation_tokens_1h"`
	CacheCreationRatio1h  float64 `json:"cache_creation_ratio_1h"`
	UsageSemantic         string  `json:"usage_semantic"`
	Claude                bool    `json:"claude"`
	WebSearchCallCount    int64   `json:"web_search_call_count"`
	FileSearchCallCount   int64   `json:"file_search_call_count"`
	AudioInputTokenCount  int64   `json:"audio_input_token_count"`
	ImageGenerationCall   bool    `json:"image_generation_call"`
}

type atiusDollarCostCorrection struct {
	LogID     int
	CreatedAt int64
	OldQuota  int64
	NewQuota  int64
	Source    string
}

type atiusWalletTokenKey struct {
	TokenID int
	UserID  int
}

type AtiusDollarCostReconciliationResult struct {
	Version                int   `json:"version"`
	AlreadyApplied         bool  `json:"-"`
	MaxLogID               int   `json:"max_log_id"`
	ScannedLogs            int64 `json:"scanned_logs"`
	DollarCostLogs         int64 `json:"dollar_cost_logs"`
	CorrectedLogs          int64 `json:"corrected_logs"`
	CorrectedQuotaBefore   int64 `json:"corrected_quota_before"`
	CorrectedQuotaAfter    int64 `json:"corrected_quota_after"`
	WalletCorrectedLogs    int64 `json:"wallet_corrected_logs"`
	WalletAdjustmentQuota  int64 `json:"wallet_adjustment_quota"`
	MissingSourceLogs      int64 `json:"missing_source_logs"`
	LogOnlyAdjustmentQuota int64 `json:"log_only_adjustment_quota"`
	HistoricalLogCutoff    int64 `json:"historical_log_cutoff"`
	CompletedAt            int64 `json:"completed_at"`
}

func parseAtiusDollarCostLogMetadata(other string) (atiusDollarCostLogMetadata, bool, error) {
	var metadata atiusDollarCostLogMetadata
	if strings.TrimSpace(other) == "" {
		return metadata, false, nil
	}
	if !strings.Contains(other, `"use_dollar_cost"`) && !strings.Contains(other, `"billing_source"`) {
		return metadata, false, nil
	}
	if err := common.UnmarshalJsonStr(other, &metadata); err != nil {
		return metadata, false, fmt.Errorf("decode billing metadata: %w", err)
	}
	return metadata, true, nil
}

func atiusDollarCostQuota(log Log, metadata atiusDollarCostLogMetadata, includePerMillionDivisor bool) (int64, error) {
	if !metadata.UseDollarCost {
		return int64(log.Quota), nil
	}
	if metadata.InputPrice < 0 || metadata.OutputPrice < 0 || metadata.GroupRatio <= 0 {
		return 0, fmt.Errorf("invalid dollar-cost prices or group ratio for log %d", log.Id)
	}
	if log.PromptTokens < 0 || log.CompletionTokens < 0 || metadata.CacheTokens < 0 || metadata.CacheCreationTokens < 0 || metadata.CacheCreationTokens5m < 0 || metadata.CacheCreationTokens1h < 0 {
		return 0, fmt.Errorf("negative token count for log %d", log.Id)
	}
	if log.PromptTokens+log.CompletionTokens == 0 {
		return 0, nil
	}

	anthropic := metadata.Claude || metadata.UsageSemantic == "anthropic"
	usesClaudeCacheSemantics := anthropic || metadata.CacheCreationTokens5m > 0 || metadata.CacheCreationTokens1h > 0
	nonCachedTokens := int64(log.PromptTokens)
	if !usesClaudeCacheSemantics {
		nonCachedTokens -= metadata.CacheTokens + metadata.CacheCreationTokens
	}
	if nonCachedTokens < 0 {
		return 0, fmt.Errorf("cache tokens exceed prompt tokens for log %d", log.Id)
	}

	cacheCreationCost := decimal.Zero
	if usesClaudeCacheSemantics {
		remaining := metadata.CacheCreationTokens - metadata.CacheCreationTokens5m - metadata.CacheCreationTokens1h
		if remaining < 0 {
			remaining = 0
		}
		cacheCreationCost = decimal.NewFromInt(remaining).Mul(decimal.NewFromFloat(metadata.CacheCreationRatio))
		cacheCreationCost = cacheCreationCost.Add(decimal.NewFromInt(metadata.CacheCreationTokens5m).Mul(decimal.NewFromFloat(metadata.CacheCreationRatio5m)))
		cacheCreationCost = cacheCreationCost.Add(decimal.NewFromInt(metadata.CacheCreationTokens1h).Mul(decimal.NewFromFloat(metadata.CacheCreationRatio1h)))
	} else {
		cacheCreationCost = decimal.NewFromInt(metadata.CacheCreationTokens).Mul(decimal.NewFromFloat(metadata.CacheCreationRatio))
	}

	inputCost := decimal.NewFromInt(nonCachedTokens)
	inputCost = inputCost.Add(decimal.NewFromInt(metadata.CacheTokens).Mul(decimal.NewFromFloat(metadata.CacheRatio)))
	inputCost = inputCost.Add(cacheCreationCost)
	inputCost = inputCost.Mul(decimal.NewFromFloat(metadata.InputPrice))
	outputCost := decimal.NewFromInt(int64(log.CompletionTokens)).Mul(decimal.NewFromFloat(metadata.OutputPrice))
	quota := inputCost.Add(outputCost).
		Mul(decimal.NewFromInt(atiusDollarCostQuotaPerUSD)).
		Mul(decimal.NewFromFloat(metadata.GroupRatio))
	if includePerMillionDivisor {
		quota = quota.Div(decimal.NewFromInt(atiusDollarCostPerMillionDivisor))
	}
	if quota.LessThanOrEqual(decimal.Zero) {
		return 1, nil
	}
	quota = quota.Round(0)
	if quota.GreaterThan(decimal.NewFromInt(math.MaxInt64)) {
		return 0, fmt.Errorf("quota overflow for log %d", log.Id)
	}
	return quota.IntPart(), nil
}

func classifyAtiusDollarCostLog(log Log, metadata atiusDollarCostLogMetadata) (atiusDollarCostCorrection, bool, error) {
	if !metadata.UseDollarCost {
		return atiusDollarCostCorrection{}, false, nil
	}
	if metadata.BillingMode == "tiered_expr" || metadata.WebSearchCallCount > 0 || metadata.FileSearchCallCount > 0 || metadata.AudioInputTokenCount > 0 || metadata.ImageGenerationCall {
		return atiusDollarCostCorrection{}, false, nil
	}
	expected, err := atiusDollarCostQuota(log, metadata, true)
	if err != nil {
		return atiusDollarCostCorrection{}, false, err
	}
	inflated, err := atiusDollarCostQuota(log, metadata, false)
	if err != nil {
		return atiusDollarCostCorrection{}, false, err
	}
	current := int64(log.Quota)
	if current != inflated || current == expected {
		return atiusDollarCostCorrection{}, false, nil
	}
	return atiusDollarCostCorrection{
		LogID:     log.Id,
		CreatedAt: log.CreatedAt,
		OldQuota:  current,
		NewQuota:  expected,
		Source:    metadata.BillingSource,
	}, true, nil
}

func applyAtiusDollarCostCorrections(tx *gorm.DB, corrections []atiusDollarCostCorrection) error {
	for start := 0; start < len(corrections); start += atiusDollarCostUpdateBatchSize {
		end := start + atiusDollarCostUpdateBatchSize
		if end > len(corrections) {
			end = len(corrections)
		}
		batch := corrections[start:end]
		var updateCase strings.Builder
		var guardCase strings.Builder
		updateCase.WriteString("UPDATE logs SET quota = CASE id")
		guardCase.WriteString(" AND quota = CASE id")
		args := make([]any, 0, len(batch)*4+1)
		ids := make([]int, 0, len(batch))
		for _, correction := range batch {
			updateCase.WriteString(" WHEN ? THEN ?")
			args = append(args, correction.LogID, correction.NewQuota)
			ids = append(ids, correction.LogID)
		}
		updateCase.WriteString(" ELSE quota END WHERE id IN ?")
		args = append(args, ids)
		for _, correction := range batch {
			guardCase.WriteString(" WHEN ? THEN ?")
			args = append(args, correction.LogID, correction.OldQuota)
		}
		guardCase.WriteString(" ELSE quota END")
		result := tx.Exec(updateCase.String()+guardCase.String(), args...)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != int64(len(batch)) {
			return fmt.Errorf("log correction guard matched %d of %d rows", result.RowsAffected, len(batch))
		}
	}
	return nil
}

func addAtiusWalletAdjustment(adjustments map[int]int64, id int, adjustment int64) error {
	if adjustment == 0 {
		return nil
	}
	if id <= 0 {
		return errors.New("wallet correction requires positive user, token, and channel identifiers")
	}
	if adjustment < 0 || adjustments[id] > math.MaxInt64-adjustment {
		return fmt.Errorf("wallet adjustment overflow for id %d", id)
	}
	adjustments[id] += adjustment
	return nil
}

func addAtiusResultTotal(total *int64, value int64, field string) error {
	if value < 0 || *total > math.MaxInt64-value {
		return fmt.Errorf("%s overflow", field)
	}
	*total += value
	return nil
}

func reconcileAtiusWalletUsers(tx *gorm.DB, adjustments map[int]int64) error {
	for id, adjustment := range adjustments {
		result := tx.Unscoped().Model(&User{}).
			Where("id = ? AND used_quota >= ? AND quota <= ?", id, adjustment, math.MaxInt64-adjustment).
			Updates(map[string]any{
				"quota":      gorm.Expr("quota + ?", adjustment),
				"used_quota": gorm.Expr("used_quota - ?", adjustment),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("wallet user %d is missing or fails adjustment guards", id)
		}
	}
	return nil
}

func addAtiusWalletTokenAdjustment(adjustments map[atiusWalletTokenKey]int64, tokenID int, userID int, adjustment int64) error {
	if adjustment == 0 {
		return nil
	}
	if tokenID <= 0 || userID <= 0 {
		return errors.New("wallet correction requires positive user, token, and channel identifiers")
	}
	key := atiusWalletTokenKey{TokenID: tokenID, UserID: userID}
	if adjustment < 0 || adjustments[key] > math.MaxInt64-adjustment {
		return fmt.Errorf("wallet token adjustment overflow for token %d and user %d", tokenID, userID)
	}
	adjustments[key] += adjustment
	return nil
}

func reconcileAtiusWalletTokens(tx *gorm.DB, adjustments map[atiusWalletTokenKey]int64) error {
	for key, adjustment := range adjustments {
		result := tx.Unscoped().Model(&Token{}).
			Where("id = ? AND user_id = ? AND used_quota >= ? AND remain_quota <= ?", key.TokenID, key.UserID, adjustment, math.MaxInt64-adjustment).
			Updates(map[string]any{
				"remain_quota": gorm.Expr("remain_quota + ?", adjustment),
				"used_quota":   gorm.Expr("used_quota - ?", adjustment),
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("wallet token %d for user %d is missing or fails adjustment guards", key.TokenID, key.UserID)
		}
	}
	return nil
}

func reconcileAtiusWalletChannels(tx *gorm.DB, adjustments map[int]int64) error {
	for id, adjustment := range adjustments {
		result := tx.Model(&Channel{}).
			Where("id = ? AND used_quota >= ?", id, adjustment).
			Update("used_quota", gorm.Expr("used_quota - ?", adjustment))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("wallet channel %d is missing or fails adjustment guards", id)
		}
	}
	return nil
}

func optionalAtiusExpectedInt64(name string) (*int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return nil, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return &value, nil
}

func validateAtiusDollarCostExpectations(result AtiusDollarCostReconciliationResult) error {
	expectedLogs, err := optionalAtiusExpectedInt64("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS")
	if err != nil {
		return err
	}
	if expectedLogs != nil && result.CorrectedLogs != *expectedLogs {
		return fmt.Errorf("corrected log count %d does not match expected %d", result.CorrectedLogs, *expectedLogs)
	}
	expectedDelta, err := optionalAtiusExpectedInt64("ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_DELTA")
	if err != nil {
		return err
	}
	if result.CorrectedQuotaBefore < result.CorrectedQuotaAfter {
		return errors.New("corrected quota total increased unexpectedly")
	}
	actualDelta := result.CorrectedQuotaBefore - result.CorrectedQuotaAfter
	if expectedDelta != nil && actualDelta != *expectedDelta {
		return fmt.Errorf("corrected quota delta %d does not match expected %d", actualDelta, *expectedDelta)
	}
	for _, expectation := range []struct {
		name   string
		actual int64
	}{
		{"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_LOGS", result.WalletCorrectedLogs},
		{"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_DELTA", result.WalletAdjustmentQuota},
		{"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_LOGS", result.MissingSourceLogs},
		{"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_DELTA", result.LogOnlyAdjustmentQuota},
		{"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_CUTOFF", result.HistoricalLogCutoff},
	} {
		expected, err := optionalAtiusExpectedInt64(expectation.name)
		if err != nil {
			return err
		}
		if expected != nil && expectation.actual != *expected {
			return fmt.Errorf("%s actual %d does not match expected %d", expectation.name, expectation.actual, *expected)
		}
	}
	return nil
}

func loadAtiusDollarCostReconciliationMarker(tx *gorm.DB) (AtiusDollarCostReconciliationResult, bool, error) {
	var option Option
	err := tx.Where(commonKeyCol+" = ?", atiusDollarCostReconciliationV1Option).First(&option).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AtiusDollarCostReconciliationResult{}, false, nil
	}
	if err != nil {
		return AtiusDollarCostReconciliationResult{}, false, err
	}
	var result AtiusDollarCostReconciliationResult
	if err := common.UnmarshalJsonStr(option.Value, &result); err != nil {
		return AtiusDollarCostReconciliationResult{}, false, fmt.Errorf("decode reconciliation marker: %w", err)
	}
	result.AlreadyApplied = true
	return result, true, nil
}

func ReconcileAtiusDollarCostV1(db *gorm.DB) (AtiusDollarCostReconciliationResult, error) {
	if db == nil {
		return AtiusDollarCostReconciliationResult{}, errors.New("database is nil")
	}
	result := AtiusDollarCostReconciliationResult{Version: 1}
	err := db.Transaction(func(tx *gorm.DB) error {
		marker, found, err := loadAtiusDollarCostReconciliationMarker(tx)
		if err != nil {
			return err
		}
		if found {
			result = marker
			return nil
		}

		var watermark struct {
			MaxID int `gorm:"column:max_id"`
		}
		if err := tx.Model(&Log{}).
			Select("COALESCE(MAX(id), 0) AS max_id").
			Where("type = ?", LogTypeConsume).
			Scan(&watermark).Error; err != nil {
			return err
		}
		result.MaxLogID = watermark.MaxID
		users := make(map[int]int64)
		tokens := make(map[atiusWalletTokenKey]int64)
		channels := make(map[int]int64)
		corrections := make([]atiusDollarCostCorrection, 0)
		lastID := 0
		for {
			var logs []Log
			selectFields := "id, user_id, username, type, quota, prompt_tokens, completion_tokens, model_name, created_at, channel_id, token_id, other, " + commonGroupCol
			if err := tx.Select(selectFields).
				Where("type = ? AND id > ? AND id <= ?", LogTypeConsume, lastID, result.MaxLogID).
				Order("id ASC").
				Limit(atiusDollarCostScanBatchSize).
				Find(&logs).Error; err != nil {
				return err
			}
			if len(logs) == 0 {
				break
			}
			for _, log := range logs {
				lastID = log.Id
				result.ScannedLogs++
				metadata, relevant, err := parseAtiusDollarCostLogMetadata(log.Other)
				if err != nil {
					return fmt.Errorf("log %d: %w", log.Id, err)
				}
				if relevant && metadata.UseDollarCost {
					result.DollarCostLogs++
					correction, ok, err := classifyAtiusDollarCostLog(log, metadata)
					if err != nil {
						return err
					}
					if ok {
						corrections = append(corrections, correction)
						if correction.CreatedAt > 0 {
							cutoff := ((correction.CreatedAt / 3600) + 1) * 3600
							if cutoff > result.HistoricalLogCutoff {
								result.HistoricalLogCutoff = cutoff
							}
						}
						result.CorrectedLogs++
						if err := addAtiusResultTotal(&result.CorrectedQuotaBefore, correction.OldQuota, "corrected quota before"); err != nil {
							return err
						}
						if err := addAtiusResultTotal(&result.CorrectedQuotaAfter, correction.NewQuota, "corrected quota after"); err != nil {
							return err
						}
						if correction.OldQuota < correction.NewQuota {
							return fmt.Errorf("log %d correction increases quota", correction.LogID)
						}
						adjustment := correction.OldQuota - correction.NewQuota
						switch strings.TrimSpace(correction.Source) {
						case "wallet":
							result.WalletCorrectedLogs++
							if err := addAtiusResultTotal(&result.WalletAdjustmentQuota, adjustment, "wallet adjustment quota"); err != nil {
								return err
							}
						case "":
							result.MissingSourceLogs++
							if err := addAtiusResultTotal(&result.LogOnlyAdjustmentQuota, adjustment, "log-only adjustment quota"); err != nil {
								return err
							}
						default:
							return fmt.Errorf("log %d has unsupported billing source %q", correction.LogID, correction.Source)
						}
						if strings.TrimSpace(correction.Source) == "wallet" {
							if err := addAtiusWalletAdjustment(users, log.UserId, adjustment); err != nil {
								return err
							}
							if err := addAtiusWalletTokenAdjustment(tokens, log.TokenId, log.UserId, adjustment); err != nil {
								return err
							}
							if err := addAtiusWalletAdjustment(channels, log.ChannelId, adjustment); err != nil {
								return err
							}
						}
					}
				}
			}
		}

		if err := validateAtiusDollarCostExpectations(result); err != nil {
			return err
		}
		currentHour := (time.Now().Unix() / 3600) * 3600
		if result.HistoricalLogCutoff > currentHour {
			return fmt.Errorf("historical log cutoff %d is not closed yet; retry after the current hour", result.HistoricalLogCutoff)
		}
		if err := applyAtiusDollarCostCorrections(tx, corrections); err != nil {
			return err
		}
		if err := reconcileAtiusWalletUsers(tx, users); err != nil {
			return err
		}
		if err := reconcileAtiusWalletTokens(tx, tokens); err != nil {
			return err
		}
		if err := reconcileAtiusWalletChannels(tx, channels); err != nil {
			return err
		}
		result.CompletedAt = time.Now().Unix()
		encoded, err := common.Marshal(result)
		if err != nil {
			return err
		}
		return tx.Create(&Option{Key: atiusDollarCostReconciliationV1Option, Value: string(encoded)}).Error
	})
	return result, err
}

func MigrateAtiusDollarCostV1() error {
	if !common.GetEnvOrDefaultBool("ATIUS_DOLLAR_COST_RECONCILIATION_V1_ENABLED", false) {
		return nil
	}
	for _, name := range []string{
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_LOGS",
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_DELTA",
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_LOGS",
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_WALLET_DELTA",
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_LOGS",
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_MISSING_SOURCE_DELTA",
		"ATIUS_DOLLAR_COST_RECONCILIATION_V1_EXPECTED_CUTOFF",
	} {
		if strings.TrimSpace(os.Getenv(name)) == "" {
			return fmt.Errorf("%s is required when dollar-cost reconciliation is enabled", name)
		}
	}
	if strings.TrimSpace(os.Getenv("LOG_SQL_DSN")) != "" {
		return errors.New("Atius dollar-cost reconciliation cannot run with LOG_SQL_DSN; disable the flag or perform a coordinated external migration")
	}
	result, err := ReconcileAtiusDollarCostV1(DB)
	if err != nil {
		return err
	}
	if !result.AlreadyApplied {
		common.SysLog(fmt.Sprintf("Atius dollar-cost reconciliation completed: logs=%d adjustment=%d", result.CorrectedLogs, result.CorrectedQuotaBefore-result.CorrectedQuotaAfter))
	}
	return nil
}
