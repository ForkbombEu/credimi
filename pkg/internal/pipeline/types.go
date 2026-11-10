// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"slices"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

// Struct-typed fields carry omitzero next to omitempty: encoding/json ignores omitempty on
// structs, so without omitzero every serialized definition repeats empty objects. omitempty
// stays for the YAML encoder and the schema generator, which read it as "optional".
type WorkflowDefinition struct {
	Version string            `yaml:"version,omitempty" json:"version,omitempty"`
	Name    string            `yaml:"name"              json:"name"`
	Runtime RuntimeConfig     `yaml:"runtime,omitempty" json:"runtime,omitempty,omitzero"`
	Config  map[string]any    `yaml:"config,omitempty"  json:"config,omitempty"`
	Steps   []StepDefinition  `yaml:"steps,omitempty"   json:"steps,omitempty"`
	Finally FinallyDefinition `yaml:"finally,omitempty" json:"finally,omitempty,omitzero"`
}

type StepSpec struct {
	ID              string                 `yaml:"id"                         json:"id"`
	Use             string                 `yaml:"use"                        json:"use"`
	With            StepInputs             `yaml:"with"                       json:"with"`
	ActivityOptions *ActivityOptionsConfig `yaml:"activity_options,omitempty" json:"activity_options,omitempty"`
	Metadata        map[string]any         `yaml:"metadata,omitempty"         json:"metadata,omitempty"`
}

type StepDefinition struct {
	StepSpec        `     yaml:",inline"                     json:",inline"`
	ContinueOnError bool `yaml:"continue_on_error,omitempty" json:"continue_on_error,omitempty"`
	// LinkedTo names the immediately previous main step. Consecutive linked steps form a
	// block that runs as one Linked Steps Workflow, retried as a whole by Temporal.
	LinkedTo  string                     `yaml:"linked_to,omitempty"  json:"linked_to,omitempty"`
	OnError   []*OnErrorStepDefinition   `yaml:"on_error,omitempty"   json:"on_error,omitempty"`
	OnSuccess []*OnSuccessStepDefinition `yaml:"on_success,omitempty" json:"on_success,omitempty"`
}

// LinkedStepsWorkflowName is the Temporal workflow type that runs a block of linked steps.
const LinkedStepsWorkflowName = "Run linked steps"

// LinkedStepBlocks returns the inclusive [start, end] index ranges of the blocks of at
// least two steps formed by consecutive linked_to fields.
func LinkedStepBlocks(steps []StepDefinition) [][2]int {
	var blocks [][2]int
	for i := 0; i < len(steps); i++ {
		end := i
		for end+1 < len(steps) && steps[end+1].LinkedTo != "" {
			end++
		}
		if end > i {
			blocks = append(blocks, [2]int{i, end})
		}
		i = end
	}
	return blocks
}

type OnErrorStepDefinition struct {
	StepSpec `yaml:",inline" json:",inline"`
}

type OnSuccessStepDefinition struct {
	StepSpec `yaml:",inline" json:",inline"`
}

type FinallyStepDefinition struct {
	StepSpec `yaml:",inline" json:",inline"`
}

type FinallyDefinition struct {
	Always    []FinallyStepDefinition `yaml:"always,omitempty"     json:"always,omitempty"`
	OnSuccess []FinallyStepDefinition `yaml:"on_success,omitempty" json:"on_success,omitempty"`
	OnFailure []FinallyStepDefinition `yaml:"on_failure,omitempty" json:"on_failure,omitempty"`
}

func (f FinallyDefinition) IsZero() bool {
	return len(f.Always) == 0 && len(f.OnSuccess) == 0 && len(f.OnFailure) == 0
}

func (f FinallyDefinition) AllSteps() []FinallyStepDefinition {
	return slices.Concat(f.Always, f.OnSuccess, f.OnFailure)
}

type StepInputs struct {
	Config  map[string]any `yaml:"config,omitempty"  json:"config,omitempty"`
	Payload map[string]any `yaml:"payload,omitempty" json:"payload,omitempty"`
}

type RuntimeConfig struct {
	Fixture  map[string]string `yaml:"fixture,omitempty"  json:"fixture,omitempty"`
	Schedule struct {
		Interval *time.Duration `yaml:"interval,omitempty" json:"interval,omitempty"`
	} `yaml:"schedule,omitempty" json:"schedule,omitempty,omitzero"`
	// GlobalDeviceID is retained as an internal Go field while callers are migrated;
	// the serialized pipeline contract is exclusively global_device_id.
	GlobalDeviceID          string `yaml:"global_device_id,omitempty"           json:"global_device_id,omitempty"`
	DisableAndroidPlayStore bool   `yaml:"disable_android_play_store,omitempty" json:"disable_android_play_store,omitempty"`
	// NoVideo skips the device screen recording for runs that would outlast the
	// recording time limit anyway; the device log and a final screenshot are kept.
	NoVideo  bool `yaml:"no_video,omitempty" json:"no_video,omitempty"`
	Debug    bool `yaml:"debug,omitempty"    json:"debug,omitempty"`
	Temporal struct {
		ExecutionTimeout string                `yaml:"execution_timeout,omitempty" json:"execution_timeout,omitempty"`
		ActivityOptions  ActivityOptionsConfig `yaml:"activity_options,omitempty" json:"activity_options,omitempty,omitzero"`
	} `yaml:"temporal,omitempty" json:"temporal,omitempty,omitzero"`
}

type ActivityOptionsConfig struct {
	ScheduleToCloseTimeout string      `yaml:"schedule_to_close_timeout,omitempty" json:"schedule_to_close_timeout,omitempty"` //nolint
	StartToCloseTimeout    string      `yaml:"start_to_close_timeout,omitempty"    json:"start_to_close_timeout,omitempty"`
	HeartbeatTimeout       string      `yaml:"heartbeat_timeout,omitempty"         json:"heartbeat_timeout,omitempty"`
	RetryPolicy            RetryPolicy `yaml:"retry_policy,omitempty"              json:"retry_policy,omitempty,omitzero"`
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
