// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestParseWalletURLActivity_Execute(t *testing.T) {
	var ts testsuite.WorkflowTestSuite
	env := ts.NewTestActivityEnvironment()

	act := NewParseWalletURLActivity()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
		Name: act.Name(),
	})
	tests := []struct {
		name       string
		payload    ParseWalletURLActivityPayload
		wantOutput map[string]any
		expectErr  bool
		errCode    errorcodes.Code
	}{
		{
			name: "valid Google Play URL",
			payload: ParseWalletURLActivityPayload{
				URL: "https://play.google.com/store/apps/details?id=com.example.wallet",
			},
			wantOutput: map[string]any{
				"api_input":  "https://play.google.com/store/apps/details?id=com.example.wallet",
				"store_type": "google",
			},
			expectErr: false,
		},
		{
			name: "valid Apple App Store URL",
			payload: ParseWalletURLActivityPayload{
				URL: "https://apps.apple.com/us/app/example-wallet/id1234567890",
			},
			wantOutput: map[string]any{
				"api_input":  "1234567890",
				"store_type": "apple",
			},
			expectErr: false,
		},
		{
			name:      "missing url field",
			payload:   ParseWalletURLActivityPayload{},
			expectErr: true,
			errCode:   errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "invalid url format",
			payload: ParseWalletURLActivityPayload{
				URL: "::::://bad-url",
			},
			expectErr: true,
			errCode:   errorcodes.Codes[errorcodes.ParseURLFailed],
		},
		{
			name: "apple url without id",
			payload: ParseWalletURLActivityPayload{
				URL: "https://apps.apple.com/us/app/example-wallet/",
			},
			expectErr: true,
			errCode:   errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
		{
			name: "unsupported store domain",
			payload: ParseWalletURLActivityPayload{
				URL: "https://example.com/wallet",
			},
			expectErr: true,
			errCode:   errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workflowengine.ActivityInput{
				Payload: tt.payload,
			}

			future, err := env.ExecuteActivity(act.Execute, input)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errCode.Code)
				require.Contains(t, err.Error(), tt.errCode.Description)
			} else {
				require.NoError(t, err)
				var result workflowengine.ActivityResult
				require.NoError(t, future.Get(&result))
				require.Equal(t, tt.wantOutput, result.Output.(map[string]any))
			}
		})
	}
}
