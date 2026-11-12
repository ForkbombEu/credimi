// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
)

const (
	runTestWorkflowID = "Pipeline-wallet-check-ticket-1"
	runTestRunID      = "pipeline-run-1"
	runTestPipeline   = "acme/wallet-check"
)

// pipelineRunFixture is a session of the fixture's wallet and the caller's
// namespace client, where the test adds the pipeline run's executions.
type pipelineRunFixture struct {
	app       core.App
	session   actionauthor.BuildRequest
	caller    authorCaller
	client    *temporalmocks.Client
	selection actionauthor.Selection
	// namespaces lists the namespace of every client the activity asked for.
	namespaces *[]string
}

func newPipelineRunFixture(t *testing.T) pipelineRunFixture {
	t.Helper()
	setFCAFTestEnv(t)
	app, session, namespace := authorSaveFixture(t)
	caller := authorTestCaller()
	selection, err := actionauthor.Select(app, namespace, session)
	require.NoError(t, err)
	return pipelineRunFixture{
		app:        app,
		session:    session,
		caller:     caller,
		client:     replayedSessionClient(buildSessionLookup(t, app, session)),
		selection:  selection,
		namespaces: &[]string{},
	}
}

// replayedSessionClient fakes the session of lookup for every read: each
// lookup reads the session's start input from a fresh history.
func replayedSessionClient(lookup sessionLookup) *temporalmocks.Client {
	events := lookup.history.(*fcafHistoryIterator).events
	mockClient := &temporalmocks.Client{}
	stubSessionRuns(mockClient, authorTestSessionID, lookup.runs...)
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestAuthorID, authorTestAuthorRun).
		Return(lookup.author, nil).
		Maybe()
	mockClient.On("DescribeWorkflowExecution", mock.Anything, authorTestSessionID, authorTestParentRun).
		Return(lookup.session, nil).
		Maybe()
	mockClient.On(
		"GetWorkflowHistory", mock.Anything, authorTestSessionID, authorTestParentRun, false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(func(
		context.Context, string, string, bool, enumspb.HistoryEventFilterType,
	) client.HistoryEventIterator {
		return &fcafHistoryIterator{events: events}
	}).Maybe()
	return mockClient
}

func (f pipelineRunFixture) read(
	t *testing.T,
	caller authorCaller,
	text string,
) (ReadActionAuthorPipelineRunOutput, error) {
	t.Helper()
	a := NewReadActionAuthorPipelineRunActivity(f.app)
	a.authorSessionLookup = authorSessionLookup{
		app: f.app,
		temporalClient: func(namespace string) (client.Client, error) {
			*f.namespaces = append(*f.namespaces, namespace)
			return f.client, nil
		},
	}
	return a.read(t.Context(), caller, ReadActionAuthorPipelineRunInput{Text: text})
}

// seedRunAction adds an action of the session's wallet with category.
func (f pipelineRunFixture) seedRunAction(t *testing.T, name, category string) string {
	t.Helper()
	path := seedAuthorAction(t, f.app, f.session, name, "appId: eu.wallet\n---\n- launchApp\n")
	record, err := actionauthor.CheckRecordAccess(
		f.app,
		"wallet_actions",
		path,
		f.selection.Organization.Id,
	)
	require.NoError(t, err)
	record.Set("category", category)
	require.NoError(t, f.app.Save(record))
	return path
}

// describeRun answers the description of the run with type and status.
func (f pipelineRunFixture) describeRun(
	workflowID, runID, workflowType string,
	status enumspb.WorkflowExecutionStatus,
) {
	identifier, _ := converter.GetDefaultDataConverter().ToPayload(runTestPipeline)
	f.client.On("DescribeWorkflowExecution", mock.Anything, workflowID, runID).
		Return(&workflowservice.DescribeWorkflowExecutionResponse{
			WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{
				Execution: &commonpb.WorkflowExecution{WorkflowId: workflowID, RunId: runTestRunID},
				Type:      &commonpb.WorkflowType{Name: workflowType},
				Status:    status,
				SearchAttributes: &commonpb.SearchAttributes{
					IndexedFields: map[string]*commonpb.Payload{
						workflowengine.PipelineIdentifierSearchAttribute: identifier,
					},
				},
			},
		}, nil).
		Once()
}

