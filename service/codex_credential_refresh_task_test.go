package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestShouldAutoRefreshCodexChannel(t *testing.T) {
	externalSetting := `{"codex_credential_source":"` + dto.CodexCredentialSourceExternalFile + `"}`
	tests := []struct {
		name    string
		channel *model.Channel
		want    bool
	}{
		{name: "nil channel", channel: nil, want: false},
		{
			name:    "enabled router-managed credential",
			channel: &model.Channel{Status: common.ChannelStatusEnabled},
			want:    true,
		},
		{
			name:    "auto-disabled router-managed credential",
			channel: &model.Channel{Status: common.ChannelStatusAutoDisabled},
			want:    true,
		},
		{
			name:    "disabled credential",
			channel: &model.Channel{Status: common.ChannelStatusManuallyDisabled},
			want:    false,
		},
		{
			name: "multi-key credential",
			channel: &model.Channel{
				Status:      common.ChannelStatusEnabled,
				ChannelInfo: model.ChannelInfo{IsMultiKey: true},
			},
			want: false,
		},
		{
			name: "externally managed credential",
			channel: &model.Channel{
				Status:  common.ChannelStatusEnabled,
				Setting: &externalSetting,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldAutoRefreshCodexChannel(tt.channel))
		})
	}
}
