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

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/pipeline"
	"github.com/forkbombeu/credimi/pkg/workflowengine/registry"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
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
		namespaces, err := FetchNamespaces(app)
		if err != nil {
			log.Fatalf("Failed to fetch namespaces: %v", err)
		}

		log.Printf("[WorkersHook] Ensuring %d namespace(s) are ready...", len(namespaces))

		for _, ns := range namespaces {
			if err := ensureNamespaceReadyWithRetry(ns); err != nil {
				log.Fatalf("[WorkersHook] Failed to connect to namespace %q: %v", ns, err)
			}
			log.Printf("[WorkersHook] Starting workers for namespace %q", ns)
			go StartAllWorkersByNamespace(ns)
			StartWorkerManagerWorkflow(ns, "")
		}

		log.Printf("[WorkersHook] All namespaces ready, workers started")
		return se.Next()
	})
	app.OnTerminate().BindFunc(func(_ *core.TerminateEvent) error {
		temporalclient.ShutdownClients()
		return nil
	})
}

type workerConfig struct {
	TaskQueue  string
	Workflows  []workflowengine.Workflow
	Activities []workflowengine.ExecutableActivity
}

var OrgWorkers = []workerConfig{
	{
		TaskQueue: workflows.OpenIDNetTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.OpenIDNetWorkflow{},
			&workflows.OpenIDNetLogsWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewStepCIWorkflowActivity(),
			activities.NewSendMailActivity(),
			activities.NewHTTPActivity(),
		},
	},
	{
		TaskQueue: workflows.EWCTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.EWCWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewStepCIWorkflowActivity(),
			activities.NewSendMailActivity(),
			activities.NewHTTPActivity(),
		},
	},
	{
		TaskQueue: workflows.EudiwTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.EudiwWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewStepCIWorkflowActivity(),
			activities.NewSendMailActivity(),
			activities.NewHTTPActivity(),
		},
	},
	{
		TaskQueue: workflows.CredentialsTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.CredentialsIssuersWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
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
		},
	},
	{
		TaskQueue: workflows.WalletTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.WalletWorkflow{},
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
		TaskQueue: workflows.CustomCheckTaskQueque,
		Workflows: []workflowengine.Workflow{
			&workflows.CustomCheckWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewStepCIWorkflowActivity(),
			activities.NewHTTPActivity(),
		},
	},
	{
		TaskQueue: workflows.VLEIValidationTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.VLEIValidationWorkflow{},
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
			&workflows.VLEIValidationLocalWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewCESRParsingActivity(),
			activities.NewCESRValidateActivity(),
		},
	},
}

var DefaultWorkers = []workerConfig{
	{
		TaskQueue: workflows.CustomCheckTaskQueque,
		Workflows: []workflowengine.Workflow{
			&workflows.CustomCheckWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewStepCIWorkflowActivity(),
			activities.NewHTTPActivity(),
		},
	},
	{
		TaskQueue: workflows.WorkerManagerTaskQueue,
		Workflows: []workflowengine.Workflow{
			&workflows.WorkerManagerWorkflow{},
		},
		Activities: []workflowengine.ExecutableActivity{
			activities.NewHTTPActivity(),
		},
	},
}

func startWorker(ctx context.Context, c client.Client, config workerConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	w := worker.New(c, config.TaskQueue, worker.Options{})

	for _, wf := range config.Workflows {
		w.RegisterWorkflowWithOptions(wf.Workflow, workflow.RegisterOptions{Name: wf.Name()})
	}

	for _, act := range config.Activities {
		w.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
	}
	shutdownCh := make(chan interface{})
	go func() {
		<-ctx.Done()
		close(shutdownCh)
	}()
	if err := w.Run(shutdownCh); err != nil {
		log.Printf("Failed to start worker for %s: %v", config.TaskQueue, err)
	}
}

func startPipelineWorker(ctx context.Context, c client.Client, wg *sync.WaitGroup) {
	defer wg.Done()
	w := worker.New(c, pipeline.PipelineTaskQueue, worker.Options{})

	pipelineWf := &pipeline.PipelineWorkflow{}
	w.RegisterWorkflowWithOptions(
		pipelineWf.Workflow,
		workflow.RegisterOptions{Name: pipelineWf.Name()},
	)

	for key, step := range registry.Registry {
		if _, skip := registry.PipelineWorkerDenylist[key]; skip {
			continue
		}
		switch step.Kind {
		case registry.TaskActivity:
			act := step.NewFunc().(workflowengine.ExecutableActivity)
			w.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{Name: act.Name()})
		case registry.TaskWorkflow:
			wf := step.NewFunc().(workflowengine.Workflow)
			w.RegisterWorkflowWithOptions(wf.Workflow, workflow.RegisterOptions{Name: wf.Name()})
		}
	}

	shutdownCh := make(chan interface{})
	go func() {
		<-ctx.Done()
		close(shutdownCh)
	}()
	if err := w.Run(shutdownCh); err != nil {
		log.Printf("Failed to start worker for %s: %v", pipeline.PipelineTaskQueue, err)
	}
}