// failedRun answers the history of the failed run with its definition and
// close failure.
func (f pipelineRunFixture) failedRun(
	t *testing.T,
	definition map[string]any,
	failure error,
) {
	t.Helper()
	input, err := temporalcrypto.DataConverter().ToPayloads(map[string]any{
		"workflow_definition": definition,
	})
	require.NoError(t, err)
	f.client.On(
		"GetWorkflowHistory", mock.Anything, runTestWorkflowID, runTestRunID, false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	).Return(&fcafHistoryIterator{events: []*historypb.HistoryEvent{{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_STARTED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionStartedEventAttributes{
			WorkflowExecutionStartedEventAttributes: &historypb.WorkflowExecutionStartedEventAttributes{
				Input: input,
			},
		},
	}}}).Once()
	f.client.On(
		"GetWorkflowHistory", mock.Anything, runTestWorkflowID, runTestRunID, false,
		enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT,
	).Return(&fcafHistoryIterator{events: []*historypb.HistoryEvent{{
		EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED,
		Attributes: &historypb.HistoryEvent_WorkflowExecutionFailedEventAttributes{
			WorkflowExecutionFailedEventAttributes: &historypb.WorkflowExecutionFailedEventAttributes{
				Failure: temporal.GetDefaultFailureConverter().ErrorToFailure(failure),
			},
		},
	}}}).Once()
}

// pipelineFailure is the pipeline's CRE229 close failure listing stepErrors.
func pipelineFailure(stepErrors ...workflowengine.WorkflowError) error {
	return workflowengine.NewAppError(workflowengine.WorkflowError{
		Code:    errorcodes.Codes[errorcodes.PipelineExecutionError].Code,
		Summary: "Pipeline failed",
		Details: map[string]any{"errors": stepErrors},
	})
}

// mobileRunStep is a mobile-automation step running action on the session's build.
func (f pipelineRunFixture) mobileRunStep(
	id, action string,
	parameters map[string]any,
) map[string]any {
	return map[string]any{
		"id":  id,
		"use": "mobile-automation",
		"with": map[string]any{
			"action_id":  action,
			"version_id": f.session.WalletVersionIdentifier,
			"parameters": parameters,
		},
	}
}

