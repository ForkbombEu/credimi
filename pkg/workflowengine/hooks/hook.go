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
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
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
		c, err := client.NewNamespaceClient(client.Options{})
		if err != nil {
			log.Fatalln("Unable to create client", err)
		}
		defer c.Close()

		namespaces, err := FetchNamespaces(app)
		if err != nil {
			log.Fatalf("Failed to fetch namespaces: %v", err)
		}
		for _, ns := range namespaces {
			_, err = c.Describe(context.Background(), ns)
			if err == nil {
				go StartAllWorkersByNamespace(ns)
				continue
			}
			var notFound *serviceerror.NamespaceNotFound
			if !errors.As(err, &notFound) {
				log.Fatalln("unexpected error while describing namespace", err)
			}

			err = c.Register(context.Background(), &workflowservice.RegisterNamespaceRequest{
				Namespace:                        ns,
				WorkflowExecutionRetentionPeriod: durationpb.New(7 * 24 * time.Hour),
			})
			if err != nil {
				log.Printf("Unable to create namespace %s: %v", ns, err)
			}
			go StartAllWorkersByNamespace(ns)
		}
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

func startWorker(client client.Client, config workerConfig, wg *sync.WaitGroup) {
	defer wg.Done()
	w := worker.New(client, config.TaskQueue, worker.Options{})

	for _, wf := range config.Workflows {
		w.RegisterWorkflowWithOptions(wf.Workflow, workflow.RegisterOptions{
			Name: wf.Name(),
		})
	}

	for _, act := range config.Activities {
		w.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
			Name: act.Name(),
		})
	}

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Printf("Failed to start worker for %s: %v", config.TaskQueue, err)
	}
}
func startPipelineWorker(client client.Client, wg *sync.WaitGroup) {
	defer wg.Done()
	w := worker.New(client, pipeline.PipelineTaskQueue, worker.Options{})
	pipelineWf := &pipeline.PipelineWorkflow{}
	w.RegisterWorkflowWithOptions(pipelineWf.Workflow, workflow.RegisterOptions{
		Name: pipelineWf.Name(),
	})
	for _, step := range registry.Registry {
		switch step.Kind {
		case registry.TaskActivity:

			act := step.NewFunc().(workflowengine.ExecutableActivity)
			w.RegisterActivityWithOptions(act.Execute, activity.RegisterOptions{
				Name: act.Name(),
			})
		case registry.TaskWorkflow:
			wf := step.NewFunc().(workflowengine.Workflow)
			w.RegisterWorkflowWithOptions(wf.Workflow, workflow.RegisterOptions{
				Name: wf.Name(),
			})
		}

	}
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Printf("Failed to start worker for %s: %v", pipeline.PipelineTaskQueue, err)
	}
}
func StartAllWorkersByNamespace(namespace string) {
	c, err := temporalclient.GetTemporalClientWithNamespace(namespace)
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}

	var wg sync.WaitGroup

	workers := []workerConfig{
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

	for _, config := range workers {
		wg.Add(1)
		go startWorker(c, config, &wg)
	}

	wg.Add(1)
	go startPipelineWorker(c, &wg)

	wg.Wait()
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

	namespaces := make([]string, 0, len(records))
	for _, r := range records {
		namespaces = append(namespaces, r.Id)
	}
	return namespaces, nil
}
