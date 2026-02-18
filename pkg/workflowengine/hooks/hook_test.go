// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
)

const testDataDir = "../../../test_pb_data"

func TestFetchNamespacesIncludesDefault(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	org, err := app.FindFirstRecordByFilter("organizations", `name="userA's organization"`)
	require.NoError(t, err)

	namespaces, err := FetchNamespaces(app)
	require.NoError(t, err)
	require.Contains(t, namespaces, "default")
	require.Contains(t, namespaces, org.GetString("canonified_name"))
}

func TestStartAllWorkersByNamespaceDefault(t *testing.T) {
	workerCancels = sync.Map{}

	originalGetTemporalClient := getTemporalClient
	originalStartWorker := startWorkerFn
	originalStartPipelineWorker := startPipelineWorkerFn

	t.Cleanup(func() {
		getTemporalClient = originalGetTemporalClient
		startWorkerFn = originalStartWorker
		startPipelineWorkerFn = originalStartPipelineWorker
		workerCancels = sync.Map{}
	})

	getTemporalClient = func(namespace string) (client.Client, error) {
		return nil, nil
	}

	workerCh := make(chan string, len(DefaultWorkers))
	pipelineCh := make(chan struct{}, 1)

	startWorkerFn = func(_ context.Context, _ client.Client, config workerConfig, wg *sync.WaitGroup) {
		workerCh <- config.TaskQueue
		wg.Done()
	}
	startPipelineWorkerFn = func(_ context.Context, _ client.Client, wg *sync.WaitGroup) {
		pipelineCh <- struct{}{}
		wg.Done()
	}

	StartAllWorkersByNamespace("default")

	for i := 0; i < len(DefaultWorkers); i++ {
		<-workerCh
	}
	<-pipelineCh

	_, ok := workerCancels.Load("default")
	require.True(t, ok)

	StopAllWorkersByNamespace("default")
	_, ok = workerCancels.Load("default")
	require.False(t, ok)
}

func TestStartAllWorkersByNamespaceOrg(t *testing.T) {
	workerCancels = sync.Map{}

	originalGetTemporalClient := getTemporalClient
	originalStartWorker := startWorkerFn
	originalStartPipelineWorker := startPipelineWorkerFn

	t.Cleanup(func() {
		getTemporalClient = originalGetTemporalClient
		startWorkerFn = originalStartWorker
		startPipelineWorkerFn = originalStartPipelineWorker
		workerCancels = sync.Map{}
	})

	getTemporalClient = func(namespace string) (client.Client, error) {
		return nil, nil
	}

	workerCh := make(chan string, len(OrgWorkers))
	pipelineCh := make(chan struct{}, 1)

	startWorkerFn = func(_ context.Context, _ client.Client, config workerConfig, wg *sync.WaitGroup) {
		workerCh <- config.TaskQueue
		wg.Done()
	}
	startPipelineWorkerFn = func(_ context.Context, _ client.Client, wg *sync.WaitGroup) {
		pipelineCh <- struct{}{}
		wg.Done()
	}

	StartAllWorkersByNamespace("acme-org")

	for i := 0; i < len(OrgWorkers); i++ {
		<-workerCh
	}
	<-pipelineCh

	_, ok := workerCancels.Load("acme-org")
	require.True(t, ok)

	StopAllWorkersByNamespace("acme-org")
	_, ok = workerCancels.Load("acme-org")
	require.False(t, ok)
}

func TestEnsureNamespaceReadyWithRetrySuccess(t *testing.T) {
	originalNewNamespaceClient := newNamespaceClientFn
	originalSleep := sleepFn
	originalNow := nowFn

	mockClient := mocks.NewNamespaceClient(t)
	mockClient.On("Describe", mock.Anything, "default").Return(
		&workflowservice.DescribeNamespaceResponse{},
		nil,
	)
	mockClient.On("Close").Return()

	newNamespaceClientFn = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}
	sleepFn = func(time.Duration) {}
	nowFn = time.Now

	t.Cleanup(func() {
		newNamespaceClientFn = originalNewNamespaceClient
		sleepFn = originalSleep
		nowFn = originalNow
	})

	err := ensureNamespaceReadyWithRetry("default")
	require.NoError(t, err)
	mockClient.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
}

func TestEnsureNamespaceReadyWithRetryRegistersOnNotFound(t *testing.T) {
	originalNewNamespaceClient := newNamespaceClientFn
	originalSleep := sleepFn
	originalNow := nowFn

	mockClient := mocks.NewNamespaceClient(t)
	callCount := 0
	mockClient.On("Describe", mock.Anything, "tenant").Return(
		func(_ context.Context, _ string) *workflowservice.DescribeNamespaceResponse {
			callCount++
			if callCount == 1 {
				return nil
			}
			return &workflowservice.DescribeNamespaceResponse{}
		},
		func(_ context.Context, _ string) error {
			if callCount == 1 {
				return &serviceerror.NamespaceNotFound{}
			}
			return nil
		},
	)
	mockClient.On("Register", mock.Anything, mock.MatchedBy(func(req *workflowservice.RegisterNamespaceRequest) bool {
		return req.GetNamespace() == "tenant"
	})).
		Return(nil)
	mockClient.On("Close").Return()

	newNamespaceClientFn = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}
	sleepFn = func(time.Duration) {}
	nowFn = time.Now

	t.Cleanup(func() {
		newNamespaceClientFn = originalNewNamespaceClient
		sleepFn = originalSleep
		nowFn = originalNow
	})

	err := ensureNamespaceReadyWithRetry("tenant")
	require.NoError(t, err)
	mockClient.AssertCalled(t, "Register", mock.Anything, mock.Anything)
}