func TestReadActionAuthorPipelineRunReadsTheFailedSteps(t *testing.T) {
	f := newPipelineRunFixture(t)
	onboard := f.seedRunAction(t, "Onboard", "onboarding")
	get := f.seedRunAction(t, "Get PID", "get-credential-generic")
	verify := f.seedRunAction(t, "Verify PID", "verify-credential")
	f.describeRun(runTestWorkflowID, runTestRunID, pipelineWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_FAILED)
	f.failedRun(t, map[string]any{
		"name": "Wallet check",
		"steps": []any{
			f.mobileRunStep("onboard", onboard, nil),
			map[string]any{
				"id":   "offer",
				"use":  "credential-offer",
				"with": map[string]any{"credential_id": "tenant/issuer/pid"},
			},
			map[string]any{
				"id":  "create-session",
				"use": "http-request",
				"with": map[string]any{
					"method": "POST",
					"url":    "https://verifier.example/sessions",
				},
			},
			f.mobileRunStep("get", get, map[string]any{"OFFER": "${{ offer.outputs }}"}),
			f.mobileRunStep("verify", verify, map[string]any{
				"REQUEST": "${{ create-session.outputs.body.deeplink }}",
			}),
		},
	}, pipelineFailure(
		workflowengine.WorkflowError{
			Code:    "CRE301",
			Summary: "Command execution failed",
			Message: "Maestro flow failed.",
			Details: map[string]any{
				"step_id": "verify",
				"stdout":  "Launch app... COMPLETED\nAssert that \"MY WALLET\" is visible... FAILED\n",
			},
		},
		workflowengine.WorkflowError{
			Code:    "CRE304",
			Summary: "Unexpected HTTP response body",
			Message: "verifier answered 500",
			Details: map[string]any{"step_id": "create-session"},
		},
		workflowengine.WorkflowError{
			Code:    "CRE301",
			Summary: "Command execution failed",
			Message: "Maestro flow failed.",
			Details: map[string]any{"step_id": "get", "stdout": "Tap on \"Add\"... FAILED"},
		},
	))

	output, err := f.read(
		t,
		f.caller,
		"https://credimi.io/my/tests/runs/"+runTestWorkflowID+"/"+runTestRunID,
	)
	require.NoError(t, err)
	require.Equal(t, runTestPipeline, output.Pipeline)
	require.Equal(t, runTestWorkflowID, output.WorkflowID)
	require.Equal(t, runTestRunID, output.RunID)
	require.Equal(t, "FAILED", output.Status)
	require.Equal(t, []string{
		"create-session (http-request): verifier answered 500",
	}, output.OtherFailures)
	require.Len(t, output.Failures, 2)

	first := output.Failures[0]
	require.Equal(t, "get", first.StepID)
	require.Equal(t, get, first.Action)
	require.Equal(t, "Get PID", first.ActionName)
	require.True(t, first.Owned)
	require.True(t, first.InSessionWallet)
	require.Equal(t, "CRE301", first.Code)
	require.Equal(t, "Maestro flow failed.", first.Message)
	require.Equal(t, `Tap on "Add"`, first.FailedCommand)
	require.Empty(t, first.Notes)
	require.Equal(t, &actionauthor.Spec{
		Slot:     "get-credential",
		Outcome:  actionauthor.OutcomeSuccess,
		Replaces: &actionauthor.ReplaceSpec{Action: get},
		Prerequisites: []actionauthor.PrerequisiteSpec{{
			Action:     onboard,
			Parameters: []actionauthor.ParameterSpec{},
		}},
		Parameters: []actionauthor.ParameterSpec{{
			Name: "OFFER",
			Source: actionauthor.SourceSpec{
				Kind:       actionauthor.SourceCredentialOffer,
				Identifier: "tenant/issuer/pid",
			},
		}},
	}, first.Plan)

	second := output.Failures[1]
	require.Equal(t, "verify", second.StepID)
	require.Equal(t, `Assert that "MY WALLET" is visible`, second.FailedCommand)
	require.Equal(t,
		"Launch app... COMPLETED\nAssert that \"MY WALLET\" is visible... FAILED",
		second.OutputTail,
	)
	require.Equal(t, "verify-credential", second.Plan.Slot)
	require.Equal(t, []actionauthor.PrerequisiteSpec{
		{Action: onboard, Parameters: []actionauthor.ParameterSpec{}},
		{Action: get, Parameters: []actionauthor.ParameterSpec{{
			Name: "OFFER",
			Source: actionauthor.SourceSpec{
				Kind:       actionauthor.SourceCredentialOffer,
				Identifier: "tenant/issuer/pid",
			},
		}}},
	}, second.Plan.Prerequisites)
	require.Equal(t, []actionauthor.ParameterSpec{{
		Name: "REQUEST",
		Source: actionauthor.SourceSpec{
			Kind: actionauthor.SourceHTTP,
			Request: &actionauthor.HTTPSourceRequest{
				Method: "POST",
				URL:    "https://verifier.example/sessions",
			},
			ValuePath: "deeplink",
		},
	}}, second.Plan.Parameters)
	f.client.AssertExpectations(t)
}

func TestReadActionAuthorPipelineRunWithoutStepErrors(t *testing.T) {
	f := newPipelineRunFixture(t)
	f.describeRun(runTestWorkflowID, "", pipelineWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_FAILED)
	f.failedRun(t, map[string]any{"name": "Wallet check"}, workflowengine.NewAppError(
		workflowengine.WorkflowError{Code: "CRE227", Summary: "Failed to parse pipeline yaml"},
	))

	output, err := f.read(t, f.caller, runTestWorkflowID)
	require.NoError(t, err)
	require.Empty(t, output.Failures)
	require.Equal(t, []string{"Failed to parse pipeline yaml"}, output.OtherFailures)
	// The session is looked up in the runner namespace it runs in, the run
	// in the namespace of the session's organization.
	require.Equal(t, []string{
		authorTestNamespace,
		f.selection.Organization.GetString("canonified_name"),
	}, *f.namespaces)
}

func TestReadActionAuthorPipelineRunOfACompletedRun(t *testing.T) {
	f := newPipelineRunFixture(t)
	f.describeRun(runTestWorkflowID, "", pipelineWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)

	output, err := f.read(t, f.caller, runTestWorkflowID)
	require.NoError(t, err)
	require.Equal(t, ReadActionAuthorPipelineRunOutput{
		Pipeline:      runTestPipeline,
		WorkflowID:    runTestWorkflowID,
		RunID:         runTestRunID,
		Status:        "COMPLETED",
		Failures:      []actionauthor.PipelineRunFailure{},
		OtherFailures: []string{},
	}, output)
}

