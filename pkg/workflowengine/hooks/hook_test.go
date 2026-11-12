// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/mobilerunner"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/authorsession"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
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

func TestSleepWithContext(t *testing.T) {
	require.True(t, sleepWithContext(context.Background(), time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.False(t, sleepWithContext(ctx, time.Hour))
}

func TestWorkerManagerRecordQueries(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	organizations, err := workerManagerAllOrganizationRecords(app)
	require.NoError(t, err)
	require.NotEmpty(t, organizations)

	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	if collection.Fields.GetByName("admin_managed") == nil {
		collection.Fields.Add(&core.BoolField{Name: "admin_managed"})
	}
	if collection.Fields.GetByName("disabled") == nil {
		collection.Fields.Add(&core.BoolField{Name: "disabled"})
	}
	require.NoError(t, app.Save(collection))
	newRunner := func(name, ip, port string, published, adminManaged, disabled, online bool) {
		record := core.NewRecord(collection)
		record.Set("owner", organizations[0].Id)
		record.Set("name", name)
		record.Set("canonified_name", name)
		record.Set("type", "android_emulator")
		record.Set("ip", ip)
		record.Set("port", port)
		record.Set("published", published)
		record.Set("admin_managed", adminManaged)
		record.Set("disabled", disabled)
		record.Set("online", online)
		require.NoError(t, app.Save(record))
	}

	newRunner("admin-runner", "https://admin.test/", "8080", false, true, false, true)
	newRunner("admin-disabled", "https://admin-disabled.test", "", false, true, true, true)
	newRunner("admin-offline", "https://admin-offline.test", "", false, true, false, false)
	newRunner("published-runner", "https://published.test", "", true, false, false, true)
	newRunner("published-disabled", "https://published-disabled.test", "", true, false, true, true)
	newRunner("published-offline", "https://published-offline.test", "", true, false, false, false)
	newRunner("empty-runner", " ", "", true, false, false, true)
	// A runner whose owner organization is gone has no identifier; it must be
	// skipped instead of failing the startup and organization hooks.
	orphan := core.NewRecord(collection)
	orphan.Set("owner", "missingorg00000")
	orphan.Set("name", "orphan-runner")
	orphan.Set("canonified_name", "orphan-runner")
	orphan.Set("type", "android_emulator")
	orphan.Set("ip", "https://orphan.test")
	orphan.Set("admin_managed", true)
	orphan.Set("online", true)
	require.NoError(t, app.SaveNoValidate(orphan))

	orgName := organizations[0].GetString("canonified_name")

	adminIDs, err := WorkerManagerAdminRunnerIDs(app)
	require.NoError(t, err)
	require.Equal(t, []string{orgName + "/admin-runner"}, adminIDs)

	publishedIDs, err := WorkerManagerPublishedNonAdminRunnerIDs(app)
	require.NoError(t, err)
	require.Equal(t, []string{orgName + "/published-runner"}, publishedIDs)
}

type fakeActivity struct {
	name string
}

func (f fakeActivity) Name() string {
	return f.name
}

func (f fakeActivity) NewActivityError(workflowengine.ActivityError) error {
	return errors.New("activity error")
}

func (f fakeActivity) NewNonRetryableActivityError(workflowengine.ActivityError) error {
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
	runCalled            atomic.Bool
}

type retryTestWorker struct {
	runFn func(<-chan interface{}) error
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
	f.runCalled.Store(true)
	<-interruptCh
	return nil
}

func (f *fakeWorker) Stop() {}

func (*retryTestWorker) RegisterWorkflow(interface{}) {}

func (*retryTestWorker) RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions) {}

func (*retryTestWorker) RegisterDynamicWorkflow(interface{}, workflow.DynamicRegisterOptions) {}

func (*retryTestWorker) RegisterActivity(interface{}) {}

func (*retryTestWorker) RegisterActivityWithOptions(interface{}, activity.RegisterOptions) {}

func (*retryTestWorker) RegisterDynamicActivity(interface{}, activity.DynamicRegisterOptions) {}

func (*retryTestWorker) RegisterNexusService(*nexus.Service) {}

func (*retryTestWorker) Start() error { return nil }

func (w *retryTestWorker) Run(interruptCh <-chan interface{}) error {
	return w.runFn(interruptCh)
}

func (*retryTestWorker) Stop() {}

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
	t.Setenv(TemporalWorkersDisabledEnv, "")

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: t.TempDir(),
	})

	origFetch := fetchNamespacesFn
	origOrgRecords := workerManagerOrgRecordsFn
	origAdminIDs := adminRunnerIDsFn
	origPublishedIDs := publishedRunnerIDsFn
	origEnsure := ensureNamespaceReadyFn
	origStartAll := startAllWorkersByNamespace
	origStartWorkerManager := startWorkerManagerWorkflow
	origShutdown := shutdownTemporalClientsFn
	origAgentIDs := runnerAgentIDsFn
	origEnsureAgent := ensureRunnerAgentNamespace

	t.Cleanup(func() {
		runnerAgentIDsFn = origAgentIDs
		ensureRunnerAgentNamespace = origEnsureAgent
		fetchNamespacesFn = origFetch
		workerManagerOrgRecordsFn = origOrgRecords
		adminRunnerIDsFn = origAdminIDs
		publishedRunnerIDsFn = origPublishedIDs
		ensureNamespaceReadyFn = origEnsure
		startAllWorkersByNamespace = origStartAll
		startWorkerManagerWorkflow = origStartWorkerManager
		shutdownTemporalClientsFn = origShutdown
	})

	fetchNamespacesFn = func(_ core.App) ([]string, error) {
		return []string{"default", "org-1"}, nil
	}
	workerManagerOrgRecordsFn = func(_ core.App) ([]*core.Record, error) {
		collection := core.NewBaseCollection("organizations")
		record := core.NewRecord(collection)
		record.Set("canonified_name", "org-1")
		record.Set("published", false)
		return []*core.Record{record}, nil
	}
	adminRunnerIDsFn = func(_ core.App) ([]string, error) {
		return []string{"admin/runner"}, nil
	}
	publishedRunnerIDsFn = func(_ core.App) ([]string, error) {
		return []string{"tenant/runner"}, nil
	}

	runnerAgentIDsFn = func(core.App) ([]string, error) {
		return []string{"boot"}, nil
	}
	var agentPrepared atomic.Bool
	ensureRunnerAgentNamespace = func(_ core.App, runnerID string) {
		require.Equal(t, "boot", runnerID)
		agentPrepared.Store(true)
	}

	ensureCalls := make(chan string, 2)
	ensureNamespaceReadyFn = func(ns string) error {
		ensureCalls <- ns
		return nil
	}

	startCalls := make(chan string, 2)
	startAllWorkersByNamespace = func(_ core.App, ns string) {
		startCalls <- ns
	}

	type workerManagerCall struct {
		namespace string
		runnerIDs []string
	}
	managerCalls := make(chan workerManagerCall, 2)
	startWorkerManagerWorkflow = func(ns, _ string, runnerIDs []string) {
		managerCalls <- workerManagerCall{namespace: ns, runnerIDs: runnerIDs}
	}

	shutdownCalled := make(chan struct{}, 1)
	shutdownTemporalClientsFn = func() {
		shutdownCalled <- struct{}{}
	}

	WorkersHook(app)

	serveEvent := &core.ServeEvent{App: app}
	serveErr := app.OnServe().Trigger(serveEvent, func(_ *core.ServeEvent) error {
		// Runner agent namespaces are prepared before Credimi serves.
		require.True(t, agentPrepared.Load())
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

	gotManagers := map[string][]string{}
	for i := 0; i < 2; i++ {
		select {
		case call := <-managerCalls:
			gotManagers[call.namespace] = call.runnerIDs
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for start worker manager call")
		}
	}
	require.Equal(t, []string{"admin/runner"}, gotManagers["default"])
	require.Equal(t, []string{"admin/runner"}, gotManagers["org-1"])

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

func TestWorkersHookSkipsWhenTemporalWorkersDisabled(t *testing.T) {
	t.Setenv(TemporalWorkersDisabledEnv, "1")

	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: t.TempDir(),
	})

	origFetch := fetchNamespacesFn
	t.Cleanup(func() {
		fetchNamespacesFn = origFetch
	})
	fetchNamespacesFn = func(_ core.App) ([]string, error) {
		t.Fatal("fetchNamespaces should not be called when workers are disabled")
		return nil, nil
	}

	WorkersHook(app)

	serveErr := app.OnServe().Trigger(
		&core.ServeEvent{App: app},
		func(_ *core.ServeEvent) error { return nil },
	)
	require.NoError(t, serveErr)
}

