// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// workflowengine is a package that provides a framework for defining and executing workflows.
// It includes interfaces and types for activities, activity input and output, and error handling.
package workflowengine

import (
	"context"
	"errors"
)

// ActivityInput represents the input to an activity, including payload and configuration.
// The payload is a map of string keys to any type of value.
type ActivityInput struct {
	Payload map[string]any
	Config  map[string]string
}

// ActivityResult represents the result of an activity execution, including output, errors, and logs.
// It is designed to be extensible, allowing for different types of output and error handling.
type ActivityResult struct {
	Output any
	Errors []error
	Log    []string
}

// ExecutableActivity is an interface that defines the structure of an activity.
// It includes methods for executing the activity and retrieving its name.
type ExecutableActivity interface {
	Execute(ctx context.Context, input ActivityInput) (ActivityResult, error)
	Name() string
}

// ConfigurableActivity is an interface that defines the structure of a configurable activity.
type ConfigurableActivity interface {
	Configure(ctx context.Context, input *ActivityInput) error
}

// Fail is a utility function that appends an error message to the activity result's errors.
func Fail(result *ActivityResult, msg string) (ActivityResult, error) {
	err := errors.New(msg)
	result.Errors = append(result.Errors, err)
	return *result, err
}
