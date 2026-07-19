package codex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelListExcludesRetiredCodexModels(t *testing.T) {
	assert.NotContains(t, ModelList, "gpt-5.4")
	assert.NotContains(t, ModelList, "gpt-5.4-mini")
	assert.Contains(t, ModelList, "gpt-5.6-sol")
	assert.Contains(t, ModelList, "gpt-5.6-terra")
	assert.Contains(t, ModelList, "gpt-5.6-luna")
}
