package model

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	inputPriceOptionKey       = "InputPrice"
	outputPriceOptionKey      = "OutputPrice"
	cacheRatioOptionKey       = "CacheRatio"
	createCacheRatioOptionKey = "CreateCacheRatio"
)

type DollarCostPrice struct {
	Input           float64  `json:"input"`
	Output          float64  `json:"output"`
	CacheReadRatio  *float64 `json:"-"`
	CacheWriteRatio *float64 `json:"-"`
	SyncCacheRead   bool     `json:"-"`
	SyncCacheWrite  bool     `json:"-"`
}

var dollarCostPricePatchMutex sync.Mutex

var errDollarCostPriceConflict = errors.New("dollar-cost price options changed concurrently")

func validateDollarCostPricePatches(patches map[string]*DollarCostPrice) error {
	for modelName, price := range patches {
		if strings.TrimSpace(modelName) == "" {
			return fmt.Errorf("model name cannot be empty")
		}
		if price == nil {
			continue
		}
		if price.Input < 0 || price.Output < 0 ||
			math.IsNaN(price.Input) || math.IsNaN(price.Output) ||
			math.IsInf(price.Input, 0) || math.IsInf(price.Output, 0) {
			return fmt.Errorf("invalid dollar-cost price for model %s", modelName)
		}
		for _, ratio := range []*float64{price.CacheReadRatio, price.CacheWriteRatio} {
			if ratio != nil && (*ratio < 0 || math.IsNaN(*ratio) || math.IsInf(*ratio, 0)) {
				return fmt.Errorf("invalid dollar-cost cache ratio for model %s", modelName)
			}
		}
	}
	return nil
}

func decodeDollarCostPriceMap(raw string) (map[string]float64, error) {
	prices := make(map[string]float64)
	if strings.TrimSpace(raw) == "" {
		return prices, nil
	}
	if err := common.UnmarshalJsonStr(raw, &prices); err != nil {
		return nil, err
	}
	return prices, nil
}

