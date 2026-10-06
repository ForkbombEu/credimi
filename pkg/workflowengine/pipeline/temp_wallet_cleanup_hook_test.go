// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"errors"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

var tempWalletVersionDeleteInput = activities.DeleteTempRecordInput{
	Collection:         "wallet_versions",
	RecordID:           "version-1",
	ExpectedOwnerID:    "owner-1",
	ExpectedIdentifier: "org/wallet/sha",
}

func TestTempWalletVersionCleanupHookSkipsWhenConfigAbsent(t *testing.T) {
	var ctx workflow.Context
	var ao workflow.ActivityOptions
	output := map[string]any{}

	err := tempWalletVersionCleanupHook(
		ctx,
		nil,
		&ao,
		map[string]any{},
		nil,
		&output,
	)

	require.NoError(t, err)
}

func TestTempWalletVersionCleanupHookDeletesTempRecord(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	registerDeleteTempRecordActivity(env)
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			return tempWalletVersionCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{
					tempWalletVersionConfigKey: map[string]any{
						"record_id":  "version-1",
						"owner_id":   "owner-1",
						"identifier": "org/wallet/sha",
						"cleanup":    true,
					},
				},
				nil,
				nil,
			)
		},
		workflow.RegisterOptions{Name: "test-temp-wallet-cleanup"},
	)

	env.OnActivity(
		activities.DeleteTempRecordActivityName,
		mock.Anything,
		deleteTempRecordInputMatcher(tempWalletVersionDeleteInput),
	).Return(workflowengine.ActivityResult{}, nil).Once()

	env.ExecuteWorkflow("test-temp-wallet-cleanup")

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestPipelineTempWalletCleanupRunsAfterSetupFailure(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	pipelineWf := NewPipelineWorkflow()
	env.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	registerDeleteTempRecordActivity(env)

	originalSetupHooks := setupHooks
	originalCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{
		func(
			_ workflow.Context,
			_ *pipelineinternal.WorkflowDefinition,
			_ map[string]any,
			_ *map[string]any,
			_ *map[string]any,
			_ log.Logger,
		) error {
			return errors.New("setup failed")
		},
	}
	cleanupHooks = []CleanupFunc{tempWalletVersionCleanupHook}
	t.Cleanup(func() {
		setupHooks = originalSetupHooks
		cleanupHooks = originalCleanupHooks
	})

	env.OnActivity(
		activities.DeleteTempRecordActivityName,
		mock.Anything,
		deleteTempRecordInputMatcher(tempWalletVersionDeleteInput),
	).Return(workflowengine.ActivityResult{}, nil).Once()

	input := PipelineWorkflowInput{
		WorkflowDefinition: &pipelineinternal.WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []pipelineinternal.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				tempWalletVersionConfigKey: map[string]any{
					"record_id":  "version-1",
					"owner_id":   "owner-1",
					"identifier": "org/wallet/sha",
					"cleanup":    true,
				},
			},
			ActivityOptions: &workflow.ActivityOptions{StartToCloseTimeout: time.Second},
		},
	}

	env.ExecuteWorkflow(pipelineWf.Name(), input)

	require.Error(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
