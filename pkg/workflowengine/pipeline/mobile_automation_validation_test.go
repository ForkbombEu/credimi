// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/stretchr/testify/require"
)

func TestValidateMobileDeviceIDConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		steps          []pipeline.StepDefinition
		globalDeviceID string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "no mobile-automation steps - should pass",
			steps:          []pipeline.StepDefinition{},
			globalDeviceID: "",
			expectError:    false,
		},
		{
			name: "all steps have device_id - should pass",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step1",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"device_id": "runner1/device1",
								"action_id": "action1",
							},
						},
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step2",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"device_id": "runner2/device2",
								"action_id": "action2",
							},
						},
					},
				},
			},
			globalDeviceID: "",
			expectError:    false,
		},
		{
			name: "no step-level device_id but global_device_id is set - should pass",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step1",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action1",
							},
						},
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step2",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action2",
							},
						},
					},
				},
			},
			globalDeviceID: "global-runner/device",
			expectError:    false,
		},
		{
			name: "some steps missing device_id and no global_device_id - should fail",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step1",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"device_id": "runner1/device1",
								"action_id": "action1",
							},
						},
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step2",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action2",
							},
						},
					},
				},
			},
			globalDeviceID: "",
			expectError:    true,
			errorContains:  "device_id",
		},
		{
			name: "no device_id anywhere - should fail",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step1",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action1",
							},
						},
					},
				},
			},
			globalDeviceID: "",
			expectError:    true,
			errorContains:  "device_id",
		},
		{
			name: "mixed step types - mobile-automation without device_id but has global - should pass",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step1",
						Use: "echo",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"message": "hello",
							},
						},
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step2",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action1",
							},
						},
					},
				},
			},
			globalDeviceID: "global-runner/device",
			expectError:    false,
		},
		{
			name: "some steps with device_id, some without, with global_device_id - should fail",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step1",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"device_id": "specific-runner/device",
								"action_id": "action1",
							},
						},
					},
				},
				{
					StepSpec: pipeline.StepSpec{
						ID:  "step2",
						Use: "mobile-automation",
						With: pipeline.StepInputs{
							Payload: map[string]any{
								"action_id": "action2",
							},
						},
					},
				},
			},
			globalDeviceID: "global-runner/device",
			expectError:    true,
			errorContains:  "global_device_id is set",
		},
		{
			name: "nested on_error step without device_id should fail",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{ID: "parent", Use: "echo"},
					OnError: []*pipeline.OnErrorStepDefinition{
						{StepSpec: pipeline.StepSpec{ID: "nested-error", Use: "mobile-automation"}},
					},
				},
			},
			globalDeviceID: "",
			expectError:    true,
			errorContains:  "nested-error",
		},
		{
			name: "nested on_success step with device_id should pass",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{ID: "parent", Use: "echo"},
					OnSuccess: []*pipeline.OnSuccessStepDefinition{
						{StepSpec: pipeline.StepSpec{
							ID:  "nested-success",
							Use: mobileAutomationStepUse,
							With: pipeline.StepInputs{Payload: map[string]any{
								"device_id": " /tenant/runner/device ",
							}},
						}},
					},
				},
			},
			globalDeviceID: "",
			expectError:    false,
		},
		{
			name: "nested step with device_id conflicts with global_device_id",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{ID: "parent", Use: "echo"},
					OnError: []*pipeline.OnErrorStepDefinition{
						{StepSpec: pipeline.StepSpec{
							ID:  "nested-error",
							Use: mobileAutomationStepUse,
							With: pipeline.StepInputs{Payload: map[string]any{
								"device_id": "runner/device",
							}},
						}},
					},
				},
			},
			globalDeviceID: "global/runner/device",
			expectError:    true,
			errorContains:  "nested-error",
		},
		{
			name: "nested non-mobile step does not require device_id",
			steps: []pipeline.StepDefinition{
				{
					StepSpec: pipeline.StepSpec{ID: "parent", Use: "echo"},
					OnSuccess: []*pipeline.OnSuccessStepDefinition{
						{StepSpec: pipeline.StepSpec{ID: "nested-success", Use: "echo"}},
					},
				},
			},
			globalDeviceID: " /global/runner/device ",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMobileDeviceIDConfiguration(tt.steps, tt.globalDeviceID)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWorkflowDefinition_GlobalDeviceID(t *testing.T) {
	t.Run("parse workflow with global_device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  global_device_id: my-global-runner/device
steps:
  - id: step1
    use: mobile-automation
    with:
      action_id: action1
`
		wfDef, err := pipeline.ParseWorkflow(yamlContent)
		require.NoError(t, err)
		require.Equal(t, "my-global-runner/device", wfDef.Runtime.GlobalDeviceID)
		require.Equal(t, "Test Pipeline", wfDef.Name)
	})

	t.Run("parse workflow without global_device_id", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: mobile-automation
    with:
      device_id: step-runner/device
      action_id: action1
`
		wfDef, err := pipeline.ParseWorkflow(yamlContent)
		require.NoError(t, err)
		require.Equal(t, "", wfDef.Runtime.GlobalDeviceID)
		require.Equal(t, "Test Pipeline", wfDef.Name)
	})

	t.Run("parse workflow with disable_android_play_store", func(t *testing.T) {
		yamlContent := `
name: Test Pipeline
runtime:
  disable_android_play_store: true
steps:
  - id: step1
    use: mobile-automation
    with:
      device_id: step-runner/device
      action_id: action1
`
		wfDef, err := pipeline.ParseWorkflow(yamlContent)
		require.NoError(t, err)
		require.True(t, wfDef.Runtime.DisableAndroidPlayStore)
	})
}
