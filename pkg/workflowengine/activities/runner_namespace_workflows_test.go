// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/mobiledevicesession"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

func testPipelineRun() mobiledevicesession.PipelineRunRef {
	return mobiledevicesession.PipelineRunRef{
		Namespace:  "acme",
		WorkflowID: "Pipeline-acme-t1",
		RunID:      "pipeline-run",
	}
}

func clientsFor(byNamespace map[string]client.Client) runnerWorkflowClients {
	return func(namespace string) (client.Client, error) {
		c, ok := byNamespace[namespace]
		if !ok {
			return nil, errors.New("unexpected namespace " + namespace)
		}
		return c, nil
	}
}

func runPayload() mobiledevicesession.RunRunnerWorkflowPayload {
	return mobiledevicesession.RunRunnerWorkflowPayload{
		Namespace:    testRunnerNamespace,
		WorkflowID:   "Pipeline-acme-t1-step",
		WorkflowName: mobiledevicesession.MobileAutomationWorkflowName,
		Memo:         map[string]any{"test": "flow"},
		Summary:      "step",
		Pipeline:     testPipelineRun(),
	}
}

func TestRunRunnerWorkflowStartsWithPipelineMemoAndReturnsResult(t *testing.T) {
	c := &temporalmocks.Client{}
	run := &temporalmocks.WorkflowRun{}
	run.On("GetRunID").Return("step-run")
	run.On("Get", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		out := args.Get(1).(*workflowengine.WorkflowResult)
		out.Output = map[string]any{"ok": true}
	}).Return(nil)
	c.On(
		"ExecuteWorkflow",
		mock.Anything,
		mock.MatchedBy(func(opts client.StartWorkflowOptions) bool {
			return opts.ID == "Pipeline-acme-t1-step" &&
				opts.TaskQueue == pipelineTaskQueue &&
				opts.Memo["test"] == "flow" &&
				opts.Memo[mobiledevicesession.PipelineRunMemoKey] == testPipelineRun() &&
				opts.WorkflowIDReusePolicy == enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE &&
				opts.WorkflowIDConflictPolicy == enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING
		}),
		mobiledevicesession.MobileAutomationWorkflowName,
		mock.Anything,
	).Return(run, nil)

	act := NewRunRunnerWorkflowActivity()
	act.clients = clientsFor(map[string]client.Client{testRunnerNamespace: c})
	res, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: runPayload()},
	)
	require.NoError(t, err)
	out := res.Output.(mobiledevicesession.RunRunnerWorkflowResult)
	require.Equal(t, "step-run", out.RunID)
	require.Equal(t, testRunnerNamespace, out.Namespace)
	require.Equal(t, map[string]any{"ok": true}, out.Result.Output)
}

func TestRunRunnerWorkflowReusesClosedRun(t *testing.T) {
	c := &temporalmocks.Client{}
	run := &temporalmocks.WorkflowRun{}
	run.On("GetRunID").Return("closed-run")
	run.On("Get", mock.Anything, mock.Anything).Return(nil)
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, serviceerror.NewWorkflowExecutionAlreadyStarted("started", "", ""))
	c.On("GetWorkflow", mock.Anything, "Pipeline-acme-t1-step", "").Return(run)

	act := NewRunRunnerWorkflowActivity()
	act.clients = clientsFor(map[string]client.Client{testRunnerNamespace: c})
	res, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: runPayload()},
	)
	require.NoError(t, err)
	require.Equal(
		t,
		"closed-run",
		res.Output.(mobiledevicesession.RunRunnerWorkflowResult).RunID,
	)
}