// PatchDollarCostPrices merges only the named models while holding database
// row locks for all canonical price and cache maps. Legacy full-map cache
// updates can use extraOptions and receive the same transaction/CAS guarantee.
func PatchDollarCostPrices(patches map[string]*DollarCostPrice, extraOptions map[string]string) (int, error) {
	if err := validateDollarCostPricePatches(patches); err != nil {
		return 0, err
	}
	if _, exists := extraOptions[inputPriceOptionKey]; exists {
		return 0, fmt.Errorf("%s must be changed through price patches", inputPriceOptionKey)
	}
	if _, exists := extraOptions[outputPriceOptionKey]; exists {
		return 0, fmt.Errorf("%s must be changed through price patches", outputPriceOptionKey)
	}
	cacheRatioReplacement, replaceCacheRatio := extraOptions[cacheRatioOptionKey]
	createCacheRatioReplacement, replaceCreateCacheRatio := extraOptions[createCacheRatioOptionKey]
	syncCacheRead := replaceCacheRatio
	syncCacheWrite := replaceCreateCacheRatio
	for _, price := range patches {
		if price != nil && price.SyncCacheRead {
			syncCacheRead = true
		}
		if price != nil && price.SyncCacheWrite {
			syncCacheWrite = true
		}
	}

	dollarCostPricePatchMutex.Lock()
	defer dollarCostPricePatchMutex.Unlock()

	var inputJSON string
	var outputJSON string
	var cacheRatioJSON string
	var createCacheRatioJSON string
	changedModels := 0
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		changedModels = 0
		err = DB.Transaction(func(tx *gorm.DB) error {
			keys := []string{inputPriceOptionKey, outputPriceOptionKey}
			if syncCacheRead {
				keys = append(keys, cacheRatioOptionKey)
			}
			if syncCacheWrite {
				keys = append(keys, createCacheRatioOptionKey)
			}
			for key := range extraOptions {
				keys = append(keys, key)
			}
			keys = normalizeDollarCostOptionKeys(keys)
			sort.Strings(keys)

			for _, key := range keys {
				if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&Option{Key: key}).Error; err != nil {
					return err
				}
			}

			var rows []Option
			query := tx.Where("key IN ?", keys).Order("key")
			if !common.UsingMainDatabase(common.DatabaseTypeSQLite) {
				query = query.Clauses(clause.Locking{Strength: "UPDATE"})
			}
			if err := query.Find(&rows).Error; err != nil {
				return err
			}
			byKey := make(map[string]*Option, len(rows))
			for i := range rows {
				byKey[rows[i].Key] = &rows[i]
			}
			for _, key := range keys {
				if byKey[key] == nil {
					return fmt.Errorf("canonical price option %s was not loaded", key)
				}
			}

			inputPrices, err := decodeDollarCostPriceMap(byKey[inputPriceOptionKey].Value)
			if err != nil {
				return fmt.Errorf("decode %s: %w", inputPriceOptionKey, err)
			}
			outputPrices, err := decodeDollarCostPriceMap(byKey[outputPriceOptionKey].Value)
			if err != nil {
				return fmt.Errorf("decode %s: %w", outputPriceOptionKey, err)
			}
			cacheRatios := make(map[string]float64)
			if syncCacheRead {
				cacheRatioSource := byKey[cacheRatioOptionKey].Value
				if replaceCacheRatio {
					cacheRatioSource = cacheRatioReplacement
				}
				cacheRatios, err = decodeDollarCostPriceMap(cacheRatioSource)
				if err != nil {
					return fmt.Errorf("decode %s: %w", cacheRatioOptionKey, err)
				}
			}
			createCacheRatios := make(map[string]float64)
			if syncCacheWrite {
				createCacheRatioSource := byKey[createCacheRatioOptionKey].Value
				if replaceCreateCacheRatio {
					createCacheRatioSource = createCacheRatioReplacement
				}
				createCacheRatios, err = decodeDollarCostPriceMap(createCacheRatioSource)
				if err != nil {
					return fmt.Errorf("decode %s: %w", createCacheRatioOptionKey, err)
				}
			}

			for rawModelName, price := range patches {
				modelName := strings.TrimSpace(rawModelName)
				modelChanged := false
				oldInput, hadInput := inputPrices[modelName]
				oldOutput, hadOutput := outputPrices[modelName]
				if price == nil {
					if hadInput || hadOutput {
						delete(inputPrices, modelName)
						delete(outputPrices, modelName)
						modelChanged = true
					}
				} else if !hadInput || !hadOutput || oldInput != price.Input || oldOutput != price.Output {
					inputPrices[modelName] = price.Input
					outputPrices[modelName] = price.Output
					modelChanged = true
				}
				if price != nil && price.SyncCacheRead {
					oldRatio, exists := cacheRatios[modelName]
					if price.CacheReadRatio == nil {
						if exists {
							delete(cacheRatios, modelName)
							modelChanged = true
						}
					} else if !exists || oldRatio != *price.CacheReadRatio {
						cacheRatios[modelName] = *price.CacheReadRatio
						modelChanged = true
					}
				}
				if price != nil && price.SyncCacheWrite {
					oldRatio, exists := createCacheRatios[modelName]
					if price.CacheWriteRatio == nil {
						if exists {
							delete(createCacheRatios, modelName)
							modelChanged = true
						}
					} else if !exists || oldRatio != *price.CacheWriteRatio {
						createCacheRatios[modelName] = *price.CacheWriteRatio
						modelChanged = true
					}
				}
				if modelChanged {
					changedModels++
				}
			}

			inputRaw, err := common.Marshal(inputPrices)
			if err != nil {
				return fmt.Errorf("marshal %s: %w", inputPriceOptionKey, err)
			}
			outputRaw, err := common.Marshal(outputPrices)
			if err != nil {
				return fmt.Errorf("marshal %s: %w", outputPriceOptionKey, err)
			}
			inputJSON = string(inputRaw)
			outputJSON = string(outputRaw)
			if syncCacheRead {
				cacheRatioRaw, err := common.Marshal(cacheRatios)
				if err != nil {
					return fmt.Errorf("marshal %s: %w", cacheRatioOptionKey, err)
				}
				cacheRatioJSON = string(cacheRatioRaw)
			}
			if syncCacheWrite {
				createCacheRatioRaw, err := common.Marshal(createCacheRatios)
				if err != nil {
					return fmt.Errorf("marshal %s: %w", createCacheRatioOptionKey, err)
				}
				createCacheRatioJSON = string(createCacheRatioRaw)
			}

			updateOption := func(key, value string) error {
				if byKey[key].Value == value {
					return nil
				}
				result := tx.Model(&Option{}).
					Where("key = ? AND value = ?", key, byKey[key].Value).
					Update("value", value)
				if result.Error != nil {
					return result.Error
				}
				if result.RowsAffected != 1 {
					return errDollarCostPriceConflict
				}
				return nil
			}
			if err := updateOption(inputPriceOptionKey, inputJSON); err != nil {
				return err
			}
			if err := updateOption(outputPriceOptionKey, outputJSON); err != nil {
				return err
			}
			if syncCacheRead {
				if err := updateOption(cacheRatioOptionKey, cacheRatioJSON); err != nil {
					return err
				}
			}
			if syncCacheWrite {
				if err := updateOption(createCacheRatioOptionKey, createCacheRatioJSON); err != nil {
					return err
				}
			}
			for key, value := range extraOptions {
				if key == cacheRatioOptionKey || key == createCacheRatioOptionKey {
					continue
				}
				if err := updateOption(key, value); err != nil {
					return err
				}
			}
			return nil
		})
		if !isOptionWriteRetryableError(err) {
			break
		}
		if attempt < 2 {
			time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
		}
	}
	if err != nil {
		return 0, err
	}

	var cacheRatioUpdate *string
	if syncCacheRead {
		cacheRatioUpdate = &cacheRatioJSON
	}
	var createCacheRatioUpdate *string
	if syncCacheWrite {
		createCacheRatioUpdate = &createCacheRatioJSON
	}
	if err := ratio_setting.UpdateDollarCostPricingByJSONStrings(
		&inputJSON,
		&outputJSON,
		cacheRatioUpdate,
		createCacheRatioUpdate,
	); err != nil {
		return 0, err
	}
	common.OptionMapRWMutex.Lock()
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	common.OptionMap[inputPriceOptionKey] = inputJSON
	common.OptionMap[outputPriceOptionKey] = outputJSON
	if syncCacheRead {
		common.OptionMap[cacheRatioOptionKey] = cacheRatioJSON
	}
	if syncCacheWrite {
		common.OptionMap[createCacheRatioOptionKey] = createCacheRatioJSON
	}
	common.OptionMapRWMutex.Unlock()
	InvalidatePricingCache()

	for key, value := range extraOptions {
		if key == cacheRatioOptionKey || key == createCacheRatioOptionKey {
			continue
		}
		if err := updateOptionMap(key, value); err != nil {
			return 0, err
		}
	}
	return changedModels, nil
}

func isOptionWriteRetryableError(err error) bool {
	if err == nil || errors.Is(err, errDollarCostPriceConflict) {
		return err != nil
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"database is locked",
		"deadlock",
		"could not serialize",
		"serialization failure",
		"lock wait timeout",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func normalizeDollarCostOptionKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	normalized := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	return normalized
}
