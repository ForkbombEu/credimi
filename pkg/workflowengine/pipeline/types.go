// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

type WorkflowDefinition struct {
	Version         string           `yaml:"version,omitempty"          json:"version,omitempty"`
	Name            string           `yaml:"name"                       json:"name"`
	Runtime         RuntimeConfig    `yaml:"runtime,omitempty"          json:"runtime,omitempty"`
	Config          map[string]any   `yaml:"config,omitempty"           json:"config,omitempty"`
	GlobalRunnerID  string           `yaml:"global_runner_id,omitempty" json:"global_runner_id,omitempty"`
	Steps           []StepDefinition `yaml:"steps,omitempty"            json:"steps,omitempty"`
}

type StepSpec struct {
	ID              string                 `yaml:"id"                         json:"id"`
	Use             string                 `yaml:"use"                        json:"use"`
	With            StepInputs             `yaml:"with"                       json:"with"`
	ActivityOptions *ActivityOptionsConfig `yaml:"activity_options,omitempty" json:"activity_options,omitempty"`
	Metadata        map[string]any         `yaml:"metadata,omitempty"         json:"metadata,omitempty"`
}

type StepDefinition struct {
	StepSpec        `                           yaml:",inline"                     json:",inline"`
	ContinueOnError bool                       `yaml:"continue_on_error,omitempty" json:"continue_on_error,omitempty"`
	OnError         []*OnErrorStepDefinition   `yaml:"on_error,omitempty"          json:"on_error,omitempty"`
	OnSuccess       []*OnSuccessStepDefinition `yaml:"on_success,omitempty"        json:"on_success,omitempty"`
}

type OnErrorStepDefinition struct {
	StepSpec `yaml:",inline" json:",inline"`
}

type OnSuccessStepDefinition struct {
	StepSpec `yaml:",inline" json:",inline"`
}

type StepInputs struct {
	Config  map[string]any `yaml:"config,omitempty"  json:"config,omitempty"`
	Payload map[string]any `yaml:"payload,omitempty" json:"payload,omitempty"`
}

type RuntimeConfig struct {
	Schedule struct {
		Interval *time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	} `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Debug    bool `yaml:"debug,omitempty"    json:"debug,omitempty"`
	Temporal struct {
		ExecutionTimeout string                `yaml:"execution_timeout,omitempty" json:"execution_timeout,omitempty"`
		ActivityOptions  ActivityOptionsConfig `yaml:"activity_options,omitempty" json:"activity_options,omitempty"`
	} `yaml:"temporal,omitempty" json:"temporal,omitempty"`
}

type ActivityOptionsConfig struct {
	ScheduleToCloseTimeout string      `yaml:"schedule_to_close_timeout,omitempty" json:"schedule_to_close_timeout,omitempty"` //nolint
	StartToCloseTimeout    string      `yaml:"start_to_close_timeout,omitempty"    json:"start_to_close_timeout,omitempty"`
	RetryPolicy            RetryPolicy `yaml:"retry_policy,omitempty"              json:"retry_policy,omitempty"`
}

type RetryPolicy struct {
	MaximumAttempts    int32   `yaml:"maximum_attempts,omitempty"    json:"maximum_attempts,omitempty"`
	InitialInterval    string  `yaml:"initial_interval,omitempty"    json:"initial_interval,omitempty"`
	MaximumInterval    string  `yaml:"maximum_interval,omitempty"    json:"maximum_interval,omitempty"`
	BackoffCoefficient float64 `yaml:"backoff_coefficient,omitempty" json:"backoff_coefficient,omitempty"`
}
type WorkflowOptions struct {
	Options         client.StartWorkflowOptions `json:"options"`
	Timeout         time.Duration               `json:"timeout,omitempty"`
	ActivityOptions workflow.ActivityOptions    `json:"activityOptions"`
}
