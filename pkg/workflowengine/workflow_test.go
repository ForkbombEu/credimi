// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflowengine

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	temporalmocks "go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

type staticEncodedValue struct {
	value any
	err   error
}

func (s staticEncodedValue) HasValue() bool { return true }

func (s staticEncodedValue) Get(valuePtr interface{}) error {
	if s.err != nil {
		return s.err
	}
	data, err := json.Marshal(s.value)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, valuePtr)
}

type testWorkflow struct {
	name     string
	options  workflow.ActivityOptions
	execute  func(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error)
	metadata *WorkflowErrorMetadata
}

func (t *testWorkflow) Workflow(ctx workflow.Context, input WorkflowInput) (WorkflowResult, error) {
	return t.ExecuteWorkflow(ctx, input)
}

func (t *testWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input WorkflowInput,
) (WorkflowResult, error) {
	t.metadata = input.RunMetadata
	return t.execute(ctx, input)
}

func (t *testWorkflow) Name() string {
	return t.name
}

func (t *testWorkflow) GetOptions() workflow.ActivityOptions {
	return t.options
}

func TestNewWorkflowError_NonApplicationError(t *testing.T) {
	baseErr := errors.New("plain error")
	metadata := &WorkflowErrorMetadata{WorkflowName: "wf", TemporalUI: "https://temporal.test"}

	got := NewWorkflowError(baseErr, metadata)

	require.Equal(t, baseErr, got)
}

func TestNewWorkflowError_WrapsTemporalApplicationError(t *testing.T) {
	metadata := &WorkflowErrorMetadata{
		WorkflowName: "wf",
		WorkflowID:   "wf-1",
		Namespace:    "default",
		TemporalUI:   "https://temporal.test/runs/wf-1",
	}
	original := temporal.NewApplicationError(
		"boom",
		"CRE999",
		map[string]any{"output": map[string]any{"key": "value"}},
	)

	got := NewWorkflowError(original, metadata, "extra-payload")

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, got, &appErr)
	require.Equal(t, "CRE999", appErr.Type())
	require.Contains(t, appErr.Message(), "CRE999: workflow engine wf: boom")
	require.Contains(t, appErr.Message(), "Further information at: https://temporal.test/runs/wf-1")

	var details []any
	require.NoError(t, appErr.Details(&details))
	require.Len(t, details, 3)
}

func TestWorkflowSpecificErrorHelpers(t *testing.T) {
	metadata := &WorkflowErrorMetadata{
		WorkflowName: "wf",
		TemporalUI:   "https://temporal.test/runs/wf-1",
	}

	cancelErr := NewWorkflowCancellationError(metadata)
	require.ErrorContains(t, cancelErr, "canceled")

	appErr := NewAppError(
		errorcodes.Codes[errorcodes.MissingOrInvalidPayload],
		"field-a",
		"payload",
	)
	var temporalErr *temporal.ApplicationError
	require.ErrorAs(t, appErr, &temporalErr)
	require.Equal(t, errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code, temporalErr.Type())

	missingPayloadErr := NewMissingOrInvalidPayloadError(errors.New("bad payload"), metadata)
	require.Equal(
		t,
		errorcodes.Codes[errorcodes.MissingOrInvalidPayload].Code,
		ParseWorkflowError(missingPayloadErr).Code,
	)

	missingConfigErr := NewMissingConfigError("app_url", metadata)
	require.Equal(
		t,
		errorcodes.Codes[errorcodes.MissingOrInvalidConfig].Code,
		ParseWorkflowError(missingConfigErr).Code,
	)

	stepErr := NewStepCIOutputError("logs", map[string]any{"invalid": true}, metadata)
	require.Equal(
		t,
		errorcodes.Codes[errorcodes.UnexpectedStepCIOutput].Code,
		ParseWorkflowError(stepErr).Code,
	)
}

