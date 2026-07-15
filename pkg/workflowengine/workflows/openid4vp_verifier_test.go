// SPDX-FileCopyrightText: 2026 Forkbomb BV
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

func TestOpenID4VPVerifierWorkflowStart(t *testing.T) {
	originalStart := openID4VPVerifierStartWorkflowWithOptions
	t.Cleanup(func() { openID4VPVerifierStartWorkflowWithOptions = originalStart })

	var namespace string
	var options client.StartWorkflowOptions
	var workflowName string
	var workflowInput workflowengine.WorkflowInput
	openID4VPVerifierStartWorkflowWithOptions = func(
		ns string,
		opts client.StartWorkflowOptions,
		name string,
		input workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		namespace = ns
		options = opts
		workflowName = name
		workflowInput = input
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}

	workflow := NewOpenID4VPVerifierWorkflow()
	input := workflowengine.WorkflowInput{Config: map[string]any{"namespace": "tenant"}}
	result, err := workflow.Start(input)
	require.NoError(t, err)
	require.Equal(t, "wf-1", result.WorkflowID)
	require.Equal(t, "run-1", result.WorkflowRunID)
	require.Equal(t, "tenant", namespace)
	require.Equal(t, workflow.Name(), workflowName)
	require.Equal(t, input, workflowInput)
	require.Equal(t, OpenID4VPVerifierTaskQueue, options.TaskQueue)
	require.True(t, strings.HasPrefix(options.ID, "OpenID4VPVerifierCheckWorkflow"))
	require.Equal(t, 24*time.Hour, options.WorkflowExecutionTimeout)
	require.Equal(t, openIDConformanceActivityOptions, workflow.GetOptions())
}

func TestOpenID4VPVerifierWorkflowStartUsesDefaultNamespace(t *testing.T) {
	originalStart := openID4VPVerifierStartWorkflowWithOptions
	t.Cleanup(func() { openID4VPVerifierStartWorkflowWithOptions = originalStart })

	var namespace string
	openID4VPVerifierStartWorkflowWithOptions = func(
		ns string,
		_ client.StartWorkflowOptions,
		_ string,
		_ workflowengine.WorkflowInput,
	) (workflowengine.WorkflowResult, error) {
		namespace = ns
		return workflowengine.WorkflowResult{}, nil
	}

	_, err := NewOpenID4VPVerifierWorkflow().Start(workflowengine.WorkflowInput{Config: map[string]any{}})
	require.NoError(t, err)
	require.Equal(t, DefaultNamespace, namespace)
}
