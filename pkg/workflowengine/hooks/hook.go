// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package hooks provides functionality to manage and run Temporal workers
// for executing workflows and activities in a distributed system. It includes
// functions to start workers, register workflows and activities, and handle
// workflow execution.
package hooks

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apis/handlers"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/protobuf/types/known/durationpb"
)

// WorkersHook sets up a hook for the PocketBase application to
// create the namespaces for already existing orgs and starts the workers
// when the server starts. It binds a function to the OnServe event, which logs
// a message indicating that workers are starting and then asynchronously starts
// all workers by calling StartAllWorkers in a separate goroutine.
//
// Parameters:
//   - app: The PocketBase application instance to which the hook is attached.
func WorkersHook(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if TemporalWorkersDisabled() {
			log.Printf(
				"[WorkersHook] Skipping namespaces and workers (%s is set)",
				TemporalWorkersDisabledEnv,
			)
			return se.Next()
		}

		namespaces, err := fetchNamespacesFn(app)
		if err != nil {
			log.Fatalf("Failed to fetch namespaces: %v", err)
		}
		orgRecords, err := workerManagerOrgRecordsFn(app)
		if err != nil {
			log.Fatalf("Failed to fetch organization records: %v", err)
		}
		adminRunnerIDs, err := adminRunnerIDsFn(app)
		if err != nil {
			log.Fatalf("Failed to fetch admin-managed runner IDs: %v", err)
		}
		publishedRunnerIDs, err := publishedRunnerIDsFn(app)
		if err != nil {
			log.Fatalf("Failed to fetch published runner IDs: %v", err)
		}
		publishedByNamespace := make(map[string]bool, len(orgRecords))
		for _, org := range orgRecords {
			publishedByNamespace[org.GetString("canonified_name")] = org.GetBool("published")
		}

		log.Printf("[WorkersHook] Ensuring %d namespace(s) are ready...", len(namespaces))

		for _, ns := range namespaces {
			if err := ensureNamespaceReadyFn(ns); err != nil {
				log.Fatalf("[WorkersHook] Failed to connect to namespace %q: %v", ns, err)
			}
			log.Printf("[WorkersHook] Starting workers for namespace %q", ns)
			go startAllWorkersByNamespace(se.App, ns)

			runnerIDs := adminRunnerIDs
			if ns == "default" || publishedByNamespace[ns] {
				runnerIDs = combineWorkerManagerRunnerIDs(adminRunnerIDs, publishedRunnerIDs)
			}
			startWorkerManagerWorkflow(ns, "", runnerIDs)
		}

		log.Printf("[WorkersHook] All namespaces ready, workers started")
		return se.Next()
	})
	app.OnTerminate().BindFunc(func(te *core.TerminateEvent) error {
		StopAllWorkers()
		shutdownTemporalClientsFn()
		return te.Next()
	})
}

type workerConfig struct {
	TaskQueue  string
	Workflows  []workflowengine.Workflow
	Activities []workflowengine.ExecutableActivity
}

func orgWorkers(app core.App) []workerConfig {
	return []workerConfig{
		{
			TaskQueue: workflows.OpenID4VPWalletTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewOpenID4VPWalletWorkflow(),
				workflows.NewOpenID4VPWalletLogsWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
				activities.NewSendMailActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.OpenID4VCIIssuerTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewOpenID4VCIIssuerWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.OpenID4VPVerifierTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewOpenID4VPVerifierWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.EWCTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewEWCWorkflow(),
				workflows.NewWebuildWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
				activities.NewSendMailActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.EudiwTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewEudiwWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
				activities.NewSendMailActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.CredentialsTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewCredentialsIssuersWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewCheckCredentialsIssuerActivity(),
				activities.NewJSONActivity(
					map[string]reflect.Type{
						"map": reflect.TypeOf(
							map[string]any{},
						),
					},
				),
				activities.NewSchemaValidationActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.WalletTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewWalletWorkflow(),
			},
			Activities: []workflowengine.ExecutableActivity{
				activities.NewParseWalletURLActivity(),
				activities.NewDockerActivity(),
				activities.NewJSONActivity(
					map[string]reflect.Type{
						"map": reflect.TypeOf(
							map[string]any{},
						),
					},
				),
				activities.NewHTTPActivity(),
			},
		},
		{
			TaskQueue: workflows.CustomCheckTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewCustomCheckWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.VLEIValidationTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewVLEIValidationWorkflow(),
			},
			Activities: []workflowengine.ExecutableActivity{
				activities.NewHTTPActivity(),
				activities.NewCESRParsingActivity(),
				activities.NewCESRValidateActivity(),
			},
		},
		{
			TaskQueue: workflows.VLEIValidationLocalTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewVLEIValidationLocalWorkflow(),
			},
			Activities: []workflowengine.ExecutableActivity{
				activities.NewCESRParsingActivity(),
				activities.NewCESRValidateActivity(),
			},
		},
		{
			TaskQueue: workflows.FidesCredentialIssuersTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewFidesCredentialIssuersWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewHTTPActivity(),
				activities.NewParseFidesCredentialIssuersActivity(),
				activities.NewCheckCredentialsIssuerActivity(),
				activities.NewJSONActivity(
					map[string]reflect.Type{
						"map": reflect.TypeOf(
							map[string]any{},
						),
					},
				),
				activities.NewSchemaValidationActivity(),
			}, activities.CredimiActivities(app)...),
		},
	}
}

