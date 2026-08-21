package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/service/authz"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelStatusRoutesUseOperatePermission(t *testing.T) {
	assertChannelRoutePermission(t, http.MethodPost, "/:id/status", authz.ChannelOperate, controller.UpdateChannelStatus)
	assertChannelRoutePermission(t, http.MethodPost, "/status/batch", authz.ChannelOperate, controller.BatchUpdateChannelStatus)
	assertChannelRoutePermission(t, http.MethodPut, "/", authz.ChannelWrite, controller.UpdateChannel)
}

func TestChannelDeleteRoutesUseSensitiveWritePermission(t *testing.T) {
	assertChannelRoutePermission(t, http.MethodDelete, "/:id", authz.ChannelSensitiveWrite, controller.DeleteChannel)
	assertChannelRoutePermission(t, http.MethodPost, "/batch", authz.ChannelSensitiveWrite, controller.DeleteChannelBatch)
	assertChannelRoutePermission(t, http.MethodDelete, "/disabled", authz.ChannelSensitiveWrite, controller.DeleteDisabledChannel)
	assertChannelRoutePermission(t, http.MethodPut, "/", authz.ChannelWrite, controller.UpdateChannel)
	assertChannelRoutePermission(t, http.MethodPut, "/tag", authz.ChannelWrite, controller.EditTagChannels)
	assertChannelRoutePermission(t, http.MethodPost, "/batch/tag", authz.ChannelWrite, controller.BatchSetChannelTag)
}

func TestCodexDeviceAuthorizationRoutesUseSensitiveWritePermission(t *testing.T) {
	assertChannelRoutePermission(t, http.MethodPost, "/:id/codex/regenerate/device/start", authz.ChannelSensitiveWrite, controller.StartCodexDeviceOAuthForChannel)
	assertChannelRoutePermission(t, http.MethodPost, "/:id/codex/regenerate/device/poll", authz.ChannelSensitiveWrite, controller.PollCodexDeviceOAuthForChannel)
	assertChannelRoutePermission(t, http.MethodPost, "/:id/codex/regenerate/device/cancel", authz.ChannelSensitiveWrite, controller.CancelCodexDeviceOAuthForChannel)
}

func TestCodexLegacyPKCERoutesRemainRegisteredAsFailClosedHandlers(t *testing.T) {
	assertChannelRoutePermission(t, http.MethodPost, "/:id/codex/regenerate/start", authz.ChannelSensitiveWrite, controller.StartCodexOAuthForChannel)
	assertChannelRoutePermission(t, http.MethodPost, "/:id/codex/regenerate/complete", authz.ChannelSensitiveWrite, controller.CompleteCodexOAuthForChannel)
}

func TestCodexLegacyPKCEHandlersFailClosedBeforeAnyMutation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, handler := range []gin.HandlerFunc{
		controller.StartCodexOAuthForChannel,
		controller.CompleteCodexOAuthForChannel,
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPost, "/legacy", nil)

		handler(ctx)

		assert.Equal(t, http.StatusGone, recorder.Code)
		var response map[string]any
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		assert.Equal(t, false, response["success"])
		assert.Equal(t, "codex_pkce_disabled", response["code"])
	}
}

func TestChannelStatusRoutesRegisterWithoutConflict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")

	require.NotPanics(t, func() {
		registerChannelRoutes(api)
	})
}

func assertChannelRoutePermission(t *testing.T, method string, path string, permission authz.Permission, handler any) {
	t.Helper()
	for _, route := range channelPermissionRoutes {
		if route.method == method && route.path == path {
			assert.Equal(t, permission, route.permission)
			assert.Equal(t, reflect.ValueOf(handler).Pointer(), reflect.ValueOf(route.handler).Pointer())
			return
		}
	}
	t.Fatalf("route %s %s not found", method, path)
}