func TestStartAllWorkersByNamespaceDefault(t *testing.T) {
	workerSets = sync.Map{}

	originalGetTemporalClient := getTemporalClient
	originalStartWorker := startWorkerFn
	originalStartPipelineWorker := startPipelineWorkerFn

	t.Cleanup(func() {
		getTemporalClient = originalGetTemporalClient
		startWorkerFn = originalStartWorker
		startPipelineWorkerFn = originalStartPipelineWorker
		workerSets = sync.Map{}
	})

	getTemporalClient = func(namespace string) (client.Client, error) {
		return nil, nil
	}

	workerCh := make(chan string, len(defaultWorkers(nil)))
	pipelineCh := make(chan struct{}, 1)

	startWorkerFn = func(_ context.Context, _ client.Client, config workerConfig, wg *sync.WaitGroup) {
		workerCh <- config.TaskQueue
		wg.Done()
	}
	startPipelineWorkerFn = func(_ context.Context, _ core.App, _ client.Client, wg *sync.WaitGroup) {
		pipelineCh <- struct{}{}
		wg.Done()
	}

	StartAllWorkersByNamespace(nil, "default")

	for i := 0; i < len(defaultWorkers(nil)); i++ {
		<-workerCh
	}
	<-pipelineCh

	_, ok := workerSets.Load("default")
	require.True(t, ok)

	StopAllWorkersByNamespace("default")
	_, ok = workerSets.Load("default")
	require.False(t, ok)
}

