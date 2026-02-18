// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
)

type fakeNamespaceClient struct {
	describeErrs  []error
	describeCalls int
}

func (f *fakeNamespaceClient) Register(
	_ context.Context,
	_ *workflowservice.RegisterNamespaceRequest,
) error {
	return nil
}

func (f *fakeNamespaceClient) Describe(
	_ context.Context,
	_ string,
) (*workflowservice.DescribeNamespaceResponse, error) {
	f.describeCalls++
	if len(f.describeErrs) > 0 {
		err := f.describeErrs[0]
		f.describeErrs = f.describeErrs[1:]
		return nil, err
	}
	return &workflowservice.DescribeNamespaceResponse{}, nil
}

func (f *fakeNamespaceClient) Update(
	_ context.Context,
	_ *workflowservice.UpdateNamespaceRequest,
) error {
	return nil
}

func (f *fakeNamespaceClient) Close() {}

func TestWaitForNamespaceReadyImmediateSuccess(t *testing.T) {
	client := &fakeNamespaceClient{}

	err := waitForNamespaceReady(client, "default", time.Second)
	require.NoError(t, err)
	require.Equal(t, 1, client.describeCalls)
}

func TestWaitForNamespaceReadyRetriesThenSucceeds(t *testing.T) {
	client := &fakeNamespaceClient{describeErrs: []error{errors.New("transient")}}

	err := waitForNamespaceReady(client, "default", 3*time.Second)
	require.NoError(t, err)
	require.GreaterOrEqual(t, client.describeCalls, 2)
}

func TestWaitForNamespaceReadyTimeout(t *testing.T) {
	client := &fakeNamespaceClient{describeErrs: []error{errors.New("still failing")}}

	err := waitForNamespaceReady(client, "default", -time.Second)
	require.Error(t, err)
	require.Equal(t, 1, client.describeCalls)
}

func TestEnsureNamespaceAndWorkersCreatesNamespace(t *testing.T) {
	origClient := newNamespaceClient
	origWait := waitForNamespaceReadyFn
	origStart := startWorkersByNamespaceFn
	t.Cleanup(func() {
		newNamespaceClient = origClient
		waitForNamespaceReadyFn = origWait
		startWorkersByNamespaceFn = origStart
	})

	mockClient := mocks.NewNamespaceClient(t)
	mockClient.
		On("Describe", mock.Anything, "tenant").
		Return((*workflowservice.DescribeNamespaceResponse)(nil), &serviceerror.NamespaceNotFound{}).
		Once()
	mockClient.On("Register", mock.Anything, mock.Anything).Return(nil).Once()
	mockClient.On("Close").Return()

	newNamespaceClient = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}

	waitCalled := make(chan struct{}, 1)
	waitForNamespaceReadyFn = func(_ client.NamespaceClient, namespace string, _ time.Duration) error {
		require.Equal(t, "tenant", namespace)
		waitCalled <- struct{}{}
		return nil
	}

	started := make(chan string, 1)
	startWorkersByNamespaceFn = func(namespace string) {
		started <- namespace
	}

	ensureNamespaceAndWorkers("tenant")

	select {
	case <-waitCalled:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for namespace readiness call")
	}

	select {
	case ns := <-started:
		require.Equal(t, "tenant", ns)
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for workers start")
	}
}

func TestEnsureNamespaceAndWorkersSkipsExisting(t *testing.T) {
	origClient := newNamespaceClient
	origWait := waitForNamespaceReadyFn
	origStart := startWorkersByNamespaceFn
	t.Cleanup(func() {
		newNamespaceClient = origClient
		waitForNamespaceReadyFn = origWait
		startWorkersByNamespaceFn = origStart
	})

	mockClient := mocks.NewNamespaceClient(t)
	mockClient.
		On("Describe", mock.Anything, "tenant").
		Return(&workflowservice.DescribeNamespaceResponse{}, nil).
		Once()
	mockClient.On("Close").Return()

	newNamespaceClient = func(_ client.Options) (client.NamespaceClient, error) {
		return mockClient, nil
	}

	waitForNamespaceReadyFn = func(_ client.NamespaceClient, _ string, _ time.Duration) error {
		require.Fail(t, "waitForNamespaceReady should not be called")
		return nil
	}

	startWorkersByNamespaceFn = func(_ string) {
		require.Fail(t, "startWorkersByNamespace should not be called")
	}

	ensureNamespaceAndWorkers("tenant")
}

func TestHookNamespaceOrgsAfterCreate(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})

	origEnsure := ensureNamespaceAndWorkersFn
	origStartManager := startWorkerManagerFn
	t.Cleanup(func() {
		ensureNamespaceAndWorkersFn = origEnsure
		startWorkerManagerFn = origStartManager
	})

	var ensured string
	ensureNamespaceAndWorkersFn = func(namespace string) {
		ensured = namespace
	}

	var started struct {
		namespace    string
		oldNamespace string
	}
	startWorkerManagerFn = func(_ core.App, namespace, oldNamespace string) {
		started.namespace = namespace
		started.oldNamespace = oldNamespace
	}

	HookNamespaceOrgs(app)

	collection := core.NewBaseCollection("organizations")
	record := core.NewRecord(collection)
	record.Set("canonified_name", "org-1")
	event := &core.RecordEvent{App: app}
	event.Record = record

	err := app.OnRecordAfterCreateSuccess("organizations").Trigger(
		event,
		func(_ *core.RecordEvent) error { return nil },
	)
	require.NoError(t, err)
	require.Equal(t, "org-1", ensured)
	require.Equal(t, "org-1", started.namespace)
	require.Equal(t, "", started.oldNamespace)
}
