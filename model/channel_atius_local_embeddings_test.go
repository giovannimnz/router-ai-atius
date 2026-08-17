package model

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelInfoValueReturnsJSONText(t *testing.T) {
	value, err := (ChannelInfo{}).Value()
	require.NoError(t, err)

	textValue, ok := value.(string)
	require.True(t, ok, "driver.Value must be JSON text, got %T", value)
	require.JSONEq(t, `{
		"is_multi_key": false,
		"multi_key_size": 0,
		"multi_key_status_list": null,
		"multi_key_polling_index": 0,
		"multi_key_mode": ""
	}`, textValue)
}

func TestApplyAtiusLocalEmbeddingsDefaults(t *testing.T) {
	t.Setenv("TEI_BASE_URL", "http://embeddings.internal:3115/")
	t.Setenv("TEI_RERANKER_BASE_URL", "http://reranker.internal:31216/")

	channel := &Channel{Type: constant.ChannelTypeAtiusLocalEmbeddings}
	require.NoError(t, channel.ApplyAtiusLocalEmbeddingsDefaults())

	assert.Equal(t, "Atius Local Embeddings", channel.Name)
	assert.Equal(t, "embedding-gte-v1,reranker-gte-multilingual-v1", channel.Models)
	assert.Equal(t, "default", channel.Group)
	require.NotNil(t, channel.BaseURL)
	assert.Equal(t, "http://embeddings.internal:3115", *channel.BaseURL)
	require.NotNil(t, channel.TestModel)
	assert.Equal(t, "embedding-gte-v1", *channel.TestModel)

	config := channel.GetOtherSettings().AdvancedCustom
	require.NotNil(t, config)
	require.Len(t, config.Routes, 2)
	assert.Equal(t, dto.AdvancedCustomRoute{
		IncomingPath: "/v1/embeddings",
		UpstreamPath: "http://embeddings.internal:3115/v1/embeddings",
		Converter:    dto.AdvancedCustomConverterNone,
		Auth:         &dto.AdvancedCustomRouteAuth{Type: dto.AdvancedCustomAuthTypeNone},
	}, config.Routes[0])
	assert.Equal(t, dto.AdvancedCustomRoute{
		IncomingPath: "/v1/rerank",
		UpstreamPath: "http://reranker.internal:31216/rerank",
		Converter:    dto.AdvancedCustomConverterJinaRerankToTEINative,
		Auth:         &dto.AdvancedCustomRouteAuth{Type: dto.AdvancedCustomAuthTypeNone},
	}, config.Routes[1])
	require.NoError(t, channel.ValidateSettings())
}

func TestApplyAtiusLocalEmbeddingsDefaultsPreservesExplicitConfig(t *testing.T) {
	baseURL := "http://custom-embeddings:8080"
	testModel := "custom-embedding"
	customConfig := dto.ChannelOtherSettings{
		AdvancedCustom: &dto.AdvancedCustomConfig{
			Routes: []dto.AdvancedCustomRoute{{
				IncomingPath: "/v1/embeddings",
				UpstreamPath: "/custom/embeddings",
				Converter:    dto.AdvancedCustomConverterNone,
			}},
		},
	}
	channel := &Channel{
		Type:      constant.ChannelTypeAtiusLocalEmbeddings,
		Name:      "Custom local stack",
		Models:    "custom-embedding",
		Group:     "internal",
		BaseURL:   &baseURL,
		TestModel: &testModel,
	}
	channel.SetOtherSettings(customConfig)

	require.NoError(t, channel.ApplyAtiusLocalEmbeddingsDefaults())
	assert.Equal(t, "Custom local stack", channel.Name)
	assert.Equal(t, "custom-embedding", channel.Models)
	assert.Equal(t, "internal", channel.Group)
	assert.Equal(t, baseURL, *channel.BaseURL)
	assert.Equal(t, testModel, *channel.TestModel)
	assert.Equal(t, customConfig.AdvancedCustom, channel.GetOtherSettings().AdvancedCustom)
}
