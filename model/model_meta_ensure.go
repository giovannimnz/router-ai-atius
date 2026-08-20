package model

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
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

type exactModelMetadataSeed struct {
	ModelName    string
	Description  string
	Tags         string
	Endpoints    string
	SyncOfficial int
}

const (
	atiusLocalVendorName       = "Atius Local"
	atiusLocalLegacyVendorName = "Atius"
	atiusLocalIcon             = "AtiusLocal"
)

// EnsureAtiusLocalEmbeddingsMetadata registers the built-in local Atius
// embedding and reranker aliases in the models metadata table when they are
// missing, preserving any administrator-managed records that already exist.
func EnsureAtiusLocalEmbeddingsMetadata() (int, error) {
	embeddingsEndpoints, err := common.Marshal(map[string]any{
		string(constant.EndpointTypeEmbeddings): map[string]string{
			"path":   "/v1/embeddings",
			"method": "POST",
		},
	})
	if err != nil {
		return 0, err
	}
	rerankEndpoints, err := common.Marshal(map[string]any{
		string(constant.EndpointTypeJinaRerank): map[string]string{
			"path":   "/v1/rerank",
			"method": "POST",
		},
	})
	if err != nil {
		return 0, err
	}

	seeds := []exactModelMetadataSeed{
		{
			ModelName:    constant.AtiusLocalEmbeddingModel,
			Description:  "TEI GTE Embeddings",
			Tags:         "Embeddings,Local TEI,Governor",
			Endpoints:    string(embeddingsEndpoints),
			SyncOfficial: 1,
		},
		{
			ModelName:    constant.AtiusLocalRerankerModel,
			Description:  "TEI GTE Reranker",
			Tags:         "Reranker,Local TEI,Governor",
			Endpoints:    string(rerankEndpoints),
			SyncOfficial: 1,
		},
	}

	created := 0
	err = DB.Transaction(func(tx *gorm.DB) error {
		var vendor Vendor
		findVendorErr := tx.Where("name = ?", atiusLocalVendorName).First(&vendor).Error
		if findVendorErr != nil && findVendorErr != gorm.ErrRecordNotFound {
			return findVendorErr
		}
		if findVendorErr == gorm.ErrRecordNotFound {
			findVendorErr = tx.Where("name = ?", atiusLocalLegacyVendorName).First(&vendor).Error
			if findVendorErr != nil && findVendorErr != gorm.ErrRecordNotFound {
				return findVendorErr
			}
			if findVendorErr == gorm.ErrRecordNotFound {
				now := common.GetTimestamp()
				vendor = Vendor{
					Name:        atiusLocalVendorName,
					Icon:        atiusLocalIcon,
					Status:      1,
					CreatedTime: now,
					UpdatedTime: now,
				}
				if err := tx.Create(&vendor).Error; err != nil {
					return err
				}
			}
		}

		if vendor.Name != atiusLocalVendorName || vendor.Icon != atiusLocalIcon || vendor.Status != 1 {
			if err := tx.Model(&vendor).Updates(map[string]any{
				"name":         atiusLocalVendorName,
				"icon":         atiusLocalIcon,
				"status":       1,
				"updated_time": common.GetTimestamp(),
			}).Error; err != nil {
				return err
			}
			vendor.Name = atiusLocalVendorName
			vendor.Icon = atiusLocalIcon
			vendor.Status = 1
		}

		modelNames := make([]string, 0, len(seeds))
		for _, seed := range seeds {
			modelNames = append(modelNames, seed.ModelName)
		}
		var existing []Model
		if err := tx.Where("model_name IN ?", modelNames).Find(&existing).Error; err != nil {
			return err
		}
		existingNames := make(map[string]struct{}, len(existing))
		for _, item := range existing {
			existingNames[item.ModelName] = struct{}{}
			if item.VendorID == vendor.Id && item.Icon == atiusLocalIcon {
				continue
			}
			if err := tx.Model(&Model{}).Where("id = ?", item.Id).Updates(map[string]any{
				"vendor_id":    vendor.Id,
				"icon":         atiusLocalIcon,
				"updated_time": common.GetTimestamp(),
			}).Error; err != nil {
				return err
			}
		}

		now := common.GetTimestamp()
		for _, seed := range seeds {
			if _, exists := existingNames[seed.ModelName]; exists {
				continue
			}
			item := Model{
				ModelName:    seed.ModelName,
				Description:  seed.Description,
				Icon:         atiusLocalIcon,
				Tags:         seed.Tags,
				VendorID:     vendor.Id,
				Endpoints:    seed.Endpoints,
				Status:       1,
				SyncOfficial: seed.SyncOfficial,
				NameRule:     NameRuleExact,
				CreatedTime:  now,
				UpdatedTime:  now,
			}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			if err := tx.Model(&Model{}).Where("id = ?", item.Id).Updates(map[string]any{
				"status":        1,
				"sync_official": seed.SyncOfficial,
			}).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	return created, err
}
