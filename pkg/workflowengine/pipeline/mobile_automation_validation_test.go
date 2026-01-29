// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
"testing"

"github.com/stretchr/testify/require"
)

func TestValidateRunnerIDConfiguration(t *testing.T) {
tests := []struct {
name           string
steps          []StepDefinition
globalRunnerID string
expectError    bool
errorContains  string
}{
{
name:           "no mobile-automation steps - should pass",
steps:          []StepDefinition{},
globalRunnerID: "",
expectError:    false,
},
{
name: "all steps have runner_id - should pass",
steps: []StepDefinition{
{
StepSpec: StepSpec{
ID:  "step1",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"runner_id": "runner1",
"action_id": "action1",
},
},
},
},
{
StepSpec: StepSpec{
ID:  "step2",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"runner_id": "runner2",
"action_id": "action2",
},
},
},
},
},
globalRunnerID: "",
expectError:    false,
},
{
name: "no step-level runner_id but global_runner_id is set - should pass",
steps: []StepDefinition{
{
StepSpec: StepSpec{
ID:  "step1",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"action_id": "action1",
},
},
},
},
{
StepSpec: StepSpec{
ID:  "step2",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"action_id": "action2",
},
},
},
},
},
globalRunnerID: "global-runner",
expectError:    false,
},
{
name: "some steps missing runner_id and no global_runner_id - should fail",
steps: []StepDefinition{
{
StepSpec: StepSpec{
ID:  "step1",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"runner_id": "runner1",
"action_id": "action1",
},
},
},
},
{
StepSpec: StepSpec{
ID:  "step2",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"action_id": "action2",
},
},
},
},
},
globalRunnerID: "",
expectError:    true,
errorContains:  "runner_id",
},
{
name: "no runner_id anywhere - should fail",
steps: []StepDefinition{
{
StepSpec: StepSpec{
ID:  "step1",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"action_id": "action1",
},
},
},
},
},
globalRunnerID: "",
expectError:    true,
errorContains:  "runner_id",
},
{
name: "mixed step types - mobile-automation without runner_id but has global - should pass",
steps: []StepDefinition{
{
StepSpec: StepSpec{
ID:  "step1",
Use: "echo",
With: StepInputs{
Payload: map[string]any{
"message": "hello",
},
},
},
},
{
StepSpec: StepSpec{
ID:  "step2",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"action_id": "action1",
},
},
},
},
},
globalRunnerID: "global-runner",
expectError:    false,
},
{
name: "some steps with runner_id, some without, with global_runner_id - should pass",
steps: []StepDefinition{
{
StepSpec: StepSpec{
ID:  "step1",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"runner_id": "specific-runner",
"action_id": "action1",
},
},
},
},
{
StepSpec: StepSpec{
ID:  "step2",
Use: "mobile-automation",
With: StepInputs{
Payload: map[string]any{
"action_id": "action2",
},
},
},
},
},
globalRunnerID: "global-runner",
expectError:    false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := validateRunnerIDConfiguration(&tt.steps, tt.globalRunnerID)

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

func TestWorkflowDefinition_GlobalRunnerID(t *testing.T) {
t.Run("parse workflow with global_runner_id", func(t *testing.T) {
yamlContent := `
name: Test Pipeline
global_runner_id: my-global-runner
steps:
  - id: step1
    use: mobile-automation
    with:
      action_id: action1
`
wfDef, err := ParseWorkflow(yamlContent)
require.NoError(t, err)
require.Equal(t, "my-global-runner", wfDef.GlobalRunnerID)
require.Equal(t, "Test Pipeline", wfDef.Name)
})

t.Run("parse workflow without global_runner_id", func(t *testing.T) {
yamlContent := `
name: Test Pipeline
steps:
  - id: step1
    use: mobile-automation
    with:
      runner_id: step-runner
      action_id: action1
`
wfDef, err := ParseWorkflow(yamlContent)
require.NoError(t, err)
require.Equal(t, "", wfDef.GlobalRunnerID)
require.Equal(t, "Test Pipeline", wfDef.Name)
})
}