func TestRunRunnerWorkflowReturnsTheWorkflowFailure(t *testing.T) {
	failure := workflowengine.WorkflowError{
		Code:    "CRE999",
		Summary: "flow failed",
		Details: map[string]any{"output": map[string]any{"stdout": "boom"}},
	}
	c := &temporalmocks.Client{}
	run := &temporalmocks.WorkflowRun{}
	run.On("GetRunID").Return("step-run")
	run.On("Get", mock.Anything, mock.Anything).
		Return(workflowengine.NewAppError(failure))
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(run, nil)

	act := NewRunRunnerWorkflowActivity()
	act.clients = clientsFor(map[string]client.Client{testRunnerNamespace: c})
	_, err := act.Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: runPayload()},
	)
	require.Error(t, err)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, "CRE999", workflowengine.ParseWorkflowError(err).Code)
	require.Equal(
		t,
		map[string]any{"stdout": "boom"},
		workflowengine.ExtractOutputFromError(err),
	)
}

func TestRunRunnerWorkflowCancelsTheWorkflowWhenCanceled(t *testing.T) {
	c := &temporalmocks.Client{}
	run := &temporalmocks.WorkflowRun{}
	run.On("GetRunID").Return("step-run")
	closed := make(chan struct{})
	run.On("Get", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
		<-closed
	}).Return(temporal.NewCanceledError())
	c.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(run, nil)
	c.On("CancelWorkflow", mock.Anything, "Pipeline-acme-t1-step", "step-run").
		Run(func(mock.Arguments) { close(closed) }).
		Return(nil)

	act := NewRunRunnerWorkflowActivity()
	act.clients = clientsFor(map[string]client.Client{testRunnerNamespace: c})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := act.Execute(ctx, workflowengine.ActivityInput{Payload: runPayload()})
	require.True(t, temporal.IsCanceledError(err))
	c.AssertCalled(t, "CancelWorkflow", mock.Anything, "Pipeline-acme-t1-step", "step-run")
}

func TestRunnerNamespaceActivitiesRefuseOtherNamespaces(t *testing.T) {
	payload := runPayload()
	payload.Namespace = "acme"
	_, err := NewRunRunnerWorkflowActivity().Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: payload},
	)
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.True(t, appErr.NonRetryable())
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code, appErr.Type())

	_, err = NewAwaitDeviceSessionActivity().Execute(
		context.Background(),
		workflowengine.ActivityInput{Payload: mobiledevicesession.RunnerWorkflowRef{
			Namespace:  "runner-",
			WorkflowID: "x",
		}},
	)
	require.ErrorAs(t, err, &appErr)
	require.True(t, appErr.NonRetryable())
}

func TestReleaseDeviceSessionOfEndedSessionWarns(t *testing.T) {
	c := &temporalmocks.Client{}
	c.On("UpdateWorkflow", mock.Anything, mock.Anything).
		Return(nil, serviceerror.NewNotFound("workflow execution already completed"))
	act := NewReleaseDeviceSessionActivity()
	act.clients = clientsFor(map[string]client.Client{testRunnerNamespace: c})
	res, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReleaseDeviceSessionPayload{Ref: mobiledevicesession.RunnerWorkflowRef{
			Namespace:  testRunnerNamespace,
			WorkflowID: "Pipeline-acme-t1-device-pixel",
			RunID:      "session-run",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{
		"device session Pipeline-acme-t1-device-pixel had already ended",
	}, res.Output.(mobiledevicesession.Cleanup).Warnings)
}

func TestReleaseDeviceSessionSendsSkipCleanup(t *testing.T) {
	c := &temporalmocks.Client{}
	handle := &temporalmocks.WorkflowUpdateHandle{}
	handle.On("Get", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		out := args.Get(1).(*mobiledevicesession.Cleanup)
		out.ResultVideoURLs = []string{"https://video"}
	}).Return(nil)
	c.On("UpdateWorkflow", mock.Anything, mock.MatchedBy(func(o client.UpdateWorkflowOptions) bool {
		return o.UpdateName == mobiledevicesession.ReleaseUpdate &&
			o.UpdateID == "release" &&
			len(o.Args) == 1 &&
			o.Args[0] == mobiledevicesession.Release{SkipCleanup: true}
	})).Return(handle, nil)
	act := NewReleaseDeviceSessionActivity()
	act.clients = clientsFor(map[string]client.Client{testRunnerNamespace: c})
	res, err := act.Execute(context.Background(), workflowengine.ActivityInput{
		Payload: ReleaseDeviceSessionPayload{
			Ref: mobiledevicesession.RunnerWorkflowRef{
				Namespace:  testRunnerNamespace,
				WorkflowID: "session",
			},
			SkipCleanup: true,
		},
	})
	require.NoError(t, err)
	require.Equal(
		t,
		[]string{"https://video"},
		res.Output.(mobiledevicesession.Cleanup).ResultVideoURLs,
	)
}

