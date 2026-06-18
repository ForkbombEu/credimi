// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflowengine is a package that provides a framework for defining and executing workflows.
// It includes interfaces and types for activities, activity input and output, and error handling.
package workflowengine

import (
	"context"
	"os/exec"
	"syscall"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// ActivityInput represents the input to an activity, including payload and configuration.
// The payload is a map of string keys to any type of value.
type ActivityInput struct {
	Payload any               `json:"payload,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
	Secrets map[string]any    `json:"secrets,omitempty"`
}

// ActivityResult represents the result of an activity execution, including output, errors, and logs.
// It is designed to be extensible, allowing for different types of output and error handling.
type ActivityResult struct {
	Output  any            `json:"output,omitempty"`
	Log     []string       `json:"log,omitempty"`
	Secrets map[string]any `json:"secrets,omitempty"`
}

type ActivityError struct {
	Code         string         `json:"code"`
	Summary      string         `json:"summary"`
	Message      string         `json:"message,omitempty"`
	ActivityName string         `json:"activityName,omitempty"`
	Category     string         `json:"category,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}

// BaseActivity provides the common interface for all activities.
type Activity interface {
	Name() string
	NewActivityError(failure ActivityError) error
	NewNonRetryableActivityError(failure ActivityError) error
	NewMissingOrInvalidPayloadError(err error) error
}

// BaseActivity provides a default implementation of the Activity interface.
type BaseActivity struct {
	Name string
}

// ExecutableActivity defines an activity that can be executed.
type ExecutableActivity interface {
	Activity
	Execute(ctx context.Context, input ActivityInput) (ActivityResult, error)
}

// ConfigurableActivity defines an activity that can be configured.
type ConfigurableActivity interface {
	Activity
	Configure(input *ActivityInput) error
}

func (a *BaseActivity) NewActivityError(
	failure ActivityError,
) error {
	failure = a.withActivityName(failure)
	return temporal.NewApplicationError(errorMessage(failure.Summary, failure.Message), failure.Code, failure)
}

func (a *BaseActivity) NewNonRetryableActivityError(
	failure ActivityError,
) error {
	failure = a.withActivityName(failure)
	return temporal.NewNonRetryableApplicationError(
		errorMessage(failure.Summary, failure.Message),
		failure.Code,
		nil,
		failure,
	)
}

func (a *BaseActivity) withActivityName(failure ActivityError) ActivityError {
	if failure.ActivityName == "" && a != nil {
		failure.ActivityName = a.Name
	}
	return failure
}

func (a *BaseActivity) NewMissingOrInvalidPayloadError(err error) error {
	errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
	return a.NewActivityError(
		ActivityError{
			Code:    errCode.Code,
			Summary: errCode.Description,
			Message: err.Error(),
		},
	)
}

func RunCommandWithCancellation(
	ctx context.Context,
	cmd *exec.Cmd,
	heartbeatInterval time.Duration,
) error {
	// Ensure child processes are killed as well
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	done := make(chan error, 1)

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		done <- cmd.Wait()
	}()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				// Kill process group
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
			return ctx.Err()

		case err := <-done:
			// If the command already finished successfully, keep that result even if
			// the activity context is canceled at roughly the same time.
			if err == nil {
				return nil
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err

		case <-ticker.C:
			safeRecordActivityHeartbeat(ctx, "running")
		}
	}
}

func safeRecordActivityHeartbeat(ctx context.Context, details ...any) {
	defer func() {
		// RecordHeartbeat panics if the context is not an activity context.
		// RunCommandWithCancellation is also used in unit tests with plain contexts.
		_ = recover()
	}()
	activity.RecordHeartbeat(ctx, details...)
}

// OutputKind represents the expected type of an activity output.
type OutputKind int

const (
	OutputAny OutputKind = iota
	OutputString
	OutputMap
	OutputArrayOfString
	OutputArrayOfMap
	OutputBool
)