func TestStartAllWorkersByNamespaceSkipsWhenTemporalWorkersDisabled(t *testing.T) {
	t.Setenv(TemporalWorkersDisabledEnv, "1")
	workerSets = sync.Map{}

	originalGetTemporalClient := getTemporalClient
	t.Cleanup(func() {
		getTemporalClient = originalGetTemporalClient
		workerSets = sync.Map{}
	})

	getTemporalClient = func(_ string) (client.Client, error) {
		require.Fail(t, "getTemporalClient should not be called")
		return nil, nil
	}

	StartAllWorkersByNamespace(nil, "default")

	_, ok := workerSets.Load("default")
	require.False(t, ok)
}

func TestStartAllWorkersByNamespaceOrg(t *testing.T) {
	workerSets = sync.Map{}

	originalGetTemporalClient := getTemporalClient
	originalStartWorker := startWorkerFn
	originalStartPipelineWorker := startPipelineWorkerFn

	t.Cleanup(func() {
		getTemporalClient = originalGetTemporalClient
		startWorkerFn = originalStartWorker
		startPipelineWorkerFn = originalStartPipelineWorker
		workerSets = sync.Map{}
	})

	getTemporalClient = func(namespace string) (client.Client, error) {
		return nil, nil
	}

	workerCh := make(chan string, len(orgWorkers(nil)))
	pipelineCh := make(chan struct{}, 1)

	startWorkerFn = func(_ context.Context, _ client.Client, config workerConfig, wg *sync.WaitGroup) {
		workerCh <- config.TaskQueue
		wg.Done()
	}
	startPipelineWorkerFn = func(_ context.Context, _ core.App, _ client.Client, wg *sync.WaitGroup) {
		pipelineCh <- struct{}{}
		wg.Done()
	}

	StartAllWorkersByNamespace(nil, "acme-org")

	gotWorkers := make(map[string]struct{}, len(orgWorkers(nil)))
	for i := 0; i < len(orgWorkers(nil)); i++ {
		gotWorkers[<-workerCh] = struct{}{}
	}
	expectedWorkers := make(map[string]struct{}, len(orgWorkers(nil)))
	for _, worker := range orgWorkers(nil) {
		expectedWorkers[worker.TaskQueue] = struct{}{}
	}
	require.Equal(t, expectedWorkers, gotWorkers)
	<-pipelineCh
	_, ok := workerSets.Load("acme-org")
	require.True(t, ok)

	StopAllWorkersByNamespace("acme-org")
	_, ok = workerSets.Load("acme-org")
	require.False(t, ok)
}

func TestStartAllWorkersByNamespaceTwiceStopsFirstSet(t *testing.T) {
	workerSets = sync.Map{}

	originalGetTemporalClient := getTemporalClient
	originalStartWorker := startWorkerFn
	originalStartPipelineWorker := startPipelineWorkerFn
	t.Cleanup(func() {
		getTemporalClient = originalGetTemporalClient
		startWorkerFn = originalStartWorker
		startPipelineWorkerFn = originalStartPipelineWorker
		StopAllWorkers()
		workerSets = sync.Map{}
	})

	getTemporalClient = func(string) (client.Client, error) {
		return nil, nil
	}

	var mu sync.Mutex
	var ctxs []context.Context
	runUntilCancelled := func(ctx context.Context, wg *sync.WaitGroup) {
		mu.Lock()
		ctxs = append(ctxs, ctx)
		mu.Unlock()
		go func() {
			<-ctx.Done()
			wg.Done()
		}()
	}
	startWorkerFn = func(ctx context.Context, _ client.Client, _ workerConfig, wg *sync.WaitGroup) {
		runUntilCancelled(ctx, wg)
	}
	startPipelineWorkerFn = func(ctx context.Context, _ core.App, _ client.Client, wg *sync.WaitGroup) {
		runUntilCancelled(ctx, wg)
	}

	StartAllWorkersByNamespace(nil, "acme-org")
	first, ok := workerSets.Load("acme-org")
	require.True(t, ok)
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ctxs) == len(orgWorkers(nil))+1
	}, time.Second, 5*time.Millisecond)

	StartAllWorkersByNamespace(nil, "acme-org")

	// The first set is cancelled and fully drained before the second starts.
	select {
	case <-first.(*namespaceWorkers).done:
	default:
		t.Fatal("first worker set must be drained before restarting")
	}
	second, ok := workerSets.Load("acme-org")
	require.True(t, ok)
	require.NotSame(t, first, second)

	mu.Lock()
	firstCount := len(orgWorkers(nil)) + 1
	for i, ctx := range ctxs[:firstCount] {
		require.Error(t, ctx.Err(), "first set worker %d must be cancelled", i)
	}
	mu.Unlock()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(ctxs) == 2*firstCount
	}, time.Second, 5*time.Millisecond)
	mu.Lock()
	for _, ctx := range ctxs[firstCount:] {
		require.NoError(t, ctx.Err())
	}
	mu.Unlock()
}

// stubNamespaceSetup replaces the namespace client, sleep and search-attribute
// seams; it returns the namespaces passed to addSearchAttributesFn.
func stubNamespaceSetup(
	t *testing.T,
	nc client.NamespaceClient,
	addErr error,
) *[]string {
	t.Helper()
	originalNewNamespaceClient := newNamespaceClientFn
	originalSleep := sleepFn
	originalNow := nowFn
	originalAddSearchAttributes := addSearchAttributesFn
	t.Cleanup(func() {
		newNamespaceClientFn = originalNewNamespaceClient
		sleepFn = originalSleep
		nowFn = originalNow
		addSearchAttributesFn = originalAddSearchAttributes
	})

	var added []string
	newNamespaceClientFn = func() (client.NamespaceClient, error) {
		return nc, nil
	}
	sleepFn = func(time.Duration) {}
	nowFn = time.Now
	addSearchAttributesFn = func(_ context.Context, namespace string) error {
		added = append(added, namespace)
		return addErr
	}
	return &added
}

