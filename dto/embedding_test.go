package dto

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddingInputStatsNilInput(t *testing.T) {
	req := &EmbeddingRequest{}

	stats := req.GetInputStats()

	assert.Equal(t, 0, stats.InputCount)
	assert.Equal(t, 0, stats.InputChars)
}

func TestEmbeddingInputStatsCountsStringAndSlices(t *testing.T) {
	short := strings.Repeat("a", 12)
	medium := strings.Repeat("b", 20)
	long := strings.Repeat("c", 28)

	tests := []struct {
		name      string
		input     any
		wantCount int
		wantChars int
	}{
		{
			name:      "single string",
			input:     short,
			wantCount: 1,
			wantChars: len(short),
		},
		{
			name:      "string slice",
			input:     []string{short, medium},
			wantCount: 2,
			wantChars: len(short) + len(medium),
		},
		{
			name:      "mixed any slice keeps only strings",
			input:     []any{short, 42, medium, true, long},
			wantCount: 3,
			wantChars: len(short) + len(medium) + len(long),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := &EmbeddingRequest{Input: tc.input}

			stats := req.GetInputStats()

			require.NotNil(t, req.GetTokenCountMeta())
			assert.Equal(t, tc.wantCount, stats.InputCount)
			assert.Equal(t, tc.wantChars, stats.InputChars)

			rendered := fmt.Sprintf("%#v", stats)
			assert.NotContains(t, rendered, short)
			assert.NotContains(t, rendered, medium)
			assert.NotContains(t, rendered, long)
		})
	}
}
