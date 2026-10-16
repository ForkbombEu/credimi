// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const actionTestIssuance = `appId: wallet
name: issue
env:
  pin: "000000"
  deeplink: "unused-default"
---
- launchApp
- openLink: ${deeplink}
- inputText: ${pin}
- takeScreenshot: credential-added
`

const actionTestPresentation = `appId: wallet
---
- openLink: ${deeplink}
- takeScreenshot: shared
`

func TestComposeMobileFlowRunsActionsInOrderWithParameterPrecedence(t *testing.T) {
	parameters := map[string]any{"pin": "123456", "lang": "en"}

	composed, err := composeMobileFlow(parameters, []mobileAction{
		{Label: "tenant/issue", Code: actionTestIssuance, Parameters: map[string]any{
			"deeplink": "openid-credential-offer://offer",
		}},
		{Label: "actions[1]", Code: actionTestPresentation, Parameters: map[string]any{
			"deeplink": "openid4vp://present",
			"lang":     "it",
		}},
	})
	require.NoError(t, err)

	require.Equal(t, map[string]any{
		"pin":               "123456",
		"lang":              "en",
		"action_0_deeplink": "openid-credential-offer://offer",
		"action_1_deeplink": "openid4vp://present",
		"action_1_lang":     "it",
	}, parameters)

	header, commands := decodeComposedFlow(t, composed)
	require.Equal(t, map[string]any{"appId": "wallet"}, header)
	require.Equal(t, []any{
		map[string]any{"runFlow": map[string]any{
			// The step's pin overrides the action's default, so the default
			// is not set; the action's own deeplink overrides both.
			"env": map[string]any{"deeplink": "${action_0_deeplink}"},
			"commands": []any{
				"launchApp",
				map[string]any{"openLink": "${deeplink}"},
				map[string]any{"inputText": "${pin}"},
				map[string]any{"takeScreenshot": "credential-added"},
			},
		}},
		map[string]any{"runFlow": map[string]any{
			"env": map[string]any{
				"deeplink": "${action_1_deeplink}",
				"lang":     "${action_1_lang}",
			},
			"commands": []any{
				map[string]any{"openLink": "${deeplink}"},
				map[string]any{"takeScreenshot": "shared"},
			},
		}},
	}, commands, "every action keeps all its commands, screenshots included")
}

func TestComposeMobileFlowKeepsActionDefaultsWithoutOverride(t *testing.T) {
	composed, err := composeMobileFlow(map[string]any{}, []mobileAction{
		{Label: "tenant/issue", Code: actionTestIssuance},
	})
	require.NoError(t, err)
	_, commands := decodeComposedFlow(t, composed)
	runFlow := commands[0].(map[string]any)["runFlow"].(map[string]any)
	require.Equal(t, map[string]any{"pin": "000000", "deeplink": "unused-default"}, runFlow["env"])
}

func TestComposeMobileFlowRejectsActionsThatCannotShareAFlow(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		params map[string]any
		want   string
	}{
		{
			name: "other app",
			code: "appId: other\n---\n- launchApp\n",
			want: `drives app "other"`,
		},
		{
			name: "flow hook",
			code: "appId: wallet\nonFlowStart:\n  - launchApp\n---\n- launchApp\n",
			want: `flow key "onFlowStart"`,
		},
		{
			name:   "parameter clash",
			code:   "appId: wallet\n---\n- launchApp\n",
			params: map[string]any{"deeplink": "x"},
			want:   "parameter action_1_deeplink is already set",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := composeMobileFlow(
				map[string]any{"action_1_deeplink": "taken"},
				[]mobileAction{
					{Label: "first", Code: actionTestPresentation},
					{Label: "second", Code: tc.code, Parameters: tc.params},
				},
			)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestResolveStoredMobileActionsComposesActions(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerStubActivity(env, activities.ResolveRecordActivityName)
	// One fetch serves every step that runs the same stored action.
	onResolveMobileAction(env, "tenant/issue", map[string]any{"code": actionTestIssuance})

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) ([]map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Minute},
			)
			actions := func(offer string) []any {
				return []any{
					map[string]any{
						"id":         "tenant/issue",
						"parameters": map[string]any{"deeplink": offer},
					},
					map[string]any{"code": actionTestPresentation},
				}
			}
			steps := []pipeline.StepDefinition{
				{StepSpec: pipeline.StepSpec{
					ID:  "first",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{Payload: map[string]any{
						"version_id": mobileExternalSourceVersionID,
						"parameters": map[string]any{"deeplink": "${{ a.outputs.deeplink }}"},
						"actions":    actions("${{ offer-a.outputs.deeplink }}"),
					}},
				}},
				{StepSpec: pipeline.StepSpec{
					ID:  "second",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{Payload: map[string]any{
						"version_id": mobileExternalSourceVersionID,
						"actions":    actions("${{ offer-b.outputs.deeplink }}"),
					}},
				}},
			}
			if err := resolveStoredMobileActions(ctx, &steps, map[string]any{
				"namespace": "tenant",
			}); err != nil {
				return nil, err
			}
			return []map[string]any{steps[0].With.Payload, steps[1].With.Payload}, nil
		},
		workflow.RegisterOptions{Name: "test-compose-actions"},
	)

	env.ExecuteWorkflow("test-compose-actions")
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)

	var payloads []map[string]any
	require.NoError(t, env.GetWorkflowResult(&payloads))
	require.Equal(t, map[string]any{
		"deeplink":          "${{ a.outputs.deeplink }}",
		"action_0_deeplink": "${{ offer-a.outputs.deeplink }}",
	}, payloads[0]["parameters"])
	require.Equal(t, map[string]any{
		"action_0_deeplink": "${{ offer-b.outputs.deeplink }}",
	}, payloads[1]["parameters"])
	for _, payload := range payloads {
		require.NotContains(t, payload, mobileActionsPayloadKey)
		require.NotContains(t, payload, "stored_action_code", "an inline action is not stored")
		_, commands := decodeComposedFlow(t, payload["action_code"].(string))
		require.Len(t, commands, 2)
	}
}

func TestResolveStoredMobileActionsRejectsInvalidActions(t *testing.T) {
	tests := []struct {
		name    string
		payload map[string]any
		want    string
	}{
		{
			name: "with action_id",
			payload: map[string]any{
				"action_id": "tenant/present",
				"actions":   []any{map[string]any{"code": actionTestPresentation}},
			},
			want: "action_id cannot be combined with actions",
		},
		{
			name: "with action_code",
			payload: map[string]any{
				"action_code": actionTestPresentation,
				"actions":     []any{map[string]any{"code": actionTestPresentation}},
			},
			want: "action_code cannot be combined with actions",
		},
		{
			name:    "empty",
			payload: map[string]any{"actions": []any{}},
			want:    "actions is empty",
		},
		{
			name: "id and code",
			payload: map[string]any{"actions": []any{map[string]any{
				"id":   "tenant/issue",
				"code": actionTestPresentation,
			}}},
			want: "actions[0] has both id and code",
		},
		{
			name: "neither id nor code",
			payload: map[string]any{
				"actions": []any{map[string]any{"parameters": map[string]any{}}},
			},
			want: "actions[0] has neither id nor code",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflowWithOptions(
				testResolveStoredMobileActionsWorkflow,
				workflow.RegisterOptions{Name: "test-invalid-actions"},
			)
			tc.payload["version_id"] = mobileExternalSourceVersionID
			env.ExecuteWorkflow("test-invalid-actions", []pipeline.StepDefinition{{
				StepSpec: pipeline.StepSpec{
					ID:   "step",
					Use:  mobileAutomationStepUse,
					With: pipeline.StepInputs{Payload: tc.payload},
				},
			}})
			require.ErrorContains(t, env.GetWorkflowError(), tc.want)
		})
	}
}

func decodeComposedFlow(t *testing.T, code string) (map[string]any, []any) {
	t.Helper()
	header, commands, err := splitMaestroFlow(code)
	require.NoError(t, err)
	var decodedHeader map[string]any
	require.NoError(t, header.Decode(&decodedHeader))
	var decodedCommands []any
	require.NoError(t, commands.Decode(&decodedCommands))
	return decodedHeader, decodedCommands
}