func TestWaitForWorkflowResult(t *testing.T) {
	t.Run("returns result from workflow run", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockRun := &temporalmocks.WorkflowRun{}
		mockRun.
			On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
			Run(func(args mock.Arguments) {
				out := args.Get(1).(*WorkflowResult)
				*out = WorkflowResult{WorkflowID: "wf-id", WorkflowRunID: "run-id", Message: "ok"}
			}).
			Return(nil).
			Once()
		mockClient.On("GetWorkflow", mock.Anything, "wf-id", "run-id").Return(mockRun).Once()

		got, err := WaitForWorkflowResult(mockClient, "wf-id", "run-id")
		require.NoError(t, err)
		require.Equal(t, "ok", got.Message)

		mockClient.AssertExpectations(t)
		mockRun.AssertExpectations(t)
	})

	t.Run("bubbles get error", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockRun := &temporalmocks.WorkflowRun{}
		mockRun.
			On("Get", mock.Anything, mock.AnythingOfType("*workflowengine.WorkflowResult")).
			Return(errors.New("get failed")).
			Once()
		mockClient.On("GetWorkflow", mock.Anything, "wf-id", "run-id").Return(mockRun).Once()

		_, err := WaitForWorkflowResult(mockClient, "wf-id", "run-id")
		require.ErrorContains(t, err, "get failed")

		mockClient.AssertExpectations(t)
		mockRun.AssertExpectations(t)
	})
}

func TestWaitForPartialResult(t *testing.T) {
	t.Run("returns decoded value when query succeeds", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		encoded := staticEncodedValue{
			value: map[string]any{
				"status": "running",
				"count":  2,
			},
		}
		mockClient.
			On("QueryWorkflow", mock.Anything, "wf-id", "run-id", "status").
			Return(converter.EncodedValue(encoded), nil).
			Once()

		got, err := WaitForPartialResult[map[string]any](
			mockClient,
			"wf-id",
			"run-id",
			"status",
			time.Millisecond,
			50*time.Millisecond,
		)
		require.NoError(t, err)
		require.Equal(t, "running", got["status"])
		require.Equal(t, float64(2), got["count"])

		mockClient.AssertExpectations(t)
	})

	t.Run("continues on not-ready and times out", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On("QueryWorkflow", mock.Anything, "wf-id", "run-id", "status").
			Return(converter.EncodedValue(nil), errors.New("result not ready"))

		_, err := WaitForPartialResult[map[string]any](
			mockClient,
			"wf-id",
			"run-id",
			"status",
			time.Millisecond,
			8*time.Millisecond,
		)
		require.Error(t, err)
		require.ErrorContains(t, err, "timeout waiting for partial result")
	})

	t.Run("returns query error when not not-ready", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On("QueryWorkflow", mock.Anything, "wf-id", "run-id", "status").
			Return(converter.EncodedValue(nil), errors.New("query failed")).
			Once()

		_, err := WaitForPartialResult[map[string]any](
			mockClient,
			"wf-id",
			"run-id",
			"status",
			time.Millisecond,
			50*time.Millisecond,
		)
		require.ErrorContains(t, err, "query failed")
		mockClient.AssertExpectations(t)
	})

	t.Run("returns decode error", func(t *testing.T) {
		mockClient := &temporalmocks.Client{}
		mockClient.
			On("QueryWorkflow", mock.Anything, "wf-id", "run-id", "status").
			Return(converter.EncodedValue(staticEncodedValue{err: errors.New("decode failed")}), nil).
			Once()

		_, err := WaitForPartialResult[map[string]any](
			mockClient,
			"wf-id",
			"run-id",
			"status",
			time.Millisecond,
			50*time.Millisecond,
		)
		require.ErrorContains(t, err, "decode failed")
		mockClient.AssertExpectations(t)
	})
}

func TestParseWorkflowError(t *testing.T) {
	raw := errors.New(
		"workflow failed (workflowID: wf-123, runID: run-456) " +
			"(type: CRE999, retryable: false) CRE999: [ctx]: human readable message " +
			"(Further information at: https://temporal.test/runs/wf-123)",
	)

	got := ParseWorkflowError(raw)

	require.Equal(t, "wf-123", got.WorkflowID)
	require.Equal(t, "run-456", got.RunID)
	require.Equal(t, "CRE999", got.Code)
	require.False(t, got.Retryable)
	require.Equal(t, "https://temporal.test/runs/wf-123", got.Link)
	require.Equal(t, "human readable message", got.Summary)
}