func defaultWorkers(app core.App) []workerConfig {
	return []workerConfig{
		{
			TaskQueue: workflows.CustomCheckTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewCustomCheckWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.ConformanceCheckTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewStartCheckWorkflow(),
				workflows.NewEWCStatusWorkflow(),
				workflows.NewWebuildStatusWorkflow(),
			},
			Activities: append([]workflowengine.ExecutableActivity{
				activities.NewStepCIWorkflowActivity(),
				activities.NewHTTPActivity(),
			}, activities.CredimiActivities(app)...),
		},
		{
			TaskQueue: workflows.WorkerManagerTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewWorkerManagerWorkflow(),
			},
			Activities: []workflowengine.ExecutableActivity{
				activities.NewHTTPActivity(),
				activities.NewMobileRunnerHTTPActivity(app),
			},
		},
		{
			TaskQueue: workflows.MobileDeviceSemaphoreTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewMobileDeviceSemaphoreWorkflow(),
				workflows.NewGitHubPRCommentWorkflow(),
			},
			Activities: []workflowengine.ExecutableActivity{
				activities.NewStartQueuedPipelineActivity(app),
				activities.NewCheckWorkflowClosedActivity(),
				activities.NewSignalWorkflowActivity(),
				activities.NewCancelWorkflowActivity(),
				activities.NewCleanupMobileDeviceSemaphoreResourcesActivity(app),
				activities.NewQueryMobileDeviceSemaphoreRunStatusActivity(),
				activities.NewUpdateGitHubPRCommentActivity(),
				activities.NewPatchGitHubPRCommentActivity(),
			},
		},
		{
			TaskQueue: workflows.AggregateScoreboardTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewAggregateScoreboardWorkflow(),
			},
			Activities: []workflowengine.ExecutableActivity{
				handlers.NewListScoreboardNamespacesActivity(app),
				handlers.NewGetNamespaceScoreboardActivity(app),
				handlers.NewGetScoreboardExecutionDetailsActivity(),
				handlers.NewSaveScoreboardResultsActivity(app),
			},
		},
		{
			TaskQueue: workflows.PipelineRetentionTaskQueue,
			Workflows: []workflowengine.Workflow{
				workflows.NewPipelineRetentionWorkflow(),
			},
			Activities: activities.CredimiActivities(app),
		},
	}
}

var (
	getTemporalClient          = temporalclient.GetTemporalClientWithNamespace
	newNamespaceClientFn       = temporalclient.NewNamespaceClient
	newWorkerFn                = worker.New
	sleepFn                    = time.Sleep
	sleepWithContextFn         = sleepWithContext
	nowFn                      = time.Now
	startWorkerFn              = startWorker
	startPipelineWorkerFn      = startPipelineWorker
	fetchNamespacesFn          = FetchNamespaces
	workerManagerOrgRecordsFn  = workerManagerAllOrganizationRecords
	adminRunnerIDsFn           = WorkerManagerAdminRunnerIDs
	publishedRunnerIDsFn       = WorkerManagerPublishedNonAdminRunnerIDs
	ensureNamespaceReadyFn     = EnsureNamespaceReady
	addSearchAttributesFn      = addSearchAttributes
	startAllWorkersByNamespace = StartAllWorkersByNamespace
	startWorkerManagerWorkflow = StartWorkerManagerWorkflow
	shutdownTemporalClientsFn  = temporalclient.ShutdownClients
	workerManagerStartWorkflow = func(namespace string, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
		w := workflows.NewWorkerManagerWorkflow()
		return w.Start(namespace, input)
	}
	workerManagerTemporalClient        = temporalclient.GetTemporalClientWithNamespace
	workerManagerWaitForWorkflowResult = workflowengine.WaitForWorkflowResult
	executeWorkerManagerWorkflowFn     = executeWorkerManagerWorkflow
)

