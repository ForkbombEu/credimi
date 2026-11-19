// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package runnerworkflows

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesession"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	temporalmocks "go.temporal.io/sdk/mocks"
)

const testDataDir = "../../../test_pb_data"

type fixture struct {
	app         *tests.TestApp
	orgID       string
	otherOrgID  string
	runnerID    string
	otherRunner string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)

	orgID := userOrganization(t, app, "userA@example.org")
	otherOrgID := userOrganization(t, app, "userB@example.org")
	runner := saveRecord(t, app, "mobile_runners", map[string]any{
		"name":  "pipeline-runner",
		"owner": orgID,
		"ip":    "my_ip",
		"type":  "android_emulator",
	})
	otherRunner := saveRecord(t, app, "mobile_runners", map[string]any{
		"name":  "other-runner",
		"owner": orgID,
		"ip":    "my_ip",
		"type":  "android_emulator",
	})
	device := saveRecord(t, app, "mobile_devices", map[string]any{
		"name":            "pixel",
		"canonified_name": "pixel",
		"owner":           orgID,
		"runner":          runner.Id,
		"type":            "android_emulator",
		"serial":          "pixel-serial",
	})
	pipeline := saveRecord(t, app, "pipelines", map[string]any{
		"owner":           orgID,
		"name":            "pipeline123",
		"canonified_name": "pipeline123",
		"description":     "demo pipeline",
		"yaml":            "name: demo",
	})
	saveRecord(t, app, "pipeline_results", map[string]any{
		"owner":       orgID,
		"pipeline":    pipeline.Id,
		"workflow_id": "pipeline-1",
		"run_id":      "run-1",
		"devices":     []string{device.Id},
	})
	return fixture{
		app:         app,
		orgID:       orgID,
		otherOrgID:  otherOrgID,
		runnerID:    runner.Id,
		otherRunner: otherRunner.Id,
	}
}

func userOrganization(t *testing.T, app core.App, email string) string {
	t.Helper()
	user, err := app.FindAuthRecordByEmail("users", email)
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, user.Id)
	require.NoError(t, err)
	return orgID
}

func saveRecord(t *testing.T, app core.App, collection string, fields map[string]any) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId(collection)
	require.NoError(t, err)
	record := core.NewRecord(coll)
	for key, value := range fields {
		record.Set(key, value)
	}
	require.NoError(t, app.Save(record))
	return record
}

func description(
	t *testing.T,
	workflowType string,
	pipelineRun *mobiledevicesession.PipelineRunRef,
) *workflowservice.DescribeWorkflowExecutionResponse {
	t.Helper()
	info := &workflow.WorkflowExecutionInfo{
		Execution: &common.WorkflowExecution{WorkflowId: "pipeline-1-step", RunId: "step-run"},
		Type:      &common.WorkflowType{Name: workflowType},
		Memo:      &common.Memo{Fields: map[string]*common.Payload{}},
	}
	if pipelineRun != nil {
		payload, err := temporalcrypto.DataConverter().ToPayload(pipelineRun)
		require.NoError(t, err)
		info.Memo.Fields[mobiledevicesession.PipelineRunMemoKey] = payload
	}
	return &workflowservice.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: info}
}

func TestOwnedBy(t *testing.T) {
	ownRun := &mobiledevicesession.PipelineRunRef{
		Namespace:  "acme",
		WorkflowID: "pipeline-1",
		RunID:      "run-1",
	}
	cases := []struct {
		name      string
		otherOrg  bool
		runner    func(f fixture) string
		namespace func(f fixture) string
		describe  func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error)
		want      bool
		wantErr   bool
	}{
		{
			name: "step of the organization's pipeline run",
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(t, mobiledevicesession.MobileAutomationWorkflowName, ownRun), nil
			},
			want: true,
		},
		{
			name: "device session of the organization's pipeline run",
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(t, mobiledevicesession.WorkflowName, ownRun), nil
			},
			want: true,
		},
		{
			name:     "another organization",
			otherOrg: true,
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(t, mobiledevicesession.WorkflowName, ownRun), nil
			},
		},
		{
			name:   "runner that hosted none of the run's devices",
			runner: func(f fixture) string { return f.otherRunner },
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(t, mobiledevicesession.WorkflowName, ownRun), nil
			},
		},
		{
			name: "other workflow type",
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(t, "Action author session", ownRun), nil
			},
		},
		{
			name: "no memo",
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(t, mobiledevicesession.WorkflowName, nil), nil
			},
		},
		{
			name: "memo naming an unknown pipeline run",
			describe: func(t *testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return description(
					t,
					mobiledevicesession.WorkflowName,
					&mobiledevicesession.PipelineRunRef{
						Namespace:  "acme",
						WorkflowID: "pipeline-1",
						RunID:      "run-2",
					},
				), nil
			},
		},
		{
			name:      "organization namespace",
			namespace: func(fixture) string { return "acme" },
		},
		{
			name:      "bare runner prefix",
			namespace: func(fixture) string { return canonify.RunnerNamespacePrefix },
		},
		{
			name: "workflow not found",
			describe: func(*testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return nil, serviceerror.NewNotFound("workflow not found")
			},
		},
		{
			name: "describe failure",
			describe: func(*testing.T) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
				return nil, errors.New("unavailable")
			},
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owned.Clear()
			f := newFixture(t)
			runnerID := f.runnerID
			if tc.runner != nil {
				runnerID = tc.runner(f)
			}
			namespace := canonify.RunnerNamespacePrefix + runnerID
			if tc.namespace != nil {
				namespace = tc.namespace(f)
			}
			orgID := f.orgID
			if tc.otherOrg {
				orgID = f.otherOrgID
			}
			c := temporalmocks.NewClient(t)
			if tc.describe != nil {
				resp, err := tc.describe(t)
				c.On("DescribeWorkflow", mock.Anything, "pipeline-1-step", "step-run").
					Return(temporalclient.Description(resp), err).Once()
			}

			got, err := OwnedBy(
				context.Background(),
				f.app,
				c,
				orgID,
				namespace,
				"pipeline-1-step",
				"step-run",
			)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestOwnedByCachesLookups(t *testing.T) {
	owned.Clear()
	f := newFixture(t)
	namespace := canonify.RunnerNamespacePrefix + f.runnerID
	at := time.Now()
	original := now
	t.Cleanup(func() { now = original })
	now = func() time.Time { return at }

	c := temporalmocks.NewClient(t)
	c.On("DescribeWorkflow", mock.Anything, "pipeline-1-step", "step-run").
		Return(temporalclient.Description(description(
			t,
			mobiledevicesession.WorkflowName,
			&mobiledevicesession.PipelineRunRef{WorkflowID: "pipeline-1", RunID: "run-1"},
		)), nil).
		Twice()

	for range 2 {
		got, err := OwnedBy(
			context.Background(),
			f.app,
			c,
			f.orgID,
			namespace,
			"pipeline-1-step",
			"step-run",
		)
		require.NoError(t, err)
		require.True(t, got)
	}

	at = at.Add(ownedTTL)
	got, err := OwnedBy(
		context.Background(),
		f.app,
		c,
		f.orgID,
		namespace,
		"pipeline-1-step",
		"step-run",
	)
	require.NoError(t, err)
	require.True(t, got)
}
