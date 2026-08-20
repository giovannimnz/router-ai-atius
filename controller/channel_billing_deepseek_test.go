package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDeepSeekUSDBalance(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		want       float64
		wantErrMsg string
	}{
		{
			name: "selects normalized USD when CNY is also present",
			body: `{
				"is_available": true,
				"balance_infos": [
					{"currency": "CNY", "total_balance": "999.00"},
					{"currency": " usd ", "total_balance": " 42.75 "}
				]
			}`,
			want: 42.75,
		},
		{
			name: "rejects response without USD",
			body: `{
				"is_available": true,
				"balance_infos": [
					{"currency": "CNY", "total_balance": "100.00"}
				]
			}`,
			wantErrMsg: "DeepSeek balance response does not include USD currency",
		},
		{
			name: "rejects invalid USD total balance",
			body: `{
				"is_available": true,
				"balance_infos": [
					{"currency": "USD", "total_balance": "not-a-number"}
				]
			}`,
			wantErrMsg: "invalid DeepSeek USD total_balance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			balance, err := parseDeepSeekUSDBalance([]byte(tt.body))
			if tt.wantErrMsg != "" {
				require.ErrorContains(t, err, tt.wantErrMsg)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, balance)
		})
	}
}
