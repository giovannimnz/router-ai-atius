package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSharedCodexCredentialReference(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "default marker", key: "shared:codex", want: true},
		{name: "explicit channel marker", key: " shared:codex:5 ", want: true},
		{name: "oauth json", key: `{"access_token":"token"}`, want: false},
		{name: "empty", key: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isSharedCodexCredentialReference(tt.key))
		})
	}
}
