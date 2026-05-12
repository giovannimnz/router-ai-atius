package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ListAllModelsWithChannel returns all abilities with channel_type for internal use by FastAPI middleware
// GET /internal/v1/models (no auth required)
func ListAllModelsWithChannel(c *gin.Context) {
	// Query all abilities with channel type info
	var abilities []model.AbilityWithChannel
	subQuery := model.DB.Table("channels").Select("id").Where("status = ?", common.ChannelStatusEnabled)
	err := model.DB.Table("abilities").
		Select("abilities.model, abilities.channel_id, abilities.enabled, abilities.group, channels.type as channel_type").
		Joins("LEFT JOIN channels ON abilities.channel_id = channels.id").
		Where("abilities.channel_id IN (?) AND abilities.enabled = ?", subQuery, true).
		Find(&abilities).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "query failed"})
		return
	}

	// Return raw abilities data
	c.JSON(200, gin.H{
		"data":  abilities,
		"count": len(abilities),
	})
}