// createAgentRunner saves a runner owned by the first organization.
func createAgentRunner(
	t *testing.T,
	app core.App,
	name string,
	enabled, disabled bool,
) *core.Record {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("mobile_runners")
	require.NoError(t, err)
	org, err := app.FindFirstRecordByFilter("organizations", "")
	require.NoError(t, err)
	record := core.NewRecord(collection)
	record.Set("owner", org.Id)
	record.Set("name", name)
	record.Set("ip", "127.0.0.1")
	record.Set("agent_enabled", enabled)
	record.Set("disabled", disabled)
	require.NoError(t, app.Save(record))
	return record
}

// agentNamespaceReadyOnRecord reads the runner's persisted readiness.
func agentNamespaceReadyOnRecord(t *testing.T, app core.App, runnerID string) bool {
	t.Helper()
	runner, err := app.FindRecordById("mobile_runners", runnerID)
	require.NoError(t, err)
	return runner.GetBool(mobilerunner.AgentNamespaceReadyField)
}

func TestRunnerAgentIDs(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	before, err := RunnerAgentIDs(app)
	require.NoError(t, err)

	enabled := createAgentRunner(t, app, "agent-on", true, false)
	createAgentRunner(t, app, "agent-off", false, false)
	createAgentRunner(t, app, "agent-on-disabled", true, true)

	got, err := RunnerAgentIDs(app)
	require.NoError(t, err)
	require.ElementsMatch(t, append(before, enabled.Id), got)
}

func TestPrepareRunnerAgentNamespacesBoundsConcurrency(t *testing.T) {
	orig := ensureRunnerAgentNamespace
	t.Cleanup(func() { ensureRunnerAgentNamespace = orig })
	var running, peak atomic.Int32
	prepared := make(chan string, 20)
	ensureRunnerAgentNamespace = func(_ core.App, runnerID string) {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		running.Add(-1)
		prepared <- runnerID
	}

	ids := make([]string, 20)
	for i := range ids {
		ids[i] = string(rune('a' + i))
	}
	prepareRunnerAgentNamespaces(nil, ids)

	require.Len(t, prepared, len(ids))
	require.LessOrEqual(t, peak.Load(), int32(runnerAgentPrepareConcurrency))
}

