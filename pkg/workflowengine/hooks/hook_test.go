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

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
)

const testDataDir = "../../../test_pb_data"

type fakeWorkflow struct {
	name string
}

func (f fakeWorkflow) Name() string {
	return f.name
}

func (f fakeWorkflow) Workflow(
	workflow.Context,
	workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return workflowengine.WorkflowResult{}, nil
}

func (f fakeWorkflow) ExecuteWorkflow(
	workflow.Context,
	workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	return workflowengine.WorkflowResult{}, nil
}

func (f fakeWorkflow) GetOptions() workflow.ActivityOptions {
	return workflow.ActivityOptions{}
}

type fakeActivity struct {
	name string
}

func (f fakeActivity) Name() string {
	return f.name
}

func (f fakeActivity) NewActivityError(string, string, ...any) error {
	return errors.New("activity error")
}

func (f fakeActivity) NewNonRetryableActivityError(string, string, ...any) error {
	return errors.New("activity error")
}

func (f fakeActivity) NewMissingOrInvalidPayloadError(error) error {
	return errors.New("activity error")
}

func (f fakeActivity) Execute(
	context.Context,
	workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	return workflowengine.ActivityResult{}, nil
}

type fakeWorker struct {
	registeredWorkflows  []string
	registeredActivities []string
	runCalled            bool
}

func (f *fakeWorker) RegisterWorkflow(interface{}) {}

func (f *fakeWorker) RegisterWorkflowWithOptions(_ interface{}, options workflow.RegisterOptions) {
	f.registeredWorkflows = append(f.registeredWorkflows, options.Name)
}

func (f *fakeWorker) RegisterDynamicWorkflow(interface{}, workflow.DynamicRegisterOptions) {}

func (f *fakeWorker) RegisterActivity(interface{}) {}

func (f *fakeWorker) RegisterActivityWithOptions(_ interface{}, options activity.RegisterOptions) {
	f.registeredActivities = append(f.registeredActivities, options.Name)
}

func (f *fakeWorker) RegisterDynamicActivity(interface{}, activity.DynamicRegisterOptions) {}

func (f *fakeWorker) RegisterNexusService(*nexus.Service) {}

func (f *fakeWorker) Start() error { return nil }

func (f *fakeWorker) Run(interruptCh <-chan interface{}) error {
	f.runCalled = true
	<-interruptCh
	return nil
}

func (f *fakeWorker) Stop() {}

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

func TestWorkersHookStartsWorkersAndShutdowns(t *testing.T) {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: t.TempDir(),
	})

	origFetch := fetchNamespacesFn
	origEnsure := ensureNamespaceReadyFn
	origStartAll := startAllWorkersByNamespace
	origStartWorkerManager := startWorkerManagerWorkflow
	origShutdown := shutdownTemporalClientsFn

	t.Cleanup(func() {
		fetchNamespacesFn = origFetch
		ensureNamespaceReadyFn = origEnsure
		startAllWorkersByNamespace = origStartAll
		startWorkerManagerWorkflow = origStartWorkerManager
		shutdownTemporalClientsFn = origShutdown
	})

	fetchNamespacesFn = func(_ core.App) ([]string, error) {
		return []string{"default", "org-1"}, nil
	}

	ensureCalls := make(chan string, 2)
	ensureNamespaceReadyFn = func(ns string) error {
		ensureCalls <- ns
		return nil
	}

	startCalls := make(chan string, 2)
	startAllWorkersByNamespace = func(ns string) {
		startCalls <- ns
	}

	managerCalls := make(chan string, 2)
	startWorkerManagerWorkflow = func(_ core.App, ns, _ string) {
		managerCalls <- ns
	}

	shutdownCalled := make(chan struct{}, 1)
	shutdownTemporalClientsFn = func() {
		shutdownCalled <- struct{}{}
	}

	WorkersHook(app)

	serveEvent := &core.ServeEvent{App: app}
	serveErr := app.OnServe().Trigger(serveEvent, func(_ *core.ServeEvent) error {
		return nil
	})
	require.NoError(t, serveErr)

	gotEnsure := map[string]struct{}{}
	for i := 0; i < 2; i++ {
		select {
		case ns := <-ensureCalls:
			gotEnsure[ns] = struct{}{}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for ensure namespace call")
		}
	}
	require.Contains(t, gotEnsure, "default")
	require.Contains(t, gotEnsure, "org-1")

	gotWorkers := map[string]struct{}{}
	for i := 0; i < 2; i++ {
		select {
		case ns := <-startCalls:
			gotWorkers[ns] = struct{}{}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for start worker call")
		}
	}
	require.Contains(t, gotWorkers, "default")
	require.Contains(t, gotWorkers, "org-1")

	gotManagers := map[string]struct{}{}
	for i := 0; i < 2; i++ {
		select {
		case ns := <-managerCalls:
			gotManagers[ns] = struct{}{}
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for start worker manager call")
		}
	}
	require.Contains(t, gotManagers, "default")
	require.Contains(t, gotManagers, "org-1")

	terminateErr := app.OnTerminate().Trigger(
		&core.TerminateEvent{App: app},
		func(_ *core.TerminateEvent) error { return nil },
	)
	require.NoError(t, terminateErr)

	select {
	case <-shutdownCalled:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for shutdown call")
	}
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

