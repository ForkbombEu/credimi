// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

func TestWebuildWorkflowStart(t *testing.T) {
	origStart := webuildStartWorkflowWithOptions
	t.Cleanup(func() {
		webuildStartWorkflowWithOptions = origStart
	})

	var capturedNamespace string
	var capturedOptions client.StartWorkflowOptions
	var capturedName string
	var capturedInput workflowengine.WorkflowInput

	webuildStartWorkflowWithOptions = func(
		namespace string,
		options client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		capturedNamespace = namespace
		capturedOptions = options
		capturedName = name
		capturedInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-2", WorkflowRunID: "run-2"}, nil
	}

	w := NewWebuildWorkflow()
	input := workflowengine.WorkflowInput{
		Config: map[string]any{
			"namespace": "ns-2",
		},
	}
	result, err := w.Start(input)
	require.NoError(t, err)
	require.Equal(t, "wf-2", result.WorkflowID)
	require.Equal(t, "run-2", result.WorkflowRunID)
	require.Equal(t, "ns-2", capturedNamespace)
	require.Equal(t, w.Name(), capturedName)
	require.Equal(t, input, capturedInput)
	require.Equal(t, EWCTaskQueue, capturedOptions.TaskQueue)
	require.True(t, strings.HasPrefix(capturedOptions.ID, "WebuildWorkflow"))
	require.Equal(t, 24*time.Hour, capturedOptions.WorkflowExecutionTimeout)
}
