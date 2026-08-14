package dto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedCustomValidateTEIRerankConverterPath(t *testing.T) {
	valid := &AdvancedCustomConfig{Routes: []AdvancedCustomRoute{{
		IncomingPath: "/v1/rerank",
		UpstreamPath: "/rerank",
		Converter:    AdvancedCustomConverterJinaRerankToTEINative,
	}}}
	require.NoError(t, valid.Validate())

	invalid := &AdvancedCustomConfig{Routes: []AdvancedCustomRoute{{
		IncomingPath: "/v1/embeddings",
		UpstreamPath: "/rerank",
		Converter:    AdvancedCustomConverterJinaRerankToTEINative,
	}}}
	err := invalid.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "converter does not match incoming_path")
}
