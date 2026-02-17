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
		return req.Namespace == "tenant"
	})).Return(nil)
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