func TestPrepareRunnerAgentNamespacesStopsWaitingAfterTimeout(t *testing.T) {
	origEnsure := ensureRunnerAgentNamespace
	origTimeout := runnerAgentPrepareTimeout
	release := make(chan struct{})
	t.Cleanup(func() {
		close(release)
		ensureRunnerAgentNamespace = origEnsure
		runnerAgentPrepareTimeout = origTimeout
	})
	runnerAgentPrepareTimeout = 10 * time.Millisecond
	var fast atomic.Bool
	ensureRunnerAgentNamespace = func(_ core.App, runnerID string) {
		if runnerID == "slow" {
			<-release
			return
		}
		fast.Store(true)
	}

	done := make(chan struct{})
	go func() {
		prepareRunnerAgentNamespaces(nil, []string{"slow", "fast"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("startup waited for a runner past its timeout")
	}
	require.True(t, fast.Load())
}

// stubRunnerAgentWorkers stubs the namespace preparation and the workers of
// EnsureRunnerAgentNamespace and returns a test app with a runner whose agent
// is enabled; its workers are stopped afterwards.
func stubRunnerAgentWorkers(t *testing.T, ensure func(string) error) (*tests.TestApp, string) {
	t.Helper()
	t.Setenv(TemporalWorkersDisabledEnv, "")
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	runnerID := createAgentRunner(t, app, "agent-runner", true, false).Id
	origEnsure := ensureNamespaceReadyFn
	origGetTemporalClient := getTemporalClient
	origStartWorker := startWorkerFn
	origStartPipelineWorker := startPipelineWorkerFn
	t.Cleanup(func() {
		mobilerunner.SetAgentNamespaceServed(runnerID, false)
		StopAllWorkersByNamespace(mobilerunner.AgentNamespace(runnerID))
		ensureNamespaceReadyFn = origEnsure
		getTemporalClient = origGetTemporalClient
		startWorkerFn = origStartWorker
		startPipelineWorkerFn = origStartPipelineWorker
		app.Cleanup()
	})
	ensureNamespaceReadyFn = ensure
	getTemporalClient = func(string) (client.Client, error) { return nil, nil }
	startWorkerFn = func(context.Context, client.Client, workerConfig, *sync.WaitGroup) {
		t.Error("runner agent namespaces run the pipeline worker only")
	}
	startPipelineWorkerFn = func(ctx context.Context, _ core.App, _ client.Client, wg *sync.WaitGroup) {
		defer wg.Done()
		<-ctx.Done()
	}
	return app, runnerID
}

func TestEnsureRunnerAgentNamespaceMarksReadyOnSuccess(t *testing.T) {
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error { return nil })
	namespace := mobilerunner.AgentNamespace(runnerID)

	EnsureRunnerAgentNamespace(app, runnerID)
	require.True(t, mobilerunner.AgentNamespaceServed(runnerID))
	require.True(t, agentNamespaceReadyOnRecord(t, app, runnerID))
	_, ok := workerSets.Load(namespace)
	require.True(t, ok)

	// A stop while the stored runner still has its agent on keeps it.
	StopRunnerAgentWorkers(app, runnerID)
	require.True(t, mobilerunner.AgentNamespaceServed(runnerID))
	_, ok = workerSets.Load(namespace)
	require.True(t, ok)

	setRunnerField(t, app, runnerID, "disabled", true)
	StopRunnerAgentWorkers(app, runnerID)
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
	_, ok = workerSets.Load(namespace)
	require.False(t, ok)
}

// setRunnerField saves value as the runner's field.
func setRunnerField(t *testing.T, app core.App, runnerID, field string, value any) {
	t.Helper()
	runner, err := app.FindRecordById("mobile_runners", runnerID)
	require.NoError(t, err)
	runner.Set(field, value)
	require.NoError(t, app.Save(runner))
}

func TestEnsureRunnerAgentNamespaceStopsWorkersTurnedOffMeanwhile(t *testing.T) {
	var app core.App
	var runnerID string
	// The runner is disabled between the namespace's registration and its
	// marking as ready.
	app, runnerID = stubRunnerAgentWorkers(t, func(string) error {
		setRunnerField(t, app, runnerID, "disabled", true)
		return nil
	})

	EnsureRunnerAgentNamespace(app, runnerID)
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
	require.False(t, agentNamespaceReadyOnRecord(t, app, runnerID))
	_, ok := workerSets.Load(mobilerunner.AgentNamespace(runnerID))
	require.False(t, ok)
}

func TestRunnerAgentEnableAndDisableDoNotInterleave(t *testing.T) {
	registering := make(chan struct{})
	release := make(chan struct{})
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error {
		close(registering)
		<-release
		return nil
	})

	ensured := make(chan struct{})
	go func() {
		EnsureRunnerAgentNamespace(app, runnerID)
		close(ensured)
	}()
	<-registering

	// The disable lands while the namespace is being registered; its stop
	// waits for the preparation instead of running before it.
	setRunnerField(t, app, runnerID, "disabled", true)
	stopped := make(chan struct{})
	go func() {
		StopRunnerAgentWorkers(app, runnerID)
		close(stopped)
	}()
	// An ensure while the preparation is in flight returns at once.
	EnsureRunnerAgentNamespace(app, runnerID)
	select {
	case <-stopped:
		t.Fatal("stop ran while the namespace was being prepared")
	case <-time.After(50 * time.Millisecond):
	}

	close(release)
	<-ensured
	<-stopped
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
	require.False(t, agentNamespaceReadyOnRecord(t, app, runnerID))
	_, ok := workerSets.Load(mobilerunner.AgentNamespace(runnerID))
	require.False(t, ok)
}

func TestEnsureRunnerAgentNamespaceRestoresReadyFlag(t *testing.T) {
	var calls atomic.Int32
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error {
		calls.Add(1)
		return nil
	})
	EnsureRunnerAgentNamespace(app, runnerID)
	require.True(t, agentNamespaceReadyOnRecord(t, app, runnerID))

	// A save of a runner read before the namespace was marked ready.
	setRunnerField(t, app, runnerID, mobilerunner.AgentNamespaceReadyField, false)

	EnsureRunnerAgentNamespace(app, runnerID)
	require.True(t, agentNamespaceReadyOnRecord(t, app, runnerID))
	require.Equal(t, int32(1), calls.Load())
}

func TestEnsureRunnerAgentNamespaceSkipsRunnerWithAgentOff(t *testing.T) {
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error {
		t.Error("a runner whose agent is off is not prepared")
		return nil
	})
	setRunnerField(t, app, runnerID, "agent_enabled", false)

	EnsureRunnerAgentNamespace(app, runnerID)
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
}

func TestEnsureRunnerAgentNamespaceNotReadyOnError(t *testing.T) {
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error { return errors.New("denied") })

	EnsureRunnerAgentNamespace(app, runnerID)
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
	require.False(t, agentNamespaceReadyOnRecord(t, app, runnerID))
	_, ok := workerSets.Load(mobilerunner.AgentNamespace(runnerID))
	require.False(t, ok)
}

func TestEnsureRunnerAgentNamespaceRunsOnce(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error {
		calls.Add(1)
		<-release
		return nil
	})

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			EnsureRunnerAgentNamespace(app, runnerID)
		}()
	}
	require.Eventually(t, func() bool { return calls.Load() == 1 }, time.Second, time.Millisecond)
	close(release)
	wg.Wait()
	require.True(t, mobilerunner.AgentNamespaceServed(runnerID))

	// A served namespace is not prepared again.
	EnsureRunnerAgentNamespace(app, runnerID)
	require.Equal(t, int32(1), calls.Load())
}

