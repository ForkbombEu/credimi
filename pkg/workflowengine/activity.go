// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflowengine is a package that provides a framework for defining and executing workflows.
// It includes interfaces and types for activities, activity input and output, and error handling.
package workflowengine

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/temporal"
)

// ActivityInput represents the input to an activity, including payload and configuration.
// The payload is a map of string keys to any type of value.
type ActivityInput struct {
	Payload map[string]any    `json:"payload,omitempty"`
	Config  map[string]string `json:"config,omitempty"`
}

// ActivityResult represents the result of an activity execution, including output, errors, and logs.
// It is designed to be extensible, allowing for different types of output and error handling.
type ActivityResult struct {
	Output any      `json:"output,omitempty"`
	Log    []string `json:"log,omitempty"`
}

// BaseActivity provides the common interface for all activities.
type Activity interface {
	Name() string
	NewActivityError(errorType string, errorMsg string, payload ...any) error
	NewNonRetryableActivityError(errorType string, errorMsg string, payload ...any) error
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
	errorType string,
	errorMsg string,
	activityPayload ...any,
) error {
	msg := fmt.Sprintf("[%s]: %s", a.Name, errorMsg)
	return temporal.NewApplicationError(msg, errorType, activityPayload)
}

func (a *BaseActivity) NewNonRetryableActivityError(
	errorType string,
	errorMsg string,
	activityPayload ...any,
) error {
	msg := fmt.Sprintf("[%s]: %s", a.Name, errorMsg)
	return temporal.NewNonRetryableApplicationError(msg, errorType, nil, activityPayload)
}

// OutputKind represents the expected type of an activity output.
type OutputKind int

const (
	OutputAny OutputKind = iota
	OutputString
	OutputMap
	OutputArrayOfString
	OutputArrayOfMap
)
