// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pipeline

import (
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func registerDeleteTempRecordActivity(env *testsuite.TestWorkflowEnvironment) {
	registerStubActivity(env, activities.DeleteTempRecordActivityName)
}

func deleteTempRecordInputMatcher(want activities.DeleteTempRecordInput) any {
	return mock.MatchedBy(func(input workflowengine.ActivityInput) bool {
		payload, err := workflowengine.DecodePayload[activities.DeleteTempRecordInput](
			input.Payload,
		)
		return err == nil && payload == want
	})
}

func TestTempCredentialsCleanupHookSkipsWhenConfigAbsent(t *testing.T) {
	var ctx workflow.Context
	var ao workflow.ActivityOptions
	output := map[string]any{}

	err := tempCredentialsCleanupHook(ctx, nil, &ao, map[string]any{}, nil, &output)

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

func TestTempRecordCleanupHooksDeleteTempRecords(t *testing.T) {
	cases := []struct {
		name      string
		hook      CleanupFunc
		configKey string
		itemsKey  string
		items     []any
		want      []activities.DeleteTempRecordInput
		failFirst bool
		wantErr   bool
	}{
		{
			name:      "credentials",
			hook:      tempCredentialsCleanupHook,
			configKey: tempCredentialsConfigKey,
			itemsKey:  "credentials",
			items: []any{
				map[string]any{
					"record_id":  "credential-1",
					"owner_id":   "owner-1",
					"identifier": "org/issuer/pid-sha",
				},
				map[string]any{"owner_id": "owner-1"},
			},
			want: []activities.DeleteTempRecordInput{{
				Collection:         "credentials",
				RecordID:           "credential-1",
				ExpectedOwnerID:    "owner-1",
				ExpectedIdentifier: "org/issuer/pid-sha",
			}},
		},
		{
			name:      "use case verifications",
			hook:      tempUseCaseVerificationsCleanupHook,
			configKey: tempUseCaseVerificationsConfigKey,
			itemsKey:  "use_cases",
			items: []any{
				map[string]any{
					"record_id":  "use-case-1",
					"owner_id":   "owner-1",
					"identifier": "org/verifier/pid-sha",
				},
			},
			want: []activities.DeleteTempRecordInput{{
				Collection:         "use_cases_verifications",
				RecordID:           "use-case-1",
				ExpectedOwnerID:    "owner-1",
				ExpectedIdentifier: "org/verifier/pid-sha",
			}},
		},
		{
			name:      "first error stops the cleanup",
			hook:      tempCredentialsCleanupHook,
			configKey: tempCredentialsConfigKey,
			itemsKey:  "credentials",
			items: []any{
				map[string]any{"record_id": "credential-1", "owner_id": "o", "identifier": "i1"},
				map[string]any{"record_id": "credential-2", "owner_id": "o", "identifier": "i2"},
			},
			want: []activities.DeleteTempRecordInput{{
				Collection:         "credentials",
				RecordID:           "credential-1",
				ExpectedOwnerID:    "o",
				ExpectedIdentifier: "i1",
			}},
			failFirst: true,
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			suite := testsuite.WorkflowTestSuite{}
			env := suite.NewTestWorkflowEnvironment()
			registerDeleteTempRecordActivity(env)
			env.RegisterWorkflowWithOptions(
				func(ctx workflow.Context) error {
					ao := workflow.ActivityOptions{StartToCloseTimeout: time.Second}
					ctx = workflow.WithActivityOptions(ctx, ao)
					return tc.hook(
						ctx,
						nil,
						&ao,
						map[string]any{
							tc.configKey: map[string]any{
								tc.itemsKey: tc.items,
								"cleanup":   true,
							},
						},
						nil,
						nil,
					)
				},
				workflow.RegisterOptions{Name: "test-temp-record-cleanup"},
			)

			for _, want := range tc.want {
				var activityErr error
				if tc.failFirst {
					activityErr = temporal.NewNonRetryableApplicationError(
						"delete failed",
						"test",
						nil,
					)
				}
				env.OnActivity(
					activities.DeleteTempRecordActivityName,
					mock.Anything,
					deleteTempRecordInputMatcher(want),
				).Return(workflowengine.ActivityResult{}, activityErr).Once()
			}

			env.ExecuteWorkflow("test-temp-record-cleanup")

			if tc.wantErr {
				require.Error(t, env.GetWorkflowError())
			} else {
				require.NoError(t, env.GetWorkflowError())
			}
			env.AssertExpectations(t)
		})
	}
}