func TestEnsureRunnerAgentNamespaceDropsCallsWhileOneIsInFlight(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error {
		calls.Add(1)
		<-release
		return errors.New("denied")
	})

	var returned atomic.Int32
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			EnsureRunnerAgentNamespace(app, runnerID)
			returned.Add(1)
		}()
	}
	// The four calls behind the failing preparation return without queueing.
	require.Eventually(t, func() bool {
		return calls.Load() == 1 && returned.Load() == 4
	}, time.Second, time.Millisecond)
	close(release)
	wg.Wait()
	require.Equal(t, int32(1), calls.Load())
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))

	// Once it finished, the next call prepares the namespace again.
	EnsureRunnerAgentNamespace(app, runnerID)
	require.Equal(t, int32(2), calls.Load())
}

func TestStopRunnerAgentWorkersForgetsDeletedRunnerLock(t *testing.T) {
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error { return nil })
	EnsureRunnerAgentNamespace(app, runnerID)
	_, ok := runnerAgentLocks.Load(runnerID)
	require.True(t, ok)

	runner, err := app.FindRecordById("mobile_runners", runnerID)
	require.NoError(t, err)
	require.NoError(t, app.Delete(runner))
	StopRunnerAgentWorkers(app, runnerID)
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
	_, ok = workerSets.Load(mobilerunner.AgentNamespace(runnerID))
	require.False(t, ok)
	_, ok = runnerAgentLocks.Load(runnerID)
	require.False(t, ok)
}

func TestLockRunnerAgentWaiterTakesCurrentLock(t *testing.T) {
	const runnerID = "forgotten-runner"
	t.Cleanup(func() { runnerAgentLocks.Delete(runnerID) })
	held := lockRunnerAgent(runnerID)

	acquired := make(chan *sync.Mutex)
	go func() { acquired <- lockRunnerAgent(runnerID) }()
	time.Sleep(10 * time.Millisecond)

	// The runner is forgotten while the waiter is blocked on its old mutex.
	runnerAgentLocks.CompareAndDelete(runnerID, held)
	held.Unlock()
	mu := <-acquired
	defer mu.Unlock()
	current, ok := runnerAgentLocks.Load(runnerID)
	require.True(t, ok)
	require.Same(t, mu, current)
}

func TestEnsureRunnerAgentNamespaceSkipsWhenTemporalWorkersDisabled(t *testing.T) {
	app, runnerID := stubRunnerAgentWorkers(t, func(string) error {
		t.Error("ensure must not run when workers are disabled")
		return nil
	})
	t.Setenv(TemporalWorkersDisabledEnv, "1")

	EnsureRunnerAgentNamespace(app, runnerID)
	require.False(t, mobilerunner.AgentNamespaceServed(runnerID))
	require.False(t, agentNamespaceReadyOnRecord(t, app, runnerID))
}

func TestEnsureNamespaceReadySuccess(t *testing.T) {
	mockClient := mocks.NewNamespaceClient(t)
	mockClient.On("Describe", mock.Anything, "default").Return(
		&workflowservice.DescribeNamespaceResponse{},
		nil,
	)
	mockClient.On("Close").Return()
	stubNamespaceSetup(t, mockClient, nil)

	err := EnsureNamespaceReady("default")
	require.NoError(t, err)
	mockClient.AssertNotCalled(t, "Register", mock.Anything, mock.Anything)
}

func TestEnsureNamespaceReadyAddsSearchAttributesAfterDescribe(t *testing.T) {
	mockClient := mocks.NewNamespaceClient(t)
	describeCalls := 0
	mockClient.On("Describe", mock.Anything, "tenant").Return(
		func(_ context.Context, _ string) *workflowservice.DescribeNamespaceResponse {
			describeCalls++
			return &workflowservice.DescribeNamespaceResponse{}
		},
		nil,
	)
	mockClient.On("Close").Return()
	added := stubNamespaceSetup(t, mockClient, nil)
	addSearchAttributes := addSearchAttributesFn
	addSearchAttributesFn = func(ctx context.Context, namespace string) error {
		require.Equal(t, 1, describeCalls, "search attributes must be added after Describe")
		return addSearchAttributes(ctx, namespace)
	}

	err := EnsureNamespaceReady("tenant")
	require.NoError(t, err)
	require.Equal(t, []string{"tenant"}, *added)
}

func TestEnsureNamespaceReadyRetriesSearchAttributeErrors(t *testing.T) {
	mockClient := mocks.NewNamespaceClient(t)
	mockClient.On("Describe", mock.Anything, "tenant").Return(
		&workflowservice.DescribeNamespaceResponse{},
		nil,
	)
	mockClient.On("Close").Return()
	added := stubNamespaceSetup(t, mockClient, errors.New("boom"))
	start := time.Now()
	nowCalls := 0
	nowFn = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return start
		}
		return start.Add(91 * time.Second)
	}

	err := EnsureNamespaceReady("tenant")
	require.ErrorContains(t, err, "boom")
	require.Equal(t, []string{"tenant"}, *added)
}

