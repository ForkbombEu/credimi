// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func Test_DebugActivity_Execute(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	act := NewDebugActivity()
	env.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
		Name: act.Name(),
	})

	stepID := "step-123"
	outputs := map[string]any{
		"foo": "bar",
		"baz": float64(42),
	}

	input := workflowengine.ActivityInput{
		Payload: map[string]any{
			"step":    stepID,
			"outputs": outputs,
		},
	}

	var result workflowengine.ActivityResult
	value, err := env.ExecuteActivity(act.Execute, input)
	require.NoError(t, err)
	require.NoError(t, value.Get(&result))

	require.NotNil(t, result.Output)
	require.Equal(t, stepID, result.Output.(map[string]any)["current_step"])
	require.Equal(t, outputs, result.Output.(map[string]any)["outputs"])
}
