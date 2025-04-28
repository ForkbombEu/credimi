// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	temporalclient "github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	workflowengine "github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/forkbombeu/credimi/pkg/workflowengine/activities"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

const ZenroomTaskQueue = "ZenroomTaskQueue"

type ZenroomWorkflow struct{}

func (w *ZenroomWorkflow) Name() string {
	return "Run a Zenroom contract from the docker image"
}

func (w *ZenroomWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *ZenroomWorkflow) Workflow(ctx workflow.Context, input workflowengine.WorkflowInput) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	var sideEffectResult struct {
		TmpDir  string
		CmdArgs []string
	}

	err := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		tmpDirLocal, err := os.MkdirTemp("", "zenroom-workflow-")
		if err != nil {
			return err
		}

		contract, _ := input.Payload["contract"].(string)
		contractPath := filepath.Join(tmpDirLocal, "contract.zen")
		if err := os.WriteFile(contractPath, []byte(contract), 0644); err != nil {
			return err
		}

		cmdArgsLocal := make([]string, 0)

		if keys, ok := input.Payload["keys"].(string); ok {
			keysPath := filepath.Join(tmpDirLocal, "keys.json")
			if err := os.WriteFile(keysPath, []byte(keys), 0644); err != nil {
				return err
			}
			cmdArgsLocal = append(cmdArgsLocal, "-k", "/tmp/keys.json")
		}

		if data, ok := input.Payload["data"].(string); ok {
			dataPath := filepath.Join(tmpDirLocal, "data.json")
			if err := os.WriteFile(dataPath, []byte(data), 0644); err != nil {
				return err
			}
			cmdArgsLocal = append(cmdArgsLocal, "-a", "/tmp/data.json")
		}

		if config, ok := input.Payload["config"].(string); ok {
			cmdArgsLocal = append(cmdArgsLocal, "-c", config)
		}

		cmdArgsLocal = append(cmdArgsLocal, "-z", "/tmp/contract.zen")

		return struct {
			TmpDir  string
			CmdArgs []string
		}{
			TmpDir:  tmpDirLocal,
			CmdArgs: cmdArgsLocal,
		}
	}).Get(&sideEffectResult)
	fmt.Println("tmpDir", sideEffectResult.TmpDir)
	if err != nil {
		logger.Error("Failed to prepare files inside SideEffect", "error", err)
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to prepare temp files: %w", err)
	}

	activityInput := workflowengine.ActivityInput{
		Payload: map[string]any{
			"image":  "ghcr.io/dyne/zenroom:latest",
			"cmd":    sideEffectResult.CmdArgs,
			"mounts": []string{sideEffectResult.TmpDir + ":/tmp"},
		},
	}

	// Execute the activity

	var result workflowengine.ActivityResult
	var activity activities.DockerActivity
	err = workflow.ExecuteActivity(ctx, activity.Name(), activityInput).Get(ctx, &result)
	if err != nil {
		logger.Error("Activity failed", "error", err)
		return workflowengine.WorkflowResult{}, err

	}
	cli, err := dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to create Docker client: %v", err)
	}
	output, ok := result.Output.(map[string]any)
	if !ok {
		return workflowengine.WorkflowResult{}, errors.New("invalid output format")
	}
	cli.ContainerRemove(context.Background(), output["containerID"].(string), container.RemoveOptions{Force: true})

	exitCode, ok := output["exitCode"].(float64)
	if !ok {
		return workflowengine.WorkflowResult{}, errors.New("invalid exit code format")
	}
	if int(exitCode) != 0 {
		return workflowengine.WorkflowResult{}, fmt.Errorf("Zenroom execution failed with exit code %d", int(exitCode))
	}
	// Parse stdout as JSON
	outputStr, ok := output["stdout"].(string)
	if !ok {
		return workflowengine.WorkflowResult{}, errors.New("invalid output format")
	}
	var parsedOutput map[string]any

	if err := json.Unmarshal([]byte(outputStr), &parsedOutput); err != nil {
		logger.Error("Failed to parse stdout JSON", "error", err)
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to parse stdout JSON: %w", err)
	}

	return workflowengine.WorkflowResult{
		Message: "Zenroom execution successful",
		Output:  parsedOutput,
		Log:     result.Log,
	}, nil
}

func (w *ZenroomWorkflow) Start(
	input workflowengine.WorkflowInput,
) (result workflowengine.WorkflowResult, err error) {
	// Load environment variables.
	godotenv.Load()
	namespace := "default"
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unable to create client: %v", err)
	}
	defer c.Close()

	workflowOptions := client.StartWorkflowOptions{
		ID:        "Zenroom-Workflow-" + uuid.NewString(),
		TaskQueue: ZenroomTaskQueue,
	}
	if input.Config["Memo"] != nil {
		workflowOptions.Memo = input.Config["Memo"].(map[string]any)
	}

	// Start the workflow execution.
	_, err = c.ExecuteWorkflow(context.Background(), workflowOptions, w.Name(), input)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to start workflow: %v", err)
	}

	return workflowengine.WorkflowResult{}, nil
}