func TestStartWorkerRegistersEntries(t *testing.T) {
	origNewWorker := newWorkerFn

	t.Cleanup(func() {
		newWorkerFn = origNewWorker
	})

	fw := &fakeWorker{}
	newWorkerFn = func(_ client.Client, _ string, _ worker.Options) worker.Worker {
		return fw
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)

	go startWorker(ctx, nil, workerConfig{
		TaskQueue: "queue-a",
		Workflows: []workflowengine.Workflow{
			fakeWorkflow{name: "wf-1"},
		},
		Activities: []workflowengine.ExecutableActivity{
			fakeActivity{name: "act-1"},
		},
	}, &wg)

	require.Eventually(t, func() bool {
		return fw.runCalled
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()

	require.Equal(t, []string{"wf-1"}, fw.registeredWorkflows)
	require.Equal(t, []string{"act-1"}, fw.registeredActivities)
}

func TestStartPipelineWorkerRegistersRegistryEntries(t *testing.T) {
	origNewWorker := newWorkerFn
	origRegistry := registry.Registry
	origInternal := registry.PipelineInternalRegistry
	origDenylist := registry.PipelineWorkerDenylist

	t.Cleanup(func() {
		newWorkerFn = origNewWorker
		registry.Registry = origRegistry
		registry.PipelineInternalRegistry = origInternal
		registry.PipelineWorkerDenylist = origDenylist
	})

	fw := &fakeWorker{}
	newWorkerFn = func(_ client.Client, _ string, _ worker.Options) worker.Worker {
		return fw
	}

	registry.Registry = map[string]registry.TaskFactory{
		"skip-task": {
			Kind:    registry.TaskActivity,
			NewFunc: func() any { return fakeActivity{name: "skip-act"} },
		},
		"activity-task": {
			Kind:    registry.TaskActivity,
			NewFunc: func() any { return fakeActivity{name: "activity-act"} },
		},
		"workflow-task": {
			Kind:    registry.TaskWorkflow,
			NewFunc: func() any { return fakeWorkflow{name: "workflow-wf"} },
		},
	}
	registry.PipelineInternalRegistry = map[string]registry.TaskFactory{
		"internal-activity": {
			Kind:    registry.TaskActivity,
			NewFunc: func() any { return fakeActivity{name: "internal-act"} },
		},
		"internal-workflow": {
			Kind:    registry.TaskWorkflow,
			NewFunc: func() any { return fakeWorkflow{name: "internal-wf"} },
		},
	}
	registry.PipelineWorkerDenylist = map[string]struct{}{
		"skip-task": {},
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go startPipelineWorker(ctx, nil, &wg)

	require.Eventually(t, func() bool {
		return fw.runCalled
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()

	pipelineWf := pipeline.NewPipelineWorkflow().Name()
	debugAct := pipeline.NewDebugActivity().Name()

	require.Contains(t, fw.registeredWorkflows, pipelineWf)
	require.Contains(t, fw.registeredWorkflows, "workflow-wf")
	require.Contains(t, fw.registeredWorkflows, "internal-wf")
	require.NotContains(t, fw.registeredWorkflows, "skip-act")

	require.Contains(t, fw.registeredActivities, debugAct)
	require.Contains(t, fw.registeredActivities, "activity-act")
	require.Contains(t, fw.registeredActivities, "internal-act")
	require.NotContains(t, fw.registeredActivities, "skip-act")
}

func TestStartWorkerManagerWorkflowInvokesExecute(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()
	app.Settings().Meta.AppURL = "https://app.example"

	origExec := executeWorkerManagerWorkflowFn

	t.Cleanup(func() {
		executeWorkerManagerWorkflowFn = origExec
	})

	called := make(chan struct{}, 1)
	executeWorkerManagerWorkflowFn = func(namespace, oldNamespace, appURL string) error {
		require.Equal(t, "org-1", namespace)
		require.Equal(t, "org-0", oldNamespace)
		require.Equal(t, "https://app.example", appURL)
		called <- struct{}{}
		return nil
	}

	StartWorkerManagerWorkflow(app, "org-1", "org-0")

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for worker manager workflow")
	}
}
