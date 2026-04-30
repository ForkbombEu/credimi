// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	pipelineinternal "github.com/forkbombeu/credimi/pkg/internal/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestTempWalletVersionCleanupHookSkipsWhenConfigAbsent(t *testing.T) {
	var ctx workflow.Context
	var ao workflow.ActivityOptions
	output := map[string]any{}

	err := tempWalletVersionCleanupHook(
		ctx,
		nil,
		&ao,
		map[string]any{"app_url": "https://example.test"},
		nil,
		&output,
	)

	require.NoError(t, err)
}

func TestTempWalletVersionCleanupHookCallsInternalDelete(t *testing.T) {
	suite := testsuite.WorkflowTestSuite{}
	env := suite.NewTestWorkflowEnvironment()

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	env.RegisterActivityWithOptions(
		func(
			ctx context.Context,
			input workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{}, nil
		},
		activity.RegisterOptions{Name: internalHTTPActivity.Name()},
	)
	env.RegisterWorkflowWithOptions(
		func(ctx workflow.Context) error {
			ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
			ctx = workflow.WithActivityOptions(ctx, ao)
			return tempWalletVersionCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{
					"app_url": "https://example.test",
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
		internalHTTPActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(activities.InternalHTTPActivityPayload)
			if !ok {
				decoded, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			body, ok := payload.Body.(map[string]any)
			if !ok {
				return false
			}
			return ok &&
				payload.Method == http.MethodDelete &&
				payload.URL == "https://example.test/api/wallet/temp-version/version-1" &&
				body["expected_owner_id"] == "owner-1" &&
				body["expected_identifier"] == "org/wallet/sha" &&
				payload.ExpectedStatus == http.StatusOK
		}),
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

	internalHTTPActivity := activities.NewInternalHTTPActivity()
	env.RegisterActivityWithOptions(
		func(
			ctx context.Context,
			input workflowengine.ActivityInput,
		) (workflowengine.ActivityResult, error) {
			return workflowengine.ActivityResult{}, nil
		},
		activity.RegisterOptions{Name: internalHTTPActivity.Name()},
	)

	originalSetupHooks := setupHooks
	originalCleanupHooks := cleanupHooks
	setupHooks = []SetupFunc{
		func(
			_ workflow.Context,
			_ *[]pipelineinternal.StepDefinition,
			_ *workflow.ActivityOptions,
			_ map[string]any,
			_ *map[string]any,
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
		internalHTTPActivity.Name(),
		mock.Anything,
		mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
			payload, ok := input.Payload.(activities.InternalHTTPActivityPayload)
			if !ok {
				decoded, err := workflowengine.DecodePayload[activities.InternalHTTPActivityPayload](
					input.Payload,
				)
				if err != nil {
					return false
				}
				payload = decoded
			}
			body, ok := payload.Body.(map[string]any)
			if !ok {
				return false
			}
			return ok &&
				payload.Method == http.MethodDelete &&
				payload.URL == "https://example.test/api/wallet/temp-version/version-1" &&
				body["expected_owner_id"] == "owner-1" &&
				body["expected_identifier"] == "org/wallet/sha"
		}),
	).Return(workflowengine.ActivityResult{}, nil).Once()

	input := PipelineWorkflowInput{
		WorkflowDefinition: &pipelineinternal.WorkflowDefinition{
			Name:  "test-pipeline",
			Steps: []pipelineinternal.StepDefinition{},
		},
		WorkflowInput: workflowengine.WorkflowInput{
			Config: map[string]any{
				"app_url": "https://example.test",
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
