package controller

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/relay/channel/codex"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// FetchCodexModels returns the list of models for a Codex channel.
// If the channel has a valid credential, fetches live models from upstream.
// Falls back to the static model list.
// POST /api/channel/codex/models
func FetchCodexModels(c *gin.Context) {
	type fetchReq struct {
		Key string `json:"key"`
	}

	var req fetchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid request"})
		return
	}

	key := strings.TrimSpace(req.Key)

	// Try fetching live models if we have a credential
	if key != "" && strings.HasPrefix(key, "{") {
		oauthKey, err := codex.ParseOAuthKey(key)
		if err == nil && oauthKey.AccessToken != "" {
			models, err := service.FetchCodexLiveModels(c.Request.Context(), oauthKey.AccessToken, oauthKey.AccountID)
			if err == nil && len(models) > 0 {
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"data": gin.H{
						"models": models,
						"source": "upstream",
					},
				})
				return
			}
		}
	}

	// Fallback to static list
	staticModels := codex.ModelList
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"models": staticModels,
			"source": "static",
		},
	})
}