// seedRunPipeline adds a pipeline of owner and returns its path.
func seedRunPipeline(
	t *testing.T,
	app core.App,
	owner *core.Record,
	name string,
	published bool,
) string {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("pipelines")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", owner.Id)
	record.Set("name", name)
	record.Set("description", name)
	record.Set("yaml", "name: "+name+"\nsteps: []\n")
	record.Set("published", published)
	require.NoError(t, app.Save(record))
	return owner.GetString("canonified_name") + "/" + record.GetString("canonified_name")
}

func TestReadActionAuthorPipelineRunOfAPipeline(t *testing.T) {
	f := newPipelineRunFixture(t)
	path := seedRunPipeline(t, f.app, f.selection.Organization, "Wallet check", false)
	f.client.On("ListWorkflow", mock.Anything, &workflowservice.ListWorkflowExecutionsRequest{
		PageSize: 1,
		Query: "WorkflowType = 'Dynamic Pipeline Workflow' AND PipelineIdentifier = '" +
			path + "' AND ExecutionStatus = 'Failed'",
	}).Return(&workflowservice.ListWorkflowExecutionsResponse{
		Executions: []*workflowpb.WorkflowExecutionInfo{
			{
				Execution: &commonpb.WorkflowExecution{
					WorkflowId: runTestWorkflowID,
					RunId:      runTestRunID,
				},
			},
		},
	}, nil).Once()
	f.describeRun(runTestWorkflowID, runTestRunID, pipelineWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED)

	output, err := f.read(t, f.caller, "https://credimi.io/my/pipelines/"+path)
	require.NoError(t, err)
	require.Equal(t, runTestWorkflowID, output.WorkflowID)
	require.Equal(t, runTestRunID, output.RunID)

	f.client.On("ListWorkflow", mock.Anything, mock.Anything).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil).Once()
	_, err = f.read(t, f.caller, path)
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
	require.ErrorContains(t, err, "no failed run of pipeline "+path+" in your organization")
	f.client.AssertExpectations(t)
}

func TestReadActionAuthorPipelineRunRefusals(t *testing.T) {
	f := newPipelineRunFixture(t)
	other, err := f.app.FindFirstRecordByData("organizations", "name", "userB's organization")
	require.NoError(t, err)
	foreign := seedRunPipeline(t, f.app, other, "Private check", false)
	f.describeRun("Pipeline-running", "", pipelineWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING)
	f.describeRun("Pipeline-session", "", ActionAuthorSessionWorkflowName,
		enumspb.WORKFLOW_EXECUTION_STATUS_FAILED)
	f.client.On("DescribeWorkflowExecution", mock.Anything, "Pipeline-elsewhere", "").
		Return(nil, serviceerror.NewNotFound("not found")).Once()

	notAgent := f.caller
	notAgent.WorkflowType = "Dynamic Pipeline Workflow"
	tests := []struct {
		name    string
		caller  authorCaller
		text    string
		code    string
		message string
	}{
		{
			name:    "caller is not the agent",
			caller:  notAgent,
			text:    runTestWorkflowID,
			code:    errorcodes.RecordNotAccessible,
			message: "caller is not an action author session agent",
		},
		{
			name:    "not a reference",
			caller:  f.caller,
			text:    "fix it",
			code:    errorcodes.MissingOrInvalidPayload,
			message: "not a pipeline run or pipeline: fix it",
		},
		{
			name:    "pipeline of another organization",
			caller:  f.caller,
			text:    foreign,
			code:    errorcodes.MissingOrInvalidPayload,
			message: "pipeline " + foreign + " not found",
		},
		{
			name:    "run of another namespace",
			caller:  f.caller,
			text:    "Pipeline-elsewhere",
			code:    errorcodes.MissingOrInvalidPayload,
			message: "run Pipeline-elsewhere is not a pipeline run",
		},
		{
			name:    "another workflow",
			caller:  f.caller,
			text:    "Pipeline-session",
			code:    errorcodes.MissingOrInvalidPayload,
			message: "run Pipeline-session is not a pipeline run",
		},
		{
			name:    "running run",
			caller:  f.caller,
			text:    "Pipeline-running",
			code:    errorcodes.MissingOrInvalidPayload,
			message: "run Pipeline-running did not fail (status RUNNING)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := f.read(t, tt.caller, tt.text)
			requireActivityError(t, err, tt.code, true)
			require.ErrorContains(t, err, tt.message)
		})
	}
}
