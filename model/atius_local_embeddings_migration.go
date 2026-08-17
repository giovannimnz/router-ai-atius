package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

var modelAliasOptionKeys = []string{
	"ModelRatio",
	"ModelPrice",
	"CompletionRatio",
	"InputPrice",
	"OutputPrice",
	"CacheRatio",
	"CreateCacheRatio",
	"ImageRatio",
	"AudioRatio",
	"AudioCompletionRatio",
}

func replaceModelAliasList(value string) (string, bool) {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	changed := false
	for _, part := range parts {
		modelName := strings.TrimSpace(part)
		if modelName == constant.LegacyAtiusLocalRerankerModel {
			modelName = constant.AtiusLocalRerankerModel
			changed = true
		}
		if modelName == "" {
			continue
		}
		if _, ok := seen[modelName]; ok {
			changed = true
			continue
		}
		seen[modelName] = struct{}{}
		result = append(result, modelName)
	}
	normalized := strings.Join(result, ",")
	return normalized, changed || normalized != value
}

func migrateModelAliasOptions(tx *gorm.DB) error {
	var options []Option
	if err := tx.Where(commonKeyCol+" IN ?", modelAliasOptionKeys).Find(&options).Error; err != nil {
		return err
	}
	for _, option := range options {
		var values map[string]any
		if err := common.UnmarshalJsonStr(option.Value, &values); err != nil {
			return err
		}
		legacyValue, exists := values[constant.LegacyAtiusLocalRerankerModel]
		if !exists {
			continue
		}
		if _, alreadyMigrated := values[constant.AtiusLocalRerankerModel]; !alreadyMigrated {
			values[constant.AtiusLocalRerankerModel] = legacyValue
		}
		delete(values, constant.LegacyAtiusLocalRerankerModel)
		encoded, err := common.Marshal(values)
		if err != nil {
			return err
		}
		if err := tx.Model(&Option{}).Where(commonKeyCol+" = ?", option.Key).Update("value", string(encoded)).Error; err != nil {
			return err
		}
	}
	return nil
}

func MigrateAtiusLocalRerankerLogAlias(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Model(&Log{}).
		Where("model_name = ?", constant.LegacyAtiusLocalRerankerModel).
		Update("model_name", constant.AtiusLocalRerankerModel).Error
}

// MigrateAtiusLocalRerankerAlias keeps existing installations routable after
// the public reranker alias changed and unifies dashboard aggregates.
func MigrateAtiusLocalRerankerAlias() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var channels []Channel
		if err := tx.Where("type = ?", constant.ChannelTypeAtiusLocalEmbeddings).Find(&channels).Error; err != nil {
			return err
		}
		for i := range channels {
			channel := &channels[i]
			models, modelsChanged := replaceModelAliasList(channel.Models)
			testModelChanged := channel.TestModel != nil && strings.TrimSpace(*channel.TestModel) == constant.LegacyAtiusLocalRerankerModel
			var legacyAbilityCount int64
			if err := tx.Model(&Ability{}).
				Where("channel_id = ? AND model = ?", channel.Id, constant.LegacyAtiusLocalRerankerModel).
				Count(&legacyAbilityCount).Error; err != nil {
				return err
			}
			if !modelsChanged && !testModelChanged && legacyAbilityCount == 0 {
				continue
			}
			updates := map[string]any{}
			if modelsChanged {
				channel.Models = models
				updates["models"] = models
			}
			if testModelChanged {
				channel.TestModel = common.GetPointer(constant.AtiusLocalRerankerModel)
				updates["test_model"] = constant.AtiusLocalRerankerModel
			}
			if len(updates) > 0 {
				if err := tx.Model(&Channel{}).Where("id = ?", channel.Id).Updates(updates).Error; err != nil {
					return err
				}
			}
			if err := channel.UpdateAbilities(tx); err != nil {
				return err
			}
		}

		var tokens []Token
		if err := tx.Where("model_limits LIKE ?", "%"+constant.LegacyAtiusLocalRerankerModel+"%").Find(&tokens).Error; err != nil {
			return err
		}
		for _, token := range tokens {
			limits, changed := replaceModelAliasList(token.ModelLimits)
			if changed {
				if err := tx.Model(&Token{}).Where("id = ?", token.Id).Update("model_limits", limits).Error; err != nil {
					return err
				}
			}
		}

		var oldModels []Model
		if err := tx.Where("model_name = ?", constant.LegacyAtiusLocalRerankerModel).Find(&oldModels).Error; err != nil {
			return err
		}
		for i := range oldModels {
			var replacementCount int64
			if err := tx.Model(&Model{}).Where("model_name = ?", constant.AtiusLocalRerankerModel).Count(&replacementCount).Error; err != nil {
				return err
			}
			if replacementCount == 0 {
				if err := tx.Model(&Model{}).Where("id = ?", oldModels[i].Id).Update("model_name", constant.AtiusLocalRerankerModel).Error; err != nil {
					return err
				}
			} else if err := tx.Delete(&oldModels[i]).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&QuotaData{}).
			Where("model_name = ?", constant.LegacyAtiusLocalRerankerModel).
			Update("model_name", constant.AtiusLocalRerankerModel).Error; err != nil {
			return err
		}
		return migrateModelAliasOptions(tx)
	})
}
