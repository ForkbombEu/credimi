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

type dockerClient interface {
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
}

var newDockerClient = func() (dockerClient, error) {
	return dockerclient.NewClientWithOpts(
		dockerclient.FromEnv,
		dockerclient.WithAPIVersionNegotiation(),
	)
}

type ZenroomWorkflow struct {
	WorkflowFunc workflowengine.WorkflowFn
}

var zenroomTemporalClient = temporalclient.GetTemporalClientWithNamespace

type ZenroomWorkflowPayload struct {
	Contract string `json:"contract" yaml:"contract" validate:"required"`
	Keys     string `json:"keys"     yaml:"keys"`
	Data     string `json:"data"     yaml:"data"`
	Config   string `json:"config"   yaml:"config"`
}

func NewZenroomWorkflow() *ZenroomWorkflow {
	w := &ZenroomWorkflow{}
	w.WorkflowFunc = workflowengine.BuildWorkflow(w)
	return w
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
	return w.WorkflowFunc(ctx, input)
}
func (w *ZenroomWorkflow) ExecuteWorkflow(
	ctx workflow.Context,
	input workflowengine.WorkflowInput,
) (workflowengine.WorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	ctx = workflow.WithActivityOptions(ctx, w.GetOptions())

	payload, err := workflowengine.DecodePayload[ZenroomWorkflowPayload](input.Payload)
	if err != nil {
		return workflowengine.WorkflowResult{}, workflowengine.NewMissingOrInvalidPayloadError(
			err,
			input.RunMetadata,
		)
	}

	var sideEffectResult struct {
		TmpDir  string
		CmdArgs []string
	}

	err = workflow.SideEffect(ctx, func(_ workflow.Context) interface{} {
		tmpDirLocal, err := os.MkdirTemp("", "zenroom-workflow-")
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.MkdirFailed]
			return workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: err.Error(),
				},
			)
		}

		errCode := errorcodes.Codes[errorcodes.WriteFileFailed]
		contractPath := filepath.Join(tmpDirLocal, "contract.zen")
		if err := os.WriteFile(contractPath, []byte(payload.Contract), 0600); err != nil {
			return workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: err.Error(),
				},
			)
		}

		cmdArgsLocal := make([]string, 0)

		if payload.Keys != "" {
			keysPath := filepath.Join(tmpDirLocal, "keys.json")
			if err := os.WriteFile(keysPath, []byte(payload.Keys), 0600); err != nil {
				return workflowengine.NewAppError(
					workflowengine.WorkflowError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
					},
				)
			}
			cmdArgsLocal = append(cmdArgsLocal, "-k", "/tmp/keys.json")
		}

		if payload.Data != "" {
			dataPath := filepath.Join(tmpDirLocal, "data.json")
			if err := os.WriteFile(dataPath, []byte(payload.Data), 0600); err != nil {
				return workflowengine.NewAppError(
					workflowengine.WorkflowError{
						Code:    errCode.Code,
						Summary: errCode.Description,
						Message: err.Error(),
					},
				)
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
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
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
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			err,
			input.RunMetadata,
		)
	}
	cli, err := newDockerClient()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.DockerClientCreationFailed]
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	output, ok := result.Output.(map[string]any)
	errCode := errorcodes.Codes[errorcodes.UnexpectedDockerOutput]
	if !ok {
		if !ok {
			msg := fmt.Sprintf("unexpected output type: %T", result.Output)
			appErr := workflowengine.NewAppError(
				workflowengine.WorkflowError{
					Code:    errCode.Code,
					Summary: errCode.Description,
					Message: msg,
					Details: map[string]any{"payload": result.Output},
				},
			)
			return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
				appErr,
				input.RunMetadata,
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
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "invalid exit code",
				Details: map[string]any{"payload": output["exitCode"]},
			},
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	stderr, ok := output["stderr"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "invalid stderr ",
				Details: map[string]any{"payload": output["stderr"]},
			},
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	stdout, ok := output["stdout"].(string)
	if !ok {
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "invalid stdout ",
				Details: map[string]any{"payload": output["stout"]},
			},
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	if int(exitCode) != 0 {
		errCode := errorcodes.Codes[errorcodes.ZenroomExecutionFailed]
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: fmt.Sprintf("zenroom exited with code %d", int(exitCode)),
				Details: map[string]any{
					"exit_code": int(exitCode),
					"stderr":    stderr,
					"stdout":    stdout,
				},
			},
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
	}
	// Parse stdout as JSON
	var parsedOutput map[string]any
	if err := json.Unmarshal([]byte(stdout), &parsedOutput); err != nil {
		logger.Error("Failed to parse stdout JSON", "error", err)
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		appErr := workflowengine.NewAppError(
			workflowengine.WorkflowError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
				Details: map[string]any{"payload": stdout},
			},
		)
		return workflowengine.WorkflowResult{}, workflowengine.NewWorkflowError(
			appErr,
			input.RunMetadata,
		)
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
	c, err := zenroomTemporalClient(namespace)
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
