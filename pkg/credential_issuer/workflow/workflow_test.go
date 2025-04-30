// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func Test_SuccessfulFetchIssuersWorkflows(t *testing.T) {
	t.Skip("Skipping test for now, as it requires a real database connection")
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	issuers := []string{"issuer1", "issuer2", "issuer3"}

	env.OnActivity(FetchIssuersActivity, mock.Anything).Return(FetchIssuersActivityResponse{
		Issuers: issuers,
	}, nil)
	env.OnActivity(CreateCredentialIssuersActivity, mock.Anything, CreateCredentialIssuersInput{
		Issuers: issuers,
		DBPath:  mock.Anything,
	}).Return(nil)
	env.ExecuteWorkflow(FetchIssuersWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
}

func Test_UnsuccessfulFetchIssuersWorkflows(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(FetchIssuersActivity, mock.Anything).
		Return(FetchIssuersActivityResponse{}, errors.New("error"))
	env.ExecuteWorkflow(FetchIssuersWorkflow)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}