func TestBuildWorkflowMetadataAndErrors(t *testing.T) {
	t.Run("populates metadata on success", func(t *testing.T) {
		suite := testsuite.WorkflowTestSuite{}
		env := suite.NewTestWorkflowEnvironment()

		wf := &testWorkflow{
			name:    "test-workflow",
			options: workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			execute: func(_ workflow.Context, _ WorkflowInput) (WorkflowResult, error) {
				return WorkflowResult{Message: "ok"}, nil
			},
		}

		wfFn := BuildWorkflow(wf)
		env.RegisterWorkflow(wfFn)
		env.ExecuteWorkflow(wfFn, WorkflowInput{
			Config: map[string]any{"app_url": "https://app.example.com"},
		})

		require.True(t, env.IsWorkflowCompleted())
		require.NoError(t, env.GetWorkflowError())
		require.NotNil(t, wf.metadata)
		require.Equal(t, "test-workflow", wf.metadata.WorkflowName)
		require.NotEmpty(t, wf.metadata.WorkflowID)
		require.NotEmpty(t, wf.metadata.Namespace)
		require.Contains(t, wf.metadata.TemporalUI, "https://app.example.com")
	})

	t.Run("returns timeout error without wrapping", func(t *testing.T) {
		suite := testsuite.WorkflowTestSuite{}
		env := suite.NewTestWorkflowEnvironment()

		timeoutErr := temporal.NewTimeoutError(enumspb.TIMEOUT_TYPE_START_TO_CLOSE, nil)
		wf := &testWorkflow{
			name:    "timeout-workflow",
			options: workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			execute: func(_ workflow.Context, _ WorkflowInput) (WorkflowResult, error) {
				return WorkflowResult{}, timeoutErr
			},
		}

		wfFn := BuildWorkflow(wf)
		env.RegisterWorkflow(wfFn)
		env.ExecuteWorkflow(wfFn, WorkflowInput{
			Config: map[string]any{"app_url": "https://app.example.com"},
		})

		err := env.GetWorkflowError()
		require.Error(t, err)
		require.True(t, temporal.IsTimeoutError(err))
	})

	t.Run("wraps canceled error as workflow cancellation", func(t *testing.T) {
		suite := testsuite.WorkflowTestSuite{}
		env := suite.NewTestWorkflowEnvironment()

		wf := &testWorkflow{
			name:    "canceled-workflow",
			options: workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			execute: func(_ workflow.Context, _ WorkflowInput) (WorkflowResult, error) {
				return WorkflowResult{}, temporal.NewCanceledError()
			},
		}

		wfFn := BuildWorkflow(wf)
		env.RegisterWorkflow(wfFn)
		env.ExecuteWorkflow(wfFn, WorkflowInput{
			Config: map[string]any{"app_url": "https://app.example.com"},
		})

		err := env.GetWorkflowError()
		require.Error(t, err)
		require.True(t, temporal.IsCanceledError(err))
	})

	t.Run("wraps application errors with metadata", func(t *testing.T) {
		suite := testsuite.WorkflowTestSuite{}
		env := suite.NewTestWorkflowEnvironment()

		wf := &testWorkflow{
			name:    "error-workflow",
			options: workflow.ActivityOptions{StartToCloseTimeout: time.Second},
			execute: func(_ workflow.Context, _ WorkflowInput) (WorkflowResult, error) {
				return WorkflowResult{}, temporal.NewApplicationError("boom", "CRE-1")
			},
		}

		wfFn := BuildWorkflow(wf)
		env.RegisterWorkflow(wfFn)
		env.ExecuteWorkflow(wfFn, WorkflowInput{
			Config: map[string]any{"app_url": "https://app.example.com"},
		})

		err := env.GetWorkflowError()
		require.Error(t, err)
		var appErr *temporal.ApplicationError
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, "CRE-1", appErr.Type())
		require.Contains(t, appErr.Message(), "workflow engine")
	})
}

