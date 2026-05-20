// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStepInputs_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantConfig  map[string]any
		wantPayload map[string]any
		wantErr     string
	}{
		{
			name: "flat shape",
			input: `{
				"config": { "foo": "bar" },
				"action_id": "onboarding-0001",
				"version_id": "v1"
			}`,
			wantConfig: map[string]any{"foo": "bar"},
			wantPayload: map[string]any{
				"action_id":  "onboarding-0001",
				"version_id": "v1",
			},
		},
		{
			name: "nested payload shape",
			input: `{
				"config": { "foo": "bar" },
				"payload": {
					"action_id": "onboarding-0001",
					"version_id": "v1"
				}
			}`,
			wantConfig: map[string]any{"foo": "bar"},
			wantPayload: map[string]any{
				"action_id":  "onboarding-0001",
				"version_id": "v1",
			},
		},
		{
			name: "mixed shape overrides payload",
			input: `{
				"payload": {
					"action_id": "onboarding-0001",
					"version_id": "v1"
				},
				"version_id": "override"
			}`,
			wantConfig: map[string]any{},
			wantPayload: map[string]any{
				"action_id":  "onboarding-0001",
				"version_id": "override",
			},
		},
		{
			name:    "invalid config type",
			input:   `{ "config": "nope" }`,
			wantErr: "invalid config section",
		},
		{
			name:    "invalid payload type",
			input:   `{ "payload": "nope" }`,
			wantErr: "invalid payload section",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var inputs StepInputs
			err := json.Unmarshal([]byte(test.input), &inputs)
			if test.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.wantConfig, inputs.Config)
			require.Equal(t, test.wantPayload, inputs.Payload)
		})
	}
}

func TestFinallyDefinition_UnmarshalJSON(t *testing.T) {
	t.Run("legacy list becomes always", func(t *testing.T) {
		var wf WorkflowDefinition
		err := json.Unmarshal([]byte(`{
			"finally": [
				{
					"id": "notify",
					"use": "email",
					"with": {
						"payload": {
							"subject": "done"
						}
					}
				}
			]
		}`), &wf)
		require.NoError(t, err)
		require.Len(t, wf.Finally.Always, 1)
		require.Equal(t, "notify", wf.Finally.Always[0].ID)
		require.Empty(t, wf.Finally.OnSuccess)
		require.Empty(t, wf.Finally.OnFailure)
	})

	t.Run("grouped handlers", func(t *testing.T) {
		var wf WorkflowDefinition
		err := json.Unmarshal([]byte(`{
			"finally": {
				"always": [
					{
						"id": "notify-any",
						"use": "http-request",
						"with": {
							"payload": {
								"url": "https://example.com/any"
							}
						}
					}
				],
				"on_success": [
					{
						"id": "notify-success",
						"use": "email",
						"with": {
							"payload": {
								"subject": "success"
							}
						}
					}
				],
				"on_failure": [
					{
						"id": "notify-failure",
						"use": "email",
						"with": {
							"payload": {
								"subject": "failure"
							}
						}
					}
				]
			}
		}`), &wf)
		require.NoError(t, err)
		require.Len(t, wf.Finally.Always, 1)
		require.Len(t, wf.Finally.OnSuccess, 1)
		require.Len(t, wf.Finally.OnFailure, 1)
		require.Equal(t, "notify-any", wf.Finally.Always[0].ID)
		require.Equal(t, "notify-success", wf.Finally.OnSuccess[0].ID)
		require.Equal(t, "notify-failure", wf.Finally.OnFailure[0].ID)
	})
}
