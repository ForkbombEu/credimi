// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"fmt"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

const beforeActionTestIssuance = `appId: wallet
name: issue
env:
  pin: "000000"
---
- launchApp
- openLink: ${deeplink}
- inputText: ${pin}
- runFlow:
    when:
      visible: "View details"
    commands:
      - tapOn: "View details"
      - takeScreenshot: nested-added
- takeScreenshot: credential-added
`

const beforeActionTestPresentation = `appId: wallet
---
- launchApp
- openLink: ${deeplink}
- takeScreenshot: shared
`

func TestComposeMobileFlowRunsBeforeActionsFirstWithTheirOwnParameters(t *testing.T) {
	parameters := map[string]any{"deeplink": "openid4vp://present"}

	composed, err := composeMobileFlow(
		beforeActionTestPresentation,
		parameters,
		[]mobileBeforeAction{
			{ActionID: "tenant/issue", Code: beforeActionTestIssuance, Parameters: map[string]any{
				"deeplink": "openid-credential-offer://first",
			}},
			{ActionID: "tenant/issue", Code: beforeActionTestIssuance, Parameters: map[string]any{
				"deeplink": "openid-credential-offer://second",
			}},
		},
	)
	require.NoError(t, err)

	require.Equal(t, map[string]any{
		"deeplink":          "openid4vp://present",
		"before_0_deeplink": "openid-credential-offer://first",
		"before_1_deeplink": "openid-credential-offer://second",
	}, parameters)

	header, commands := decodeComposedFlow(t, composed)
	require.Equal(t, map[string]any{"appId": "wallet"}, header)
	require.Len(t, commands, 5)
	for i, block := range commands[:2] {
		runFlow := block.(map[string]any)["runFlow"].(map[string]any)
		require.Equal(t, map[string]any{
			"pin":      "000000",
			"deeplink": fmt.Sprintf("${before_%d_deeplink}", i),
		}, runFlow["env"])
		require.Equal(t, []any{
			"launchApp",
			map[string]any{"openLink": "${deeplink}"},
			map[string]any{"inputText": "${pin}"},
			map[string]any{"runFlow": map[string]any{
				"when":     map[string]any{"visible": "View details"},
				"commands": []any{map[string]any{"tapOn": "View details"}},
			}},
		}, runFlow["commands"], "before actions keep every command but their screenshots")
	}
	require.Equal(t, []any{
		"launchApp",
		map[string]any{"openLink": "${deeplink}"},
		map[string]any{"takeScreenshot": "shared"},
	}, commands[2:], "the step's own action follows unchanged")
}

func TestComposeMobileFlowRejectsBeforeActionsThatCannotRunInline(t *testing.T) {
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
			want:   "parameter before_0_deeplink is already set",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := composeMobileFlow(
				beforeActionTestPresentation,
				map[string]any{"before_0_deeplink": "taken"},
				[]mobileBeforeAction{{ActionID: "tenant/a", Code: tc.code, Parameters: tc.params}},
			)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestResolveStoredMobileActionsComposesBeforeActions(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()
	registerStubActivity(env, activities.ResolveRecordActivityName)
	// One fetch serves every step that runs the same before action.
	onResolveMobileAction(env, "tenant/issue", map[string]any{"code": beforeActionTestIssuance})
	onResolveMobileAction(env, "tenant/present", map[string]any{
		"code": beforeActionTestPresentation,
	})

	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) ([]map[string]any, error) {
			ctx = workflow.WithActivityOptions(
				ctx,
				workflow.ActivityOptions{StartToCloseTimeout: time.Minute},
			)
			beforeActions := func(deeplink string) []any {
				return []any{map[string]any{
					"action_id":  "tenant/issue",
					"parameters": map[string]any{"deeplink": deeplink},
				}}
			}
			steps := []pipeline.StepDefinition{
				{StepSpec: pipeline.StepSpec{
					ID:  "stored",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{Payload: map[string]any{
						"action_id":      "tenant/present",
						"version_id":     mobileExternalSourceVersionID,
						"parameters":     map[string]any{"deeplink": "${{ a.outputs.deeplink }}"},
						"before_actions": beforeActions("${{ offer-a.outputs.deeplink }}"),
					}},
				}},
				{StepSpec: pipeline.StepSpec{
					ID:  "inline",
					Use: mobileAutomationStepUse,
					With: pipeline.StepInputs{Payload: map[string]any{
						"action_code":    beforeActionTestPresentation,
						"version_id":     mobileExternalSourceVersionID,
						"before_actions": beforeActions("${{ offer-b.outputs.deeplink }}"),
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
		workflow.RegisterOptions{Name: "test-compose-before-actions"},
	)

	env.ExecuteWorkflow("test-compose-before-actions")
	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)

	var payloads []map[string]any
	require.NoError(t, env.GetWorkflowResult(&payloads))
	require.Equal(t, map[string]any{
		"deeplink":          "${{ a.outputs.deeplink }}",
		"before_0_deeplink": "${{ offer-a.outputs.deeplink }}",
	}, payloads[0]["parameters"])
	require.Equal(t, map[string]any{
		"before_0_deeplink": "${{ offer-b.outputs.deeplink }}",
	}, payloads[1]["parameters"])
	for _, payload := range payloads {
		require.NotContains(t, payload, mobileBeforeActionsPayloadKey)
		_, commands := decodeComposedFlow(t, payload["action_code"].(string))
		require.Len(t, commands, 4)
		require.Contains(t, commands[0], "runFlow")
	}
	require.Equal(t, true, payloads[0]["stored_action_code"])
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