func TestEnsureNamespaceReadyRegistersOnNotFound(t *testing.T) {
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
		return req.GetNamespace() == "tenant" &&
			req.GetWorkflowExecutionRetentionPeriod().AsDuration() == namespaceRetention
	})).
		Return(nil).
		Once()
	mockClient.On("Close").Return()
	slept := false
	added := stubNamespaceSetup(t, mockClient, nil)
	sleepFn = func(time.Duration) { slept = true }

	err := EnsureNamespaceReady("tenant")
	require.NoError(t, err)
	require.False(t, slept, "a successful Register must describe again without sleeping")
	require.Equal(t, 2, callCount)
	require.Equal(t, []string{"tenant"}, *added)
}

func TestEnsureNamespaceReadyTimeout(t *testing.T) {
	mockClient := mocks.NewNamespaceClient(t)
	mockClient.On("Describe", mock.Anything, "timeout").Return(
		(*workflowservice.DescribeNamespaceResponse)(nil),
		errors.New("boom"),
	)
	mockClient.On("Close").Return()
	added := stubNamespaceSetup(t, mockClient, nil)

	start := time.Now()
	nowCalls := 0
	nowFn = func() time.Time {
		nowCalls++
		if nowCalls == 1 {
			return start
		}
		return start.Add(91 * time.Second)
	}

	err := EnsureNamespaceReady("timeout")
	require.Error(t, err)
	require.Empty(t, *added)
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

	workerManagerWaitForWorkflowResult = func(ctx context.Context, c client.Client, workflowID, runID string) (workflowengine.WorkflowResult, error) {
		_, hasDeadline := ctx.Deadline()
		require.True(t, hasDeadline)
		require.NotNil(t, c)
		require.Equal(t, "wf-1", workflowID)
		require.Equal(t, "run-1", runID)
		return workflowengine.WorkflowResult{}, nil
	}

	err := executeWorkerManagerWorkflow(
		"org-1",
		"org-0",
		[]string{" org/runner-1 ", "", "org/runner-1", "org/runner-2"},
	)
	require.NoError(t, err)
	require.Equal(t, "default", gotNamespace)
	require.NotNil(t, gotInput.ActivityOptions)

	payload, ok := gotInput.Payload.(workflows.WorkerManagerWorkflowPayload)
	require.True(t, ok)
	require.Equal(t, "org-1", payload.Namespace)
	require.Equal(t, "org-0", payload.OldNamespace)
	require.Equal(t, []string{"org/runner-1", "org/runner-2"}, payload.RunnerIDs)
	require.Nil(t, gotInput.Config)
}

func TestExecuteWorkerManagerWorkflowStartError(t *testing.T) {
	origStart := workerManagerStartWorkflow

	t.Cleanup(func() {
		workerManagerStartWorkflow = origStart
	})

	workerManagerStartWorkflow = func(_ string, _ workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("boom")
	}

	err := executeWorkerManagerWorkflow("org-1", "", nil)
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

	err := executeWorkerManagerWorkflow("org-1", "", nil)
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
	workerManagerWaitForWorkflowResult = func(_ context.Context, _ client.Client, _ string, _ string) (workflowengine.WorkflowResult, error) {
		return workflowengine.WorkflowResult{}, errors.New("wait failed")
	}

	err := executeWorkerManagerWorkflow("org-1", "", nil)
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
		return fw.runCalled.Load()
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()

	require.Equal(t, []string{"wf-1"}, fw.registeredWorkflows)
	require.Equal(t, []string{"act-1"}, fw.registeredActivities)
}

// The semaphore worker registers its own activity list, not CredimiActivities:
// an activity the semaphore schedules but this list misses fails at runtime
// with ActivityNotRegisteredError.
func TestDefaultWorkersSemaphoreRegistersItsActivities(t *testing.T) {
	var names []string
	for _, cfg := range defaultWorkers(nil) {
		if cfg.TaskQueue != workflows.MobileDeviceSemaphoreTaskQueue {
			continue
		}
		for _, act := range cfg.Activities {
			names = append(names, act.Name())
		}
	}
	require.Subset(t, names, []string{
		activities.StartQueuedPipelineActivityName,
		activities.NotifyPipelineStartFailureActivityName,
	})
}