var workerCancels sync.Map

func StartAllWorkersByNamespace(namespace string) {
	ctx, cancel := context.WithCancel(context.Background())
	workerCancels.Store(namespace, cancel)

	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}

	var wg sync.WaitGroup

	var workers []workerConfig

	if namespace == "default" {
		workers = DefaultWorkers
	} else {
		workers = OrgWorkers
	}

	for _, config := range workers {
		wg.Add(1)
		go startWorker(ctx, c, config, &wg)
	}

	wg.Add(1)
	go startPipelineWorker(ctx, c, &wg)

	go func() {
		wg.Wait()
		<-ctx.Done()
		log.Printf("Workers for namespace %s stopped", namespace)
	}()
}

func StopAllWorkersByNamespace(namespace string) {
	if cancel, ok := workerCancels.Load(namespace); ok {
		cancel.(context.CancelFunc)()
		workerCancels.Delete(namespace)
		log.Printf("Stopped workers for namespace %s", namespace)
	}
}

func FetchNamespaces(app *pocketbase.PocketBase) ([]string, error) {
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

func ensureNamespaceReadyWithRetry(namespace string) error {
	hostPort := utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort)
	log.Printf("[WorkersHook] Connecting to Temporal at %s for namespace %q", hostPort, namespace)

	deadline := time.Now().Add(90 * time.Second)
	attempt := 0

	nc, err := client.NewNamespaceClient(client.Options{
		HostPort: hostPort,
		ConnectionOptions: client.ConnectionOptions{
			TLS: nil,
		},
	})
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
		elapsed := time.Since(start)

		if err == nil {
			log.Printf(
				"[WorkersHook] Namespace %q ready after %d attempt(s) in %v",
				namespace,
				attempt,
				elapsed,
			)
			return nil
		}

		var notFound *serviceerror.NamespaceNotFound
		if errors.As(err, &notFound) {
			err = nc.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
				Namespace: namespace,
			})
			if err != nil {
				log.Printf("[WorkersHook] Unable to create namespace %s: %v", namespace, err)
			}
			log.Printf("[WorkersHook] Created namespace %s", namespace)
		}

		log.Printf(
			"[WorkersHook] Attempt %d failed in %v: namespace=%s err=%v",
			attempt,
			elapsed,
			namespace,
			err,
		)

		if time.Now().After(deadline) {
			return err
		}

		backoff := time.Duration(attempt) * time.Second
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
		log.Printf("[WorkersHook] Sleeping %v before retry...", backoff)
		time.Sleep(backoff)
	}
}

func StartWorkerManagerWorkflow(namespace, oldNamespace string) {
	go func() {
		if err := executeWorkerManagerWorkflow(namespace, oldNamespace); err != nil {
			log.Printf("[WorkerManagerWorkflow] Failed for namespace %s: %v", namespace, err)
		} else {
			log.Printf("[WorkerManagerWorkflow] Successfully started for namespace %s", namespace)
		}
	}()
}

func executeWorkerManagerWorkflow(namespace, oldNamespace string) error {
	serverURL := utils.GetEnvironmentVariable("MAESTRO_WORKER", "http://localhost:8050")

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
		Payload: map[string]any{
			"namespace":     namespace,
			"old_namespace": oldNamespace,
		},
		Config: map[string]any{
			"server_url": serverURL,
		},
		ActivityOptions: ao,
	}

	var w workflows.WorkerManagerWorkflow
	resStart, err := w.Start("default", input)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	c, err := temporalclient.GetTemporalClientWithNamespace("default")
	if err != nil {
		return fmt.Errorf("unable to create client: %w", err)
	}

	_, err = workflowengine.WaitForWorkflowResult(
		c,
		resStart.WorkflowID,
		resStart.WorkflowRunID,
	)
	if err != nil {
		return fmt.Errorf("failed to start mobile automation worker for organization: %w", err)
	}

	return nil
}
