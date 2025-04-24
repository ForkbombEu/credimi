// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package hooks provides functionality to manage and run Temporal workers
// for executing workflows and activities in a distributed system. It includes
// functions to start workers, register workflows and activities, and handle
// workflow execution.
package hooks

import (
	"log"
	"reflect"
	"sync"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows"
	"github.com/forkbombeu/credimi/pkg/workflowengine/workflows/credentials_config"
)

// WorkersHook sets up a hook for the PocketBase application to start all workers
// when the server starts. It binds a function to the OnServe event, which logs
// a message indicating that workers are starting and then asynchronously starts
// all workers by calling StartAllWorkers in a separate goroutine.
//
// Parameters:
//   - app: The PocketBase application instance to which the hook is attached.
func WorkersHook(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		log.Println("Starting workers...")
		go startAllWorkers()
		return se.Next()
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

func startAllWorkers() {
	c, err := temporalclient.New()
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup

	workers := []workerConfig{
		{
			TaskQueue: workflows.OpenIDNetTaskQueue,
			Workflows: []workflowengine.Workflow{
				&workflows.OpenIDNetWorkflow{},
				&workflows.OpenIDNetLogsWorkflow{},
			},
			Activities: []workflowengine.ExecutableActivity{
				&activities.StepCIWorkflowActivity{},
				&activities.SendMailActivity{},
				&activities.HTTPActivity{},
			},
		},
		{
			TaskQueue: workflows.CredentialsTaskQueue,
			Workflows: []workflowengine.Workflow{
				&workflows.CredentialsIssuersWorkflow{},
			},
			Activities: []workflowengine.ExecutableActivity{
				&activities.CheckCredentialsIssuerActivity{},
				&activities.JSONActivity{
					StructRegistry: map[string]reflect.Type{
						"OpenidCredentialIssuerSchemaJson": reflect.TypeOf(
							credentials_config.OpenidCredentialIssuerSchemaJson{},
						),
					},
				},
				&activities.HTTPActivity{},
			},
		},
	}

	for _, config := range workers {
		wg.Add(1)
		go startWorker(c, config, &wg)
	}

	wg.Wait()
}
