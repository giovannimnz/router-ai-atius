package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	CodexCatalogStatusDiscovered = "discovered"
	CodexCatalogStatusEnriched   = "enriched"
	CodexCatalogStatusValidated  = "validated"
	CodexCatalogStatusPromoted   = "promoted"
	CodexCatalogStatusRejected   = "rejected"
)

type CodexCatalogSnapshot struct {
	Id            int            `json:"id"`
	ChannelID     int            `json:"channel_id" gorm:"index:idx_codex_catalog_snapshot_channel_time"`
	SnapshotHash  string         `json:"snapshot_hash" gorm:"size:64;index"`
	ClientVersion string         `json:"client_version" gorm:"size:32"`
	ModelCount    int            `json:"model_count"`
	Snapshot      string         `json:"snapshot" gorm:"type:text"`
	CreatedTime   int64          `json:"created_time" gorm:"bigint;index:idx_codex_catalog_snapshot_channel_time"`
	UpdatedTime   int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

type CodexCatalogCandidate struct {
	Id                  int            `json:"id"`
	ChannelID           int            `json:"channel_id" gorm:"uniqueIndex:uk_codex_catalog_candidate,priority:1;index"`
	ModelName           string         `json:"model_name" gorm:"size:128;not null;uniqueIndex:uk_codex_catalog_candidate,priority:2;index"`
	Status              string         `json:"status" gorm:"size:24;index"`
	DiscoveryHash       string         `json:"discovery_hash" gorm:"size:64;index"`
	DiscoveryMetadata   string         `json:"discovery_metadata" gorm:"type:text"`
	SourceMetadata      string         `json:"source_metadata" gorm:"type:text"`
	OverrideMetadata    string         `json:"override_metadata" gorm:"type:text"`
	DisplayName         string         `json:"display_name" gorm:"type:text"`
	Provider            string         `json:"provider" gorm:"size:64"`
	OwnedBy             string         `json:"owned_by" gorm:"size:64"`
	EndpointPreference  string         `json:"endpoint_preference" gorm:"size:32"`
	SupportedEndpoints  string         `json:"supported_endpoints" gorm:"type:text"`
	ContextWindowTokens int            `json:"context_window_tokens"`
	MaxTokens           int            `json:"max_tokens"`
	MaxCompletionTokens int            `json:"max_completion_tokens"`
	ValidationState     string         `json:"validation_state" gorm:"size:24;index"`
	ValidationOutput    string         `json:"validation_output" gorm:"type:text"`
	ValidationError     string         `json:"validation_error" gorm:"type:text"`
	Promoted            bool           `json:"promoted" gorm:"index"`
	LastDiscoveredTime  int64          `json:"last_discovered_time" gorm:"bigint"`
	LastValidatedTime   int64          `json:"last_validated_time" gorm:"bigint"`
	LastPromotedTime    int64          `json:"last_promoted_time" gorm:"bigint"`
	LastSeenTime        int64          `json:"last_seen_time" gorm:"bigint"`
	CreatedTime         int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime         int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (s *CodexCatalogSnapshot) Save() error {
	now := common.GetTimestamp()
	if s.CreatedTime == 0 {
		s.CreatedTime = now
	}
	s.UpdatedTime = now
	return DB.Save(s).Error
}

func (c *CodexCatalogCandidate) Save() error {
	now := common.GetTimestamp()
	if c.CreatedTime == 0 {
		c.CreatedTime = now
	}
	c.UpdatedTime = now
	return DB.Save(c).Error
}

func GetLatestCodexCatalogSnapshot(channelID int) (*CodexCatalogSnapshot, error) {
	var snapshot CodexCatalogSnapshot
	err := DB.Where("channel_id = ?", channelID).Order("created_time desc").First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func GetCodexCatalogCandidatesByChannel(channelID int) ([]*CodexCatalogCandidate, error) {
	var candidates []*CodexCatalogCandidate
	err := DB.Where("channel_id = ?", channelID).Order("model_name asc").Find(&candidates).Error
	return candidates, err
}

func GetPromotedCodexCatalogCandidatesByChannel(channelID int) ([]*CodexCatalogCandidate, error) {
	var candidates []*CodexCatalogCandidate
	err := DB.Where("channel_id = ? AND promoted = ?", channelID, true).Order("model_name asc").Find(&candidates).Error
	return candidates, err
}

func FindCodexCatalogCandidate(channelID int, modelName string) (*CodexCatalogCandidate, error) {
	var candidate CodexCatalogCandidate
	err := DB.Where("channel_id = ? AND model_name = ?", channelID, strings.TrimSpace(modelName)).First(&candidate).Error
	if err != nil {
		return nil, err
	}
	return &candidate, nil
}