func TestStartWorkflowWithOptions(t *testing.T) {
	t.Run("returns client creation error", func(t *testing.T) {
		origClient := workflowTemporalClient
		t.Cleanup(func() { workflowTemporalClient = origClient })

		workflowTemporalClient = func(_ string) (client.Client, error) {
			return nil, errors.New("no client")
		}

		_, err := StartWorkflowWithOptions(
			"ns",
			client.StartWorkflowOptions{},
			"wf",
			WorkflowInput{},
		)
		require.ErrorContains(t, err, "unable to create client")
	})

	t.Run("bubbles execute workflow error", func(t *testing.T) {
		origClient := workflowTemporalClient
		t.Cleanup(func() { workflowTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		mockClient.
			On("ExecuteWorkflow", mock.Anything, mock.Anything, "wf", mock.Anything).
			Return(nil, errors.New("start failed"))

		workflowTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		_, err := StartWorkflowWithOptions(
			"ns",
			client.StartWorkflowOptions{},
			"wf",
			WorkflowInput{},
		)
		require.ErrorContains(t, err, "failed to start workflow")
	})

	t.Run("sets memo defaults and returns IDs", func(t *testing.T) {
		origClient := workflowTemporalClient
		t.Cleanup(func() { workflowTemporalClient = origClient })

		mockClient := temporalmocks.NewClient(t)
		run := temporalmocks.NewWorkflowRun(t)
		run.On("GetID").Return("wf-1")
		run.On("GetRunID").Return("run-1")

		mockClient.
			On(
				"ExecuteWorkflow",
				mock.Anything,
				mock.MatchedBy(func(opts client.StartWorkflowOptions) bool {
					return opts.Memo != nil &&
						opts.Memo["foo"] == "bar" &&
						opts.Memo["test"] == "wf"
				}),
				"wf",
				mock.Anything,
			).
			Return(run, nil)

		workflowTemporalClient = func(_ string) (client.Client, error) {
			return mockClient, nil
		}

		result, err := StartWorkflowWithOptions(
			"ns",
			client.StartWorkflowOptions{},
			"wf",
			WorkflowInput{Config: map[string]any{"memo": map[string]any{"foo": "bar"}}},
		)
		require.NoError(t, err)
		require.Equal(t, "wf-1", result.WorkflowID)
		require.Equal(t, "run-1", result.WorkflowRunID)
	})
}

func TestExtractAppErrorPayloadAndOutput(t *testing.T) {
	metadata := &WorkflowErrorMetadata{
		WorkflowName: "wf",
		TemporalUI:   "https://temporal.test/runs/wf-1",
	}

	baseWithOutput := temporal.NewApplicationError(
		"boom",
		"CRE777",
		map[string]any{"output": map[string]any{"a": "b"}},
	)
	withOutput := NewWorkflowError(baseWithOutput, metadata)

	baseWithoutOutput := temporal.NewApplicationError(
		"boom",
		"CRE778",
		map[string]any{"other": true},
	)
	withoutOutput := NewWorkflowError(baseWithoutOutput, metadata)

	payload := extractAppErrorPayload(withOutput)
	require.NotNil(t, payload)
	require.NotEmpty(t, payload)

	output := ExtractOutputFromError(withOutput)
	require.Equal(t, map[string]any{"a": "b"}, output)

	require.Nil(t, ExtractOutputFromError(withoutOutput))
	require.Nil(t, ExtractOutputFromError(errors.New("not app error")))
}

func TestNotReadyError_Error(t *testing.T) {
	var err error = NotReadyError{}
	require.Equal(t, "result not ready", err.Error())
}

func TestParseWorkflowError_IncludesPayloadForApplicationErrors(t *testing.T) {
	metadata := &WorkflowErrorMetadata{
		WorkflowName: "wf",
		TemporalUI:   "https://temporal.test/runs/wf-1",
	}
	appErr := temporal.NewApplicationError(
		"boom",
		"CRE777",
		map[string]any{"output": map[string]any{"x": "y"}},
	)
	wrapped := NewWorkflowError(appErr, metadata)

	got := ParseWorkflowError(wrapped)
	require.NotNil(t, got.Payload)
}

func TestWaitForPartialResult_TimeoutWithoutQueryCalls(t *testing.T) {
	mockClient := &temporalmocks.Client{}

	_, err := WaitForPartialResult[map[string]any](
		mockClient,
		"wf-id",
		"run-id",
		"status",
		50*time.Millisecond,
		time.Nanosecond,
	)
	require.Error(t, err)
	require.ErrorContains(t, err, context.DeadlineExceeded.Error())
}