const (
	workerStartInitialBackoff = time.Second
	workerStartMaxBackoff     = 30 * time.Second
	// workerStopTimeout lets in-flight activities finish when a worker stops.
	workerStopTimeout = 20 * time.Second
	// workerStopWait bounds how long a stop waits for a namespace's workers.
	workerStopWait = 30 * time.Second
	// workerManagerWaitTimeout bounds how long a worker-manager run is awaited.
	workerManagerWaitTimeout = 5 * time.Minute
)

func startWorker(ctx context.Context, c client.Client, config workerConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	runWorkerWithRetry(ctx, config.TaskQueue, func() worker.Worker {
		w := newWorkerFn(c, config.TaskQueue, worker.Options{WorkerStopTimeout: workerStopTimeout})

		for _, wf := range config.Workflows {
			w.RegisterWorkflowWithOptions(wf.Workflow, workflow.RegisterOptions{Name: wf.Name()})
		}

		for _, act := range config.Activities {
			w.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
		}

		return w
	})
}

func startPipelineWorker(ctx context.Context, app core.App, c client.Client, wg *sync.WaitGroup) {
	defer wg.Done()
	runWorkerWithRetry(ctx, pipeline.PipelineTaskQueue, func() worker.Worker {
		w := newWorkerFn(
			c,
			pipeline.PipelineTaskQueue,
			worker.Options{WorkerStopTimeout: workerStopTimeout},
		)

		pipelineWf := pipeline.NewPipelineWorkflow()
		w.RegisterWorkflowWithOptions(
			pipelineWf.Workflow,
			workflow.RegisterOptions{Name: pipelineWf.Name()},
		)
		debugAct := pipeline.NewDebugActivity()
		w.RegisterActivityWithOptions(
			debugAct.Execute,
			activity.RegisterOptions{Name: debugAct.Name()},
		)
		githubPRCommentAct := activities.NewUpdateGitHubPRCommentActivity()
		w.RegisterActivityWithOptions(
			githubPRCommentAct.Execute,
			activity.RegisterOptions{Name: githubPRCommentAct.Name()},
		)

		for _, step := range registry.Registry {
			switch step.Kind {
			case registry.TaskActivity:
				act := step.NewFunc(app).(workflowengine.ExecutableActivity)
				w.RegisterActivityWithOptions(
					act.Execute,
					activity.RegisterOptions{Name: act.Name()},
				)
			case registry.TaskWorkflow:
				wf := step.NewFunc(app).(workflowengine.Workflow)
				w.RegisterWorkflowWithOptions(
					wf.Workflow,
					workflow.RegisterOptions{Name: wf.Name()},
				)
			}
		}

		for _, step := range registry.PipelineInternalRegistry {
			switch step.Kind {
			case registry.TaskActivity:
				act := step.NewFunc(app).(workflowengine.ExecutableActivity)
				w.RegisterActivityWithOptions(
					act.Execute,
					activity.RegisterOptions{Name: act.Name()},
				)
			case registry.TaskWorkflow:
				wf := step.NewFunc(app).(workflowengine.Workflow)
				w.RegisterWorkflowWithOptions(
					wf.Workflow,
					workflow.RegisterOptions{Name: wf.Name()},
				)
			}
		}

		for _, act := range activities.CredimiActivities(app) {
			w.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
		}

		return w
	})
}

