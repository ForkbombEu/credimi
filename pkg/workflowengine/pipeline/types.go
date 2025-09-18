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
	Version string                   `yaml:"version,omitempty"       json:"version,omitempty"`
	Name    string                   `yaml:"name"                    json:"name"`
	Runtime RuntimeConfig            `yaml:"runtime"                 json:"runtime"`
	Checks  map[string]WorkflowBlock `yaml:"custom_checks,omitempty" json:"custom_checks,omitempty"`
	Config  map[string]string        `yaml:"config,omitempty"        json:"config,omitempty"`
	Steps   []StepDefinition         `yaml:"steps,omitempty"         json:"steps,omitempty"`
}

type WorkflowBlock struct {
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Inputs      map[string]string `yaml:"inputs,omitempty"      json:"inputs,omitempty"`
	Outputs     map[string]string `yaml:"outputs,omitempty"     json:"outputs,omitempty"`
	Config      map[string]string `yaml:"config,omitempty"      json:"config,omitempty"`
	Steps       []StepDefinition  `yaml:"steps,omitempty"       json:"steps,omitempty"`
}

type StepDefinition struct {
	ID       string                 `yaml:"id"                 json:"id"`
	Use      string                 `yaml:"use"                json:"use"`
	With     StepInputs             `yaml:"with"               json:"with"`
	Retry    map[string]any         `yaml:"retry,omitempty"    json:"retry,omitempty"`
	Timeout  string                 `yaml:"timeout,omitempty"  json:"timeout,omitempty"`
	Metadata map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type StepInputs struct {
	Config  map[string]string      `yaml:"config,omitempty"  json:"config,omitempty"`
	Payload map[string]InputSource `yaml:"payload,omitempty" json:"payload,omitempty"`
}

type InputSource struct {
	Type  string `yaml:"type,omitempty"  json:"type,omitempty"`
	Value any    `yaml:"value,omitempty" json:"value,omitempty"`
}

type RuntimeConfig struct {
	Temporal struct {
		Namespace        string `yaml:"namespace,omitempty"         json:"namespace,omitempty"`
		TaskQueue        string `yaml:"task_queue,omitempty"        json:"task_queue,omitempty"`
		ExecutionTimeout string `yaml:"execution_timeout,omitempty" json:"execution_timeout,omitempty"`
		RetryPolicy      struct {
			MaximumAttempts    int32   `yaml:"maximum_attempts,omitempty"    json:"maximum_attempts,omitempty"`
			InitialInterval    string  `yaml:"initial_interval,omitempty"    json:"initial_interval,omitempty"`
			MaximumInterval    string  `yaml:"maximum_interval,omitempty"    json:"maximum_interval,omitempty"`
			BackoffCoefficient float64 `yaml:"backoff_coefficient,omitempty" json:"backoff_coefficient,omitempty"`
		} `yaml:"retry_policy" json:"retry_policy"`
	} `yaml:"temporal" json:"temporal"`
}

type WorkflowOptions struct {
	Namespace       string                      `json:"namespace,omitempty"`
	Options         client.StartWorkflowOptions `json:"options"`
	Timeout         time.Duration               `json:"timeout,omitempty"`
	ActivityOptions workflow.ActivityOptions    `json:"activityOptions"`
}