func TestCheckPipelineRun(t *testing.T) {
	describe := func(status enumspb.WorkflowExecutionStatus) *client.WorkflowExecutionDescription {
		return &client.WorkflowExecutionDescription{
			RawResponse: &workflowservice.DescribeWorkflowExecutionResponse{
				WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{Status: status},
			},
		}
	}
	payload := CheckPipelineRunPayload{
		Pipeline:        testPipelineRun(),
		RunnerNamespace: testRunnerNamespace,
	}

	t.Run("running pipeline keeps the device", func(t *testing.T) {
		org := &temporalmocks.Client{}
		org.On("DescribeWorkflow", mock.Anything, "Pipeline-acme-t1", "pipeline-run").
			Return(describe(enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING), nil)
		act := NewCheckPipelineRunActivity()
		act.clients = clientsFor(map[string]client.Client{"acme": org})
		res, err := act.Execute(
			context.Background(),
			workflowengine.ActivityInput{Payload: payload},
		)
		require.NoError(t, err)
		require.True(t, res.Output.(CheckPipelineRunResult).Running)
	})

	for name, describeErr := range map[string]error{
		"terminated pipeline": nil,
		"missing pipeline":    serviceerror.NewNotFound("gone"),
	} {
		t.Run(name+" cancels orphaned steps", func(t *testing.T) {
			org := &temporalmocks.Client{}
			if describeErr != nil {
				org.On("DescribeWorkflow", mock.Anything, mock.Anything, mock.Anything).
					Return(nil, describeErr)
			} else {
				org.On("DescribeWorkflow", mock.Anything, mock.Anything, mock.Anything).
					Return(describe(enumspb.WORKFLOW_EXECUTION_STATUS_TERMINATED), nil)
			}
			runner := &temporalmocks.Client{}
			runner.On("ListWorkflow", mock.Anything, mock.MatchedBy(
				func(req *workflowservice.ListWorkflowExecutionsRequest) bool {
					return req.GetNamespace() == testRunnerNamespace &&
						req.GetQuery() == `WorkflowType = "Run a mobile automation workflow"`+
							` AND ExecutionStatus = 'Running'`+
							` AND WorkflowId STARTS_WITH "Pipeline-acme-t1-"`
				},
			)).Return(&workflowservice.ListWorkflowExecutionsResponse{
				Executions: []*workflowpb.WorkflowExecutionInfo{{
					Execution: &commonpb.WorkflowExecution{
						WorkflowId: "Pipeline-acme-t1-step",
						RunId:      "r1",
					},
				}},
			}, nil)
			runner.On("CancelWorkflow", mock.Anything, "Pipeline-acme-t1-step", "r1").Return(nil)
			act := NewCheckPipelineRunActivity()
			act.clients = clientsFor(map[string]client.Client{
				"acme":              org,
				testRunnerNamespace: runner,
			})
			res, err := act.Execute(
				context.Background(),
				workflowengine.ActivityInput{Payload: payload},
			)
			require.NoError(t, err)
			require.False(t, res.Output.(CheckPipelineRunResult).Running)
			runner.AssertExpectations(t)
		})
	}
}
