package controller

import (
	"errors"
	"testing"

	"github.com/gin-contrib/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingCodexOAuthSession struct {
	values  map[interface{}]interface{}
	saveErr error
}

func (session *failingCodexOAuthSession) ID() string { return "test-session" }

func (session *failingCodexOAuthSession) Get(key interface{}) interface{} {
	return session.values[key]
}

func (session *failingCodexOAuthSession) Set(key interface{}, value interface{}) {
	session.values[key] = value
}

func (session *failingCodexOAuthSession) Delete(key interface{}) {
	delete(session.values, key)
}

func (session *failingCodexOAuthSession) Clear() {
	clear(session.values)
}

func (session *failingCodexOAuthSession) AddFlash(interface{}, ...string) {}

func (session *failingCodexOAuthSession) Flashes(...string) []interface{} { return nil }

func (session *failingCodexOAuthSession) Options(sessions.Options) {}

func (session *failingCodexOAuthSession) Save() error { return session.saveErr }

func TestCodexDeviceOAuthSessionSaveFailuresAreReturned(t *testing.T) {
	saveErr := errors.New("session backend unavailable")
	session := &failingCodexOAuthSession{values: make(map[interface{}]interface{}), saveErr: saveErr}

	err := saveCodexDeviceOAuthSession(session, 5, "device-id")
	require.ErrorIs(t, err, saveErr)
	assert.Equal(t, "device-id", session.Get(codexOAuthSessionKey(5, "device_auth_id")))

	err = clearCodexDeviceOAuthSession(session, 5)
	require.ErrorIs(t, err, saveErr)
	assert.Nil(t, session.Get(codexOAuthSessionKey(5, "device_auth_id")))
}
