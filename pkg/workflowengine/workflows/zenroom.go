// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerclient "github.com/docker/docker/client"
	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
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

type ZenroomWorkflowPayload struct {
	Contract string `json:"contract" yaml:"contract" validate:"required"`
	Keys     string `json:"keys" yaml:"keys"`
	Data     string `json:"data" yaml:"data"`
	Config   string `json:"config" yaml:"config"`
}

func (w *ZenroomWorkflow) Name() string {
	return "Run a Zenroom contract from the docker image"
}

func (w *ZenroomWorkflow) GetOptions() workflow.ActivityOptions {
	return DefaultActivityOptions
}

func (w *ZenroomWorkflow) Workflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())
	runMetadata := workflowengine.WorkflowErrorMetadata{
		WorkflowName: w.Name(),
		WorkflowID:   workflow.GetInfo(ctx).WorkflowExecution.ID,
		Namespace:    workflow.GetInfo(ctx).Namespace,
		TemporalUI: fmt.Sprintf(
			"%s/my/tests/runs/%s/%s",
			input.Config["app_url"],
			workflow.GetInfo(ctx).WorkflowExecution.ID,
			workflow.GetInfo(ctx).WorkflowExecution.RunID,
		),
	}

	payload, err := workflowengine.DecodePayload[ZenroomWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(err, runMetadata)
	}

	var sideEffectResult struct {
		TmpDir  string
		CmdArgs []string
	}

	err = workflow.SideEffect(ctx, func(_ workflow.Context) interface{} {
		tmpDirLocal, err := os.MkdirTemp("", "zenroom-workflow-")
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.MkdirFailed]
			return workflowengine.NewAppError(errCode, err.Error())
		}

		errCode := errorcodes.Codes[errorcodes.WriteFileFailed]
		contractPath := filepath.Join(tmpDirLocal, "contract.zen")
		if err := os.WriteFile(contractPath, []byte(payload.Contract), 0600); err != nil {
			return workflowengine.NewAppError(errCode, err.Error())
		}

		cmdArgsLocal := make([]string, 0)

		if payload.Keys != "" {
			keysPath := filepath.Join(tmpDirLocal, "keys.json")
			if err := os.WriteFile(keysPath, []byte(payload.Keys), 0600); err != nil {
				return workflowengine.NewAppError(errCode, err.Error())
			}
			cmdArgsLocal = append(cmdArgsLocal, "-k", "/tmp/keys.json")
		}

		if payload.Data != "" {
			dataPath := filepath.Join(tmpDirLocal, "data.json")
			if err := os.WriteFile(dataPath, []byte(payload.Data), 0600); err != nil {
				return workflowengine.NewAppError(errCode, err.Error())
			}
			cmdArgsLocal = append(cmdArgsLocal, "-a", "/tmp/data.json")
		}

		if payload.Config != "" {
			cmdArgsLocal = append(cmdArgsLocal, "-c", payload.Config)
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
	if err != nil {
		logger.Error("Failed to prepare files inside SideEffect", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}

	activityInput := workflowengine.ActivityInput{
		Payload: activities.DockerActivityPayload{
			Image:  "ghcr.io/dyne/zenroom:latest",
			Cmd:    sideEffectResult.CmdArgs,
			Mounts: []string{sideEffectResult.TmpDir + ":/tmp"},
		},
	}

	// Execute the activity

	var result workflowengine.ActivityResult
	activity := activities.NewDockerActivity()
	err = workflow.ExecuteActivity(ctx, activity.Name(), activityInput).Get(ctx, &result)
	if err != nil {
		logger.Error("Activity failed", "error", err)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(err, runMetadata)
	}
	cli, err := dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerClientCreationFailed]
		appErr := workflowengine.NewAppError(errCode, err.Error())
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	output, ok := result.Output.(map[string]any)
	errCode := errorcodes.Codes[errorcodes.UnexpectedDockerOutput]
	if !ok {
		if !ok {
			msg := fmt.Sprintf("unexpected output type: %T", result.Output)
			appErr := workflowengine.NewAppError(errCode, msg, result.Output)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				runMetadata,
			)
		}
	}
	cli.ContainerRemove(
		context.Background(),
		output["containerID"].(string),
		container.RemoveOptions{Force: true},
	)

	exitCode, ok := output["exitCode"].(float64)
	if !ok {
		appErr := workflowengine.NewAppError(errCode, "invalid exit code", output["exitCode"])
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	stderr, ok := output["stderr"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(errCode, "invalid stderr ", output["stderr"])
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	stdout, ok := output["stdout"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(errCode, "invalid stderr ", output["stout"])
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	if int(exitCode) != 0 {
		errCode := errorcodes.Codes[errorcodes.ZenroomExecutionFailed]
		appErr := workflowengine.NewAppError(
			errCode,
			strconv.Itoa(int(exitCode)),
			stderr,
			stdout,
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
	}
	// Parse stdout as JSON
	var parsedOutput map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsedOutput); err != nil {
		logger.Error("Failed to parse stdout JSON", "error", err)
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		appErr := workflowengine.NewAppError(errCode, err.Error(), stdout)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(appErr, runMetadata)
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
	namespace := DefaultNamespace
	if input.Config["namespace"] != nil {
		namespace = input.Config["namespace"].(string)
	}
	c, err := temporalclient.GetTemporalClientWithNamespace(
		namespace,
	)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("unable to create client: %w", err)
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                       "Zenroom-Workflow-" + uuid.NewString(),
		TaskQueue:                ZenroomTaskQueue,
		WorkflowExecutionTimeout: 24 * time.Hour,
	}
	if input.Config["Memo"] != nil {
		workflowOptions.Memo = input.Config["Memo"].(map[string]any)
	}

	// Start the workflow execution.
	_, err = c.ExecuteWorkflow(context.Background(), workflowOptions, w.Name(), input)
	if err != nil {
		return workflowengine.WorkflowResult{}, fmt.Errorf("failed to start workflow: %w", err)
	}

	return workflowengine.WorkflowResult{}, nil
}