func runWorkerWithRetry(ctx context.Context, taskQueue string, build func() worker.Worker) {
	backoff := workerStartInitialBackoff

	for {
		if ctx.Err() != nil {
			return
		}

		w := build()
		attemptCtx, cancelAttempt := context.WithCancel(ctx)
		shutdownCh := make(chan interface{})
		go func() {
			<-attemptCtx.Done()
			close(shutdownCh)
		}()

		err := w.Run(shutdownCh)
		cancelAttempt()

		if err == nil || ctx.Err() != nil {
			return
		}
		if !shouldRetryWorkerStartError(err) {
			log.Printf("Worker for %s stopped with non-retryable error: %v", taskQueue, err)
			return
		}

		log.Printf(
			"Worker for %s stopped with retryable error: %v (retrying in %s)",
			taskQueue,
			err,
			backoff,
		)
		if !sleepWithContextFn(ctx, backoff) {
			return
		}
		backoff = growBackoff(backoff, workerStartMaxBackoff)
	}
}

func shouldRetryWorkerStartError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}

	var namespaceNotFound *serviceerror.NamespaceNotFound
	var invalidArgument *serviceerror.InvalidArgument
	var permissionDenied *serviceerror.PermissionDenied
	var unimplemented *serviceerror.Unimplemented
	if errors.As(err, &namespaceNotFound) ||
		errors.As(err, &invalidArgument) ||
		errors.As(err, &permissionDenied) ||
		errors.As(err, &unimplemented) {
		return false
	}

	return true
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func growBackoff(current, maxDuration time.Duration) time.Duration {
	next := current * 2
	if next > maxDuration {
		return maxDuration
	}
	return next
}

// namespaceWorkers is the running worker set of one namespace: cancel stops
// it and done closes once every worker of the set has returned.
type namespaceWorkers struct {
	cancel context.CancelFunc
	done   chan struct{}
}

var workerSets sync.Map // map[string]*namespaceWorkers

func StartAllWorkersByNamespace(app core.App, namespace string) {
	if TemporalWorkersDisabled() {
		log.Printf(
			"Skipping workers for namespace %s (%s is set)",
			namespace,
			TemporalWorkersDisabledEnv,
		)
		return
	}

	if _, ok := workerSets.Load(namespace); ok {
		StopAllWorkersByNamespace(namespace)
	}

	ctx, cancel := context.WithCancel(context.Background())

	c, err := getTemporalClient(namespace)
	if err != nil {
		log.Printf("Failed to connect to Temporal for namespace %s: %v", namespace, err)
		cancel()
		return
	}

	set := &namespaceWorkers{cancel: cancel, done: make(chan struct{})}
	workerSets.Store(namespace, set)

	var wg sync.WaitGroup

	var workers []workerConfig

	if namespace == "default" {
		workers = defaultWorkers(app)
	} else {
		workers = orgWorkers(app)
	}

	for _, config := range workers {
		wg.Add(1)
		go startWorkerFn(ctx, c, config, &wg)
	}

	wg.Add(1)
	go startPipelineWorkerFn(ctx, app, c, &wg)

	go func() {
		wg.Wait()
		close(set.done)
		log.Printf("Workers for namespace %s stopped", namespace)
	}()
}

// StopAllWorkersByNamespace cancels the namespace's workers and waits up to
// workerStopWait for them to drain.
func StopAllWorkersByNamespace(namespace string) {
	value, ok := workerSets.LoadAndDelete(namespace)
	if !ok {
		return
	}
	set := value.(*namespaceWorkers)
	set.cancel()

	timer := time.NewTimer(workerStopWait)
	defer timer.Stop()
	select {
	case <-set.done:
		log.Printf("Stopped workers for namespace %s", namespace)
	case <-timer.C:
		log.Printf(
			"Timed out after %s waiting for workers of namespace %s to stop",
			workerStopWait,
			namespace,
		)
	}
}

// StopAllWorkers stops the workers of every namespace concurrently and waits
// for them to drain.
func StopAllWorkers() {
	var wg sync.WaitGroup
	workerSets.Range(func(key, _ any) bool {
		namespace := key.(string)
		wg.Add(1)
		go func() {
			defer wg.Done()
			StopAllWorkersByNamespace(namespace)
		}()
		return true
	})
	wg.Wait()
}

func FetchNamespaces(app core.App) ([]string, error) {
	collection, err := app.FindCollectionByNameOrId("organizations")
	if err != nil {
		return nil, err
	}

	records, err := app.FindRecordsByFilter(collection, "", "-created", 0, 0)
	if err != nil {
		return nil, err
	}

	namespaces := make([]string, 0, len(records)+1)
	namespaces = append(namespaces, "default")
	for _, r := range records {
		namespaces = append(namespaces, r.GetString("canonified_name"))
	}
	return namespaces, nil
}

