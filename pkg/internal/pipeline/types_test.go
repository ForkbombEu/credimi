// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowDefinitionJSONOmitsEmptyObjects(t *testing.T) {
	empty, err := json.Marshal(WorkflowDefinition{
		Name:  "minimal",
		Steps: []StepDefinition{{StepSpec: StepSpec{ID: "a", Use: "credential-offer"}}},
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"minimal","steps":[{"id":"a","use":"credential-offer","with":{}}]}`,
		string(empty))

	def := WorkflowDefinition{Name: "set"}
	def.Runtime.Temporal.ActivityOptions.RetryPolicy.MaximumAttempts = 1
	def.Finally.Always = []FinallyStepDefinition{{StepSpec: StepSpec{ID: "f", Use: "email"}}}
	set, err := json.Marshal(def)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"set",
		"runtime":{"temporal":{"activity_options":{"retry_policy":{"maximum_attempts":1}}}},
		"finally":{"always":[{"id":"f","use":"email","with":{}}]}}`, string(set))
}

func TestFinallyDefinitionHelpers(t *testing.T) {
	empty := FinallyDefinition{}
	require.True(t, empty.IsZero())
	require.Empty(t, empty.AllSteps())

	definition := FinallyDefinition{
		Always:    []FinallyStepDefinition{{StepSpec: StepSpec{ID: "always"}}},
		OnSuccess: []FinallyStepDefinition{{StepSpec: StepSpec{ID: "success"}}},
		OnFailure: []FinallyStepDefinition{{StepSpec: StepSpec{ID: "failure"}}},
	}
	require.False(t, definition.IsZero())
	require.Equal(t, []string{"always", "success", "failure"}, []string{
		definition.AllSteps()[0].ID,
		definition.AllSteps()[1].ID,
		definition.AllSteps()[2].ID,
	})
}

func TestValidRunType(t *testing.T) {
	require.True(t, ValidRunType(RunTypeManual))
	require.True(t, ValidRunType(RunTypeScheduled))
	require.True(t, ValidRunType(RunTypeCI))
	require.False(t, ValidRunType("unknown"))
}
