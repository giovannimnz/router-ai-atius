package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

// NormalizeEndpointMetadata upgrades the legacy rerank endpoint key while
// preserving the configured path, method, and any provider-specific fields.
func NormalizeEndpointMetadata(raw string) (string, bool) {
	if !strings.Contains(raw, string(constant.EndpointTypeJinaRerank)) {
		return raw, false
	}

	var endpoints map[string]any
	if err := common.Unmarshal([]byte(raw), &endpoints); err != nil || endpoints == nil {
		return raw, false
	}

	legacyKey := string(constant.EndpointTypeJinaRerank)
	legacyValue, hasLegacy := endpoints[legacyKey]
	if !hasLegacy {
		return raw, false
	}
	canonicalKey := string(constant.EndpointTypeReranker)
	if _, hasCanonical := endpoints[canonicalKey]; !hasCanonical {
		endpoints[canonicalKey] = legacyValue
	}
	delete(endpoints, legacyKey)

	normalized, err := common.Marshal(endpoints)
	if err != nil {
		return raw, false
	}
	return string(normalized), true
}

func normalizeEndpointPrefillItems(items JSONValue) (JSONValue, bool) {
	raw := string(items)
	if normalized, changed := NormalizeEndpointMetadata(raw); changed {
		return JSONValue(normalized), true
	}

	var values []any
	if err := common.Unmarshal([]byte(raw), &values); err != nil {
		return items, false
	}
	changed := false
	for index, value := range values {
		endpointType, ok := value.(string)
		if !ok {
			continue
		}
		canonical := string(constant.NormalizeEndpointType(constant.EndpointType(endpointType)))
		if canonical != endpointType {
			values[index] = canonical
			changed = true
		}
	}
	if !changed {
		return items, false
	}
	normalized, err := common.Marshal(values)
	if err != nil {
		return items, false
	}
	return JSONValue(normalized), true
}

// MigrateLegacyRerankerEndpointType is cross-database and idempotent. It
// updates persisted model metadata and reusable endpoint groups at startup.
func MigrateLegacyRerankerEndpointType() (int, error) {
	updated := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		var models []Model
		if err := tx.Find(&models).Error; err != nil {
			return err
		}
		for _, item := range models {
			normalized, changed := NormalizeEndpointMetadata(item.Endpoints)
			if !changed {
				continue
			}
			if err := tx.Model(&Model{}).Where("id = ?", item.Id).Updates(map[string]any{
				"endpoints":    normalized,
				"updated_time": common.GetTimestamp(),
			}).Error; err != nil {
				return err
			}
			updated++
		}

		var groups []PrefillGroup
		if err := tx.Where("type = ?", "endpoint").Find(&groups).Error; err != nil {
			return err
		}
		for _, group := range groups {
			normalized, changed := normalizeEndpointPrefillItems(group.Items)
			if !changed {
				continue
			}
			if err := tx.Model(&PrefillGroup{}).Where("id = ?", group.Id).Updates(map[string]any{
				"items":        normalized,
				"updated_time": common.GetTimestamp(),
			}).Error; err != nil {
				return err
			}
			updated++
		}
		return nil
	})
	return updated, err
}