const namespaceRetention = 365 * 24 * time.Hour

// EnsureNamespaceReady registers the namespace when missing, waits until Temporal
// describes it and registers Credimi's custom search attributes on it. It retries
// with backoff for up to 90 seconds.
func EnsureNamespaceReady(namespace string) error {
	log.Printf("[WorkersHook] Ensuring Temporal namespace %q", namespace)

	deadline := nowFn().Add(90 * time.Second)
	attempt := 0

	nc, err := newNamespaceClientFn()
	if err != nil {
		return err
	}
	defer nc.Close()

	for {
		attempt++
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err = nc.Describe(ctx, namespace)
		cancel()

		if err == nil {
			saCtx, saCancel := context.WithTimeout(context.Background(), 10*time.Second)
			err = addSearchAttributesFn(saCtx, namespace)
			saCancel()
			if err == nil {
				log.Printf(
					"[WorkersHook] Namespace %q ready after %d attempt(s) in %v",
					namespace,
					attempt,
					time.Since(start),
				)
				return nil
			}
			err = fmt.Errorf("add search attributes: %w", err)
		} else {
			var notFound *serviceerror.NamespaceNotFound
			if errors.As(err, &notFound) {
				regCtx, regCancel := context.WithTimeout(context.Background(), 10*time.Second)
				err = nc.Register(regCtx, &workflowservice.RegisterNamespaceRequest{
					Namespace:                        namespace,
					WorkflowExecutionRetentionPeriod: durationpb.New(namespaceRetention),
				})
				regCancel()
				if err == nil {
					log.Printf("[WorkersHook] Created namespace %s", namespace)
					continue
				}
				log.Printf("[WorkersHook] Unable to create namespace %s: %v", namespace, err)
			}
		}

		log.Printf(
			"[WorkersHook] Attempt %d failed in %v: namespace=%s err=%v",
			attempt,
			time.Since(start),
			namespace,
			err,
		)

		if nowFn().After(deadline) {
			return err
		}

		backoff := time.Duration(attempt) * time.Second
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
		log.Printf("[WorkersHook] Sleeping %v before retry...", backoff)
		sleepFn(backoff)
	}
}

func addSearchAttributes(ctx context.Context, namespace string) error {
	c, err := getTemporalClient(namespace)
	if err != nil {
		return err
	}
	_, err = c.OperatorService().
		AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
			Namespace:        namespace,
			SearchAttributes: workflowengine.CustomSearchAttributeTypes(),
		})
	return err
}

func StartWorkerManagerWorkflow(namespace, oldNamespace string, runnerIDs []string) {
	if TemporalWorkersDisabled() {
		log.Printf(
			"[WorkerManagerWorkflow] Skipping for namespace %s (%s is set)",
			namespace,
			TemporalWorkersDisabledEnv,
		)
		return
	}

	go func() {
		if err := executeWorkerManagerWorkflowFn(
			namespace,
			oldNamespace,
			runnerIDs,
		); err != nil {
			log.Printf("[WorkerManagerWorkflow] Failed for namespace %s: %v", namespace, err)
		} else {
			log.Printf("[WorkerManagerWorkflow] Successfully started for namespace %s", namespace)
		}
	}()
}

func executeWorkerManagerWorkflow(
	namespace,
	oldNamespace string,
	runnerIDs []string,
) error {
	ao := &workflow.ActivityOptions{
		ScheduleToCloseTimeout: time.Minute,
		StartToCloseTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 1.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    5,
		},
	}

	input := workflowengine.WorkflowInput{
		Payload: workflows.WorkerManagerWorkflowPayload{
			Namespace:    namespace,
			OldNamespace: oldNamespace,
			RunnerIDs:    uniqueWorkerManagerRunnerIDs(runnerIDs),
		},
		ActivityOptions: ao,
	}

	resStart, err := workerManagerStartWorkflow("default", input)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	c, err := workerManagerTemporalClient("default")
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	waitCtx, cancel := context.WithTimeout(context.Background(), workerManagerWaitTimeout)
	defer cancel()
	_, err = workerManagerWaitForWorkflowResult(
		waitCtx,
		c,
		resStart.WorkflowID,
		resStart.WorkflowRunID,
	)
	if err != nil {
		return fmt.Errorf("failed to start mobile automation worker for organization: %w", err)
	}

	return nil
}
