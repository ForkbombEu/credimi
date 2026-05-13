// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestTempCredentialsCleanupHookSkipsWhenConfigAbsent(t *testing.T) {
	var ctx workflow.Context
	var ao workflow.ActivityOptions
	output := map[string]any{}

	err := tempCredentialsCleanupHook(
		ctx,
		nil,
		&ao,
		map[string]any{"app_url": "https://example.test"},
		nil,
		&output,
	)

	require.NoError(t, err)
}

func TestTempCredentialsCleanupHookNormalizesCleanupItems(t *testing.T) {
	credentials := normalizeTempCredentialCleanupItems([]any{
		map[string]any{
			"record_id":  "credential-1",
			"owner_id":   "owner-1",
			"identifier": "org/issuer/pid-sha",
		},
		"ignored",
	})

	require.Len(t, credentials, 1)
	require.Equal(t, "credential-1", credentials[0]["record_id"])
}

func TestTempCredentialsCleanupHookCallsInternalDelete(t *testing.T) {
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
			return tempCredentialsCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{
					"app_url": "https://example.test",
					tempCredentialsConfigKey: map[string]any{
						"credentials": []any{
							map[string]any{
								"record_id":  "credential-1",
								"owner_id":   "owner-1",
								"identifier": "org/issuer/pid-sha",
							},
						},
						"cleanup": true,
					},
				},
				nil,
				nil,
			)
		},
		workflow.RegisterOptions{Name: "test-temp-credentials-cleanup"},
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
				payload.URL == "https://example.test/api/credential/temp/credential-1" &&
				body["expected_owner_id"] == "owner-1" &&
				body["expected_identifier"] == "org/issuer/pid-sha" &&
				payload.ExpectedStatus == http.StatusOK
		}),
	).Return(workflowengine.ActivityResult{}, nil).Once()

	env.ExecuteWorkflow("test-temp-credentials-cleanup")

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}

func TestTempUseCaseVerificationsCleanupHookCallsInternalDelete(t *testing.T) {
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
			return tempUseCaseVerificationsCleanupHook(
				ctx,
				nil,
				&ao,
				map[string]any{
					"app_url": "https://example.test",
					tempUseCaseVerificationsConfigKey: map[string]any{
						"use_cases": []any{
							map[string]any{
								"record_id":  "use-case-1",
								"owner_id":   "owner-1",
								"identifier": "org/verifier/pid-sha",
							},
						},
						"cleanup": true,
					},
				},
				nil,
				nil,
			)
		},
		workflow.RegisterOptions{Name: "test-temp-use-case-verifications-cleanup"},
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
				payload.URL == "https://example.test/api/verifier/temp-use-case/use-case-1" &&
				body["expected_owner_id"] == "owner-1" &&
				body["expected_identifier"] == "org/verifier/pid-sha" &&
				payload.ExpectedStatus == http.StatusOK
		}),
	).Return(workflowengine.ActivityResult{}, nil).Once()

	env.ExecuteWorkflow("test-temp-use-case-verifications-cleanup")

	require.NoError(t, env.GetWorkflowError())
	env.AssertExpectations(t)
}
