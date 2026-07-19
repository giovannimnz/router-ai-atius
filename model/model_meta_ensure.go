package model

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// EnsureExactModelMetadata registers routed models without overwriting metadata
// that an administrator already maintains in the models table.
func EnsureExactModelMetadata(modelNames []string, vendorName string, vendorIcon string) (int, error) {
	normalized := make([]string, 0, len(modelNames))
	seen := make(map[string]struct{}, len(modelNames))
	for _, modelName := range modelNames {
		modelName = strings.TrimSpace(modelName)
		if modelName == "" {
			continue
		}
		if _, exists := seen[modelName]; exists {
			continue
		}
		seen[modelName] = struct{}{}
		normalized = append(normalized, modelName)
	}
	if len(normalized) == 0 {
		return 0, nil
	}
	sort.Strings(normalized)

	created := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var vendor Vendor
		err := tx.Where("name = ?", vendorName).First(&vendor).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if err == gorm.ErrRecordNotFound {
			now := common.GetTimestamp()
			vendor = Vendor{
				Name:        vendorName,
				Icon:        vendorIcon,
				Status:      1,
				CreatedTime: now,
				UpdatedTime: now,
			}
			if err := tx.Create(&vendor).Error; err != nil {
				return err
			}
		}

		var existing []Model
		if err := tx.Where("model_name IN ?", normalized).Find(&existing).Error; err != nil {
			return err
		}
		existingNames := make(map[string]struct{}, len(existing))
		for _, item := range existing {
			existingNames[item.ModelName] = struct{}{}
		}

		now := common.GetTimestamp()
		for _, modelName := range normalized {
			if _, exists := existingNames[modelName]; exists {
				continue
			}
			item := Model{
				ModelName:    modelName,
				VendorID:     vendor.Id,
				Status:       1,
				SyncOfficial: 0,
				NameRule:     NameRuleExact,
				CreatedTime:  now,
				UpdatedTime:  now,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			if err := tx.Model(&Model{}).Where("id = ?", item.Id).Updates(map[string]any{
				"status":        1,
				"sync_official": 0,
			}).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	return created, err
}
