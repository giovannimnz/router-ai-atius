package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadCodexCredentialMetadata(t *testing.T) {
	t.Run("valid oauth key", func(t *testing.T) {
		meta := ReadCodexCredentialMetadata(`{"access_token":"token","refresh_token":"refresh","account_id":"acct_123","expired":"2026-07-04T22:00:00Z"}`)

		assert.True(t, meta.OAuth)
		assert.True(t, meta.Authenticated)
		assert.Equal(t, "2026-07-04T22:00:00Z", meta.ExpiresAt)
	})

	t.Run("invalid payload", func(t *testing.T) {
		meta := ReadCodexCredentialMetadata(`not-json`)

		assert.True(t, meta.OAuth)
		assert.False(t, meta.Authenticated)
		assert.Empty(t, meta.ExpiresAt)
	})
}
