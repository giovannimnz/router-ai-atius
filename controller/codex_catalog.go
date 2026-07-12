package controller

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const codexCatalogManualSyncTimeout = 5 * time.Minute

type codexCatalogSyncRequest struct {
	ChannelID int `json:"channel_id"`
}

func SyncCodexCatalog(c *gin.Context) {
	req := codexCatalogSyncRequest{}
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid request body"})
		return
	}

	channelID := req.ChannelID
	if channelID == 0 {
		channelID = 5
	}
	if channelID < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "channel_id must be positive"})
		return
	}

	channel, err := model.GetChannelById(channelID, true)
	if errors.Is(err, gorm.ErrRecordNotFound) || channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "channel not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to load channel"})
		return
	}
	if channel.Type != constant.ChannelTypeCodex {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "channel is not a Codex channel"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), codexCatalogManualSyncTimeout)
	defer cancel()

	result, err := service.SyncCodexCatalog(ctx, channelID)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			c.JSON(http.StatusGatewayTimeout, gin.H{"success": false, "message": "Codex catalog sync timed out"})
			return
		}
		if issue := service.ClassifyCodexCredentialIssue(err, 0); issue.IsAuth {
			c.JSON(http.StatusBadGateway, gin.H{
				"success":               false,
				"message":               issue.Message,
				"code":                  issue.Code,
				"requires_regeneration": issue.RequiresRegeneration,
				"upstream_status":       issue.UpstreamStatus,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Codex catalog sync failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"channel_id":       result.ChannelID,
			"discovered_count": len(result.Discovered),
			"promoted_count":   len(result.Promoted),
			"changed":          result.Changed,
			"validated_count":  result.ValidatedCount,
		},
	})
}
