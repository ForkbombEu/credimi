// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package worker_engine

import (
	"log"
	"reflect"
	"sync"

	temporalclient "github.com/forkbombeu/didimo/pkg/internal/temporal_client"
	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/activities"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/workflows"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/workflows/credentials_config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// WorkerConfig defines a worker setup
type WorkerConfig struct {
	TaskQueue  string
	Workflows  []workflowengine.Workflow
	Activities []workflowengine.ExecutableActivity
}

// StartWorker initializes and runs a single Temporal worker
func StartWorker(client client.Client, config WorkerConfig, wg *sync.WaitGroup) {
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
		log.Fatalf("Failed to start worker for %s: %v", config.TaskQueue, err)
	}
}

// StartAllWorkers initializes and starts multiple Temporal workers
func StartAllWorkers() {
	c, err := temporalclient.GetTemporalClient()
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup

	workers := []WorkerConfig{
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
						"OpenidCredentialIssuerSchemaJson": reflect.TypeOf(credentials_config.OpenidCredentialIssuerSchemaJson{}),
					},
				},
				&activities.HTTPActivity{},
			},
		},
	}

	for _, config := range workers {
		wg.Add(1)
		go StartWorker(c, config, &wg)
	}

	wg.Wait()
}

func WorkersHook(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		log.Println("Starting workers...")
		go StartAllWorkers()
		return se.Next()
	})

}
