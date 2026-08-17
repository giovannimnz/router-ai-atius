package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartCodexDeviceAuthorizationReturnsUserFacingCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var request map[string]string
		require.NoError(t, common.DecodeJson(r.Body, &request))
		assert.Equal(t, "client-id", request["client_id"])
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"device_auth_id":"internal-device-id","usercode":"ABCD-EFGH","interval":"7"}`))
	}))
	defer server.Close()

	result, err := startCodexDeviceAuthorization(context.Background(), server.Client(), server.URL, "client-id")

	require.NoError(t, err)
	assert.Equal(t, "internal-device-id", result.DeviceAuthID)
	assert.Equal(t, "ABCD-EFGH", result.UserCode)
	assert.Equal(t, codexDeviceVerifyURL, result.VerificationURL)
	assert.Equal(t, 7*time.Second, result.Interval)
	assert.WithinDuration(t, time.Now().Add(codexDeviceFlowTTL), result.ExpiresAt, time.Second)
}

func TestPollCodexDeviceAuthorizationDistinguishesPendingAndCompletion(t *testing.T) {
	polls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		polls++
		var request map[string]string
		require.NoError(t, common.DecodeJson(r.Body, &request))
		assert.Equal(t, "internal-device-id", request["device_auth_id"])
		assert.Equal(t, "ABCD-EFGH", request["user_code"])
		if polls == 1 {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"authorization_code":"authorization-code","code_verifier":"code-verifier"}`))
	}))
	defer server.Close()

	pending, err := pollCodexDeviceAuthorization(context.Background(), server.Client(), server.URL, "internal-device-id", "ABCD-EFGH")
	require.NoError(t, err)
	assert.True(t, pending.Pending)

	completed, err := pollCodexDeviceAuthorization(context.Background(), server.Client(), server.URL, "internal-device-id", "ABCD-EFGH")
	require.NoError(t, err)
	assert.False(t, completed.Pending)
	assert.Equal(t, "authorization-code", completed.AuthorizationCode)
	assert.Equal(t, "code-verifier", completed.CodeVerifier)
}