func TestStartPipelineWorkerRegistersRegistryEntries(t *testing.T) {
	origNewWorker := newWorkerFn
	origRegistry := registry.Registry

	t.Cleanup(func() {
		newWorkerFn = origNewWorker
		registry.Registry = origRegistry
	})

	fw := &fakeWorker{}
	newWorkerFn = func(_ client.Client, _ string, _ worker.Options) worker.Worker {
		return fw
	}

	registry.Registry = map[string]registry.TaskFactory{
		"extra-activity-task": {
			Kind:    registry.TaskActivity,
			NewFunc: func(core.App) any { return fakeActivity{name: "extra-act"} },
		},
		"activity-task": {
			Kind:    registry.TaskActivity,
			NewFunc: func(core.App) any { return fakeActivity{name: "activity-act"} },
		},
		"workflow-task": {
			Kind:    registry.TaskWorkflow,
			NewFunc: func(core.App) any { return fakeWorkflow{name: "workflow-wf"} },
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go startPipelineWorker(ctx, nil, nil, &wg)

	require.Eventually(t, func() bool {
		return fw.runCalled.Load()
	}, time.Second, 10*time.Millisecond)

	cancel()
	wg.Wait()

	pipelineWf := pipeline.NewPipelineWorkflow().Name()
	debugAct := pipeline.NewDebugActivity().Name()
	githubPRCommentAct := activities.NewUpdateGitHubPRCommentActivity().Name()

	require.Contains(t, fw.registeredWorkflows, pipelineWf)
	require.Contains(t, fw.registeredWorkflows, pipeline.NewLinkedStepsWorkflow().Name())
	require.Contains(t, fw.registeredWorkflows, "workflow-wf")
	require.Contains(t, fw.registeredWorkflows, authorsession.WorkflowName)

	require.Contains(t, fw.registeredActivities, debugAct)
	require.Contains(t, fw.registeredActivities, githubPRCommentAct)
	require.Contains(t, fw.registeredActivities, "activity-act")
	require.Contains(t, fw.registeredActivities, "extra-act")
}

func TestRunWorkerWithRetryRetriesOnTransientError(t *testing.T) {
	origSleepWithContext := sleepWithContextFn
	t.Cleanup(func() {
		sleepWithContextFn = origSleepWithContext
	})

	var sleepCalls atomic.Int32
	sleepWithContextFn = func(_ context.Context, _ time.Duration) bool {
		sleepCalls.Add(1)
		return true
	}

	var runCalls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		runWorkerWithRetry(ctx, "queue-a", func() worker.Worker {
			return &retryTestWorker{
				runFn: func(interruptCh <-chan interface{}) error {
					if runCalls.Add(1) == 1 {
						return context.DeadlineExceeded
					}
					<-interruptCh
					return nil
				},
			}
		})
		close(done)
	}()

	require.Eventually(t, func() bool {
		return runCalls.Load() >= 2
	}, time.Second, 10*time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for runWorkerWithRetry to stop")
	}

	require.Equal(t, int32(2), runCalls.Load())
	require.Equal(t, int32(1), sleepCalls.Load())
}

func TestRunWorkerWithRetryStopsOnNonRetryableError(t *testing.T) {
	origSleepWithContext := sleepWithContextFn
	t.Cleanup(func() {
		sleepWithContextFn = origSleepWithContext
	})

	var sleepCalls atomic.Int32
	sleepWithContextFn = func(_ context.Context, _ time.Duration) bool {
		sleepCalls.Add(1)
		return true
	}

	var runCalls atomic.Int32
	runWorkerWithRetry(context.Background(), "queue-a", func() worker.Worker {
		return &retryTestWorker{
			runFn: func(_ <-chan interface{}) error {
				runCalls.Add(1)
				return serviceerror.NewInvalidArgument("invalid")
			},
		}
	})

	require.Equal(t, int32(1), runCalls.Load())
	require.Equal(t, int32(0), sleepCalls.Load())
}

func TestRunWorkerWithRetryRetriesUntilContextCancelled(t *testing.T) {
	origSleepWithContext := sleepWithContextFn
	t.Cleanup(func() {
		sleepWithContextFn = origSleepWithContext
	})

	var backoffs []time.Duration
	sleepWithContextFn = func(ctx context.Context, d time.Duration) bool {
		backoffs = append(backoffs, d)
		return ctx.Err() == nil
	}

	const failuresBeforeCancel = 20
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var runCalls atomic.Int32
	runWorkerWithRetry(ctx, "queue-a", func() worker.Worker {
		return &retryTestWorker{
			runFn: func(_ <-chan interface{}) error {
				if runCalls.Add(1) == failuresBeforeCancel {
					cancel()
				}
				return context.DeadlineExceeded
			},
		}
	})

	require.Equal(t, int32(failuresBeforeCancel), runCalls.Load())
	require.Len(t, backoffs, failuresBeforeCancel-1)
	require.Equal(t, workerStartMaxBackoff, backoffs[len(backoffs)-1])
}

func TestStartWorkerManagerWorkflowInvokesExecute(t *testing.T) {
	origExec := executeWorkerManagerWorkflowFn

	t.Cleanup(func() {
		executeWorkerManagerWorkflowFn = origExec
	})

	called := make(chan struct{}, 1)
	executeWorkerManagerWorkflowFn = func(
		namespace,
		oldNamespace string,
		runnerIDs []string,
	) error {
		require.Equal(t, "org-1", namespace)
		require.Equal(t, "org-0", oldNamespace)
		require.Equal(t, []string{"org/runner-1"}, runnerIDs)
		called <- struct{}{}
		return nil
	}

	StartWorkerManagerWorkflow("org-1", "org-0", []string{"org/runner-1"})

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for worker manager workflow")
	}
}

func TestStartWorkerManagerWorkflowSkipsWhenTemporalWorkersDisabled(t *testing.T) {
	t.Setenv(TemporalWorkersDisabledEnv, "1")

	origExec := executeWorkerManagerWorkflowFn
	t.Cleanup(func() {
		executeWorkerManagerWorkflowFn = origExec
	})

	called := make(chan struct{}, 1)
	executeWorkerManagerWorkflowFn = func(_, _ string, _ []string) error {
		called <- struct{}{}
		return nil
	}

	StartWorkerManagerWorkflow("org-1", "", []string{"org/runner-1"})

	select {
	case <-called:
		t.Fatal("worker manager workflow should not be executed")
	case <-time.After(100 * time.Millisecond):
	}
}