func TestEnsureNamespaceReadyWithRetryTimeout(t *testing.T) {
	originalNewNamespaceClient := newNamespaceClientFn
	originalSleep := sleepFn
	originalNow := nowFn

	mockClient := mocks.NewNamespaceClient(t)
	mockClient.On("Describe", mock.Anything, "timeout").Return(
		(*workflowservice.DescribeNamespaceResponse)(nil),
		errors.New("boom"),
	)
	mockClient.On("Close").Return()

	start := time.Now()
	nowCalls := 0
	newNamespaceClientFn = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}
	sleepFn = func(time.Duration) {}
	nowFn = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return start
		}
		return start.Add(91 * time.Second)
	}

	t.Cleanup(func() {
		newNamespaceClientFn = originalNewNamespaceClient
		sleepFn = originalSleep
		nowFn = originalNow
	})

	err := ensureNamespaceReadyWithRetry("timeout")
	require.Error(t, err)
}

func TestExecuteWorkerManagerWorkflowSuccess(t *testing.T) {
	origStart := workerManagerStartWorkflow
	origClient := workerManagerTemporalClient
	origWait := workerManagerWaitForWorkflowResult

	t.Cleanup(func() {
		workerManagerStartWorkflow = origStart
		workerManagerTemporalClient = origClient
		workerManagerWaitForWorkflowResult = origWait
	})

	var gotNamespace string
	var gotInput workflowengine.WorkflowInput

	workerManagerStartWorkflow = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		gotNamespace = namespace
		gotInput = input
		return workflowengine.WorkflowResult{
			WorkflowID:    "wf-1",
			WorkflowRunID: "run-1",
		}, nil
	}

	workerManagerTemporalClient = func(namespace string) (client.Client, error) {
		require.Equal(t, "default", namespace)
		return &mocks.Client{}, nil
	}

	workerManagerWaitForWorkflowResult = func(c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		require.NotNil(t, c)
		require.Equal(t, "wf-1", workflowID)
		require.Equal(t, "run-1", runID)
		return workflowengine.WorkflowResult{}, nil
	}

	err := executeWorkerManagerWorkflow("org-1", "org-0", "https://app.example")
	require.NoError(t, err)
	require.Equal(t, "default", gotNamespace)
	require.NotNil(t, gotInput.ActivityOptions)

	payload, ok := gotInput.Payload.(workflows.WorkerManagerWorkflowPayload)
	require.True(t, ok)
	require.Equal(t, "org-1", payload.Namespace)
	require.Equal(t, "org-0", payload.OldNamespace)
	require.Equal(t, "https://app.example", gotInput.Config["app_url"])
}

func TestExecuteWorkerManagerWorkflowStartError(t *testing.T) {
	origStart := workerManagerStartWorkflow

	t.Cleanup(func() {
		workerManagerStartWorkflow = origStart
	})

	workerManagerStartWorkflow = func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("boom")
	}

	err := executeWorkerManagerWorkflow("org-1", "", "http://app")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to start workflow")
}

func TestExecuteWorkerManagerWorkflowTemporalClientError(t *testing.T) {
	origStart := workerManagerStartWorkflow
	origClient := workerManagerTemporalClient

	t.Cleanup(func() {
		workerManagerStartWorkflow = origStart
		workerManagerTemporalClient = origClient
	})

	workerManagerStartWorkflow = func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	workerManagerTemporalClient = func(_ string) (client.Client, error) {
		return nil, errors.New("no client")
	}

	err := executeWorkerManagerWorkflow("org-1", "", "http://app")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to create client")
}

func TestExecuteWorkerManagerWorkflowWaitError(t *testing.T) {
	origStart := workerManagerStartWorkflow
	origClient := workerManagerTemporalClient
	origWait := workerManagerWaitForWorkflowResult

	t.Cleanup(func() {
		workerManagerStartWorkflow = origStart
		workerManagerTemporalClient = origClient
		workerManagerWaitForWorkflowResult = origWait
	})

	workerManagerStartWorkflow = func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{WorkflowID: "wf-1", WorkflowRunID: "run-1"}, nil
	}
	workerManagerTemporalClient = func(_ string) (client.Client, error) {
		return &mocks.Client{}, nil
	}
	workerManagerWaitForWorkflowResult = func(_ client.Client, _ string, _ string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("wait failed")
	}

	err := executeWorkerManagerWorkflow("org-1", "", "http://app")
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to start mobile automation worker")
}
