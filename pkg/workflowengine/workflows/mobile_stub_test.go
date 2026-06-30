// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflows

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestMobileAutomationWorkflowDisabled(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	w := NewMobileAutomationWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
		Name: w.Name(),
	})

	input := workflowengine.WorkflowInput{
		RunMetadata: &workflowengine.WorkflowRunMetadata{WorkflowName: w.Name()},
		Config:      map[string]any{"app_url": ""},
	}
	env.ExecuteWorkflow(w.Name(), input)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code)
	failure := workflowengine.ParseWorkflowError(err)
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code, failure.Code)
	require.Equal(
		t,
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Description,
		failure.Summary,
	)
	require.Equal(
		t,
		"mobile automation is disabled; build with -tags=credimi_extra",
		failure.Message,
	)
}

func TestMobileExternalInstallWorkflowDisabled(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	w := NewMobileExternalInstallWorkflow()
	env.RegisterWorkflowWithOptions(w.Workflow, workflow.RegisterOptions{
		Name: w.Name(),
	})

	input := workflowengine.WorkflowInput{
		RunMetadata: &workflowengine.WorkflowRunMetadata{WorkflowName: w.Name()},
		Config:      map[string]any{"app_url": ""},
	}
	env.ExecuteWorkflow(w.Name(), input)

	err := env.GetWorkflowError()
	require.Error(t, err)
	require.Contains(t, err.Error(), errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code)
	failure := workflowengine.ParseWorkflowError(err)
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code, failure.Code)
	require.Equal(
		t,
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Description,
		failure.Summary,
	)
	require.Equal(
		t,
		"mobile automation is disabled; build with -tags=credimi_extra",
		failure.Message,
	)
}
