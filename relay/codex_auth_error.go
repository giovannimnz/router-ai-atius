package relay

import (
	"errors"
	"net/http"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
)

func normalizeCodexRelayAuthError(info *relaycommon.RelayInfo, relayErr *types.NewAPIError, upstreamStatus int) *types.NewAPIError {
	if info == nil || info.ChannelMeta == nil || relayErr == nil {
		return relayErr
	}
	if info.ApiType != constant.APITypeCodex && info.ChannelType != constant.ChannelTypeCodex {
		return relayErr
	}
	issue := service.ClassifyCodexCredentialIssue(relayErr, upstreamStatus)
	if !issue.IsAuth {
		return relayErr
	}
	if err := service.RecordCodexCredentialIssueByChannelID(info.ChannelId, issue); err != nil {
		return types.NewErrorWithStatusCode(
			errors.New("failed to persist Codex credential health"),
			types.ErrorCodeGetChannelFailed,
			http.StatusInternalServerError,
			types.ErrOptionWithSkipRetry(),
		)
	}
	return service.NormalizeCodexUpstreamAuthError(relayErr)
}
