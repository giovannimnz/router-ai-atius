package helper

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelMappedHelperCodexLongContextAliases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		alias        string
		upstreamName string
	}{
		{alias: "gpt-5.5-1m", upstreamName: "gpt-5.5"},
		{alias: "gpt-5.4-1m", upstreamName: "gpt-5.4"},
	}

	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			c, _ := gin.CreateTestContext(nil)
			c.Set("model_mapping", `{"gpt-5.5-1m":"gpt-5.5","gpt-5.4-1m":"gpt-5.4"}`)

			request := &dto.GeneralOpenAIRequest{Model: tt.alias}
			info := &relaycommon.RelayInfo{
				OriginModelName: tt.alias,
				ChannelMeta: &relaycommon.ChannelMeta{
					UpstreamModelName: tt.alias,
				},
			}

			require.NoError(t, ModelMappedHelper(c, info, request))
			assert.True(t, info.IsModelMapped)
			assert.Equal(t, tt.alias, info.OriginModelName)
			assert.Equal(t, tt.upstreamName, info.UpstreamModelName)
			assert.Equal(t, tt.upstreamName, request.Model)
		})
	}
}
