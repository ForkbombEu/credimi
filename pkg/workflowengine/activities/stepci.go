// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

const maxStepCIResultErrorBytes = 256 * 1024

type TestResult struct {
	ID            string       `json:"id"`
	Name          *string      `json:"name,omitempty"`
	Steps         []StepResult `json:"steps"`
	Passed        bool         `json:"passed"`
	Timestamp     time.Time    `json:"timestamp"`
	Duration      float64      `json:"duration"`
	CO2           float64      `json:"co2"`
	BytesSent     int64        `json:"bytesSent"`
	BytesReceived int64        `json:"bytesReceived"`
}

type StepCICliReturns struct {
	Passed   bool           `json:"passed"`
	Messages []string       `json:"messages"`
	Captures map[string]any `json:"captures"`
	Tests    []TestResult   `json:"tests"`
	Errors   []CliError     `json:"errors"`
}

type StepResult struct {
	ID            *string         `json:"id,omitempty"`
	TestID        string          `json:"testId"`
	Name          *string         `json:"name,omitempty"`
	Retries       *int            `json:"retries,omitempty"`
	Captures      *map[string]any `json:"captures,omitempty"`
	Cookies       any             `json:"cookies,omitempty"`
	Errored       bool            `json:"errored"`
	ErrorMessage  *string         `json:"errorMessage,omitempty"`
	Passed        bool            `json:"passed"`
	Skipped       bool            `json:"skipped"`
	Timestamp     time.Time       `json:"timestamp"`
	ResponseTime  int             `json:"responseTime"`
	Duration      int             `json:"duration"`
	CO2           float64         `json:"co2"`
	BytesSent     int             `json:"bytesSent"`
	BytesReceived int             `json:"bytesReceived"`
}

type StepCIFailureStep struct {
	TestID       string `json:"testId"`
	Name         string `json:"name,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Errored      bool   `json:"errored"`
	Skipped      bool   `json:"skipped"`
}

type StepCIFailureSummary struct {
	Passed      bool                `json:"passed"`
	Messages    []string            `json:"messages,omitempty"`
	Errors      []CliError          `json:"errors,omitempty"`
	FailedSteps []StepCIFailureStep `json:"failedSteps,omitempty"`
}

type CliError struct {
	Message string  `json:"message"`
	Stack   *string `json:"stack,omitempty"`
}

type StepCIWorkflowActivity struct {
	workflowengine.BaseActivity
}

// StepCIWorkflowActivityPayload is the input for the StepCIWorkflowActivity
type StepCIWorkflowActivityPayload struct {
	Yaml string         `json:"yaml"           yaml:"yaml"`
	Data map[string]any `json:"data,omitempty" yaml:"data,omitempty"`
	Env  string         `json:"env,omitempty"  yaml:"env,omitempty"`
}

func NewStepCIWorkflowActivity() *StepCIWorkflowActivity {
	return &StepCIWorkflowActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Run an automation workflow of API calls",
		},
	}
}
func (a *StepCIWorkflowActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *StepCIWorkflowActivity) Configure(input *workflowengine.ActivityInput) error {
	yamlString := input.Config["template"]
	if yamlString == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		return a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: "template is required",
			},
		)
	}
	payload, err := workflowengine.DecodePayload[StepCIWorkflowActivityPayload](input.Payload)
	if err != nil {
		return a.NewMissingOrInvalidPayloadError(err)
	}
	rendered, err := RenderYAML(yamlString, payload.Data)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.TemplateRenderFailed]
		return a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}

	payload.Yaml = rendered
	input.Payload = payload
	return nil
}

func (a *StepCIWorkflowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	payload, err := workflowengine.DecodePayload[StepCIWorkflowActivityPayload](input.Payload)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	secretBytes, err := json.Marshal(input.Secrets)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:    errCode.Code,
				Summary: errCode.Description,
				Message: err.Error(),
			},
		)
	}

	binDir := utils.GetEnvironmentVariable("BIN", ".bin")
	binPath := filepath.Join(binDir, "stepci-captured-runner")

	args := []string{payload.Yaml, "-s", string(secretBytes)}

	if payload.Env != "" {
		args = append(args, "--env", payload.Env)
	}

	cmd := exec.CommandContext(ctx, binPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = workflowengine.RunCommandWithCancellation(ctx, cmd, 2*time.Second)
	if err != nil {
		// Temporal cancellation → propagate cleanly
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:     errCode.Code,
				Summary:  "StepCI command failed",
				Message:  fmt.Sprintf("stepci-captured-runner exited with error: %v", err),
				Category: "external_command",
				Details: map[string]any{
					"command": binPath,
					"args":    args,
					"stderr":  stderrBuf.String(),
					"stdout":  stdoutBuf.String(),
				},
			},
		)
	}
	stdoutStr := stdoutBuf.String()

	var output StepCICliReturns
	if err := json.Unmarshal(stdoutBuf.Bytes(), &output); err != nil {
		result.Output = stdoutStr
		return result, nil //nolint:nilerr
	}

	result.Output = output

	if !output.Passed {
		errCode := errorcodes.Codes[errorcodes.StepCIRunFailed]
		return result, a.NewActivityError(
			workflowengine.ActivityError{
				Code:     errCode.Code,
				Summary:  "StepCI checks failed",
				Message:  "One or more StepCI assertions failed.",
				Category: "test_failure",
				Details:  stepCIFailureDetails(output),
			},
		)
	}

	return result, nil
}

func stepCIFailureDetails(output StepCICliReturns) map[string]any {
	details := map[string]any{
		"summary": summarizeStepCIFailure(output),
	}

	raw, err := json.Marshal(output)
	if err != nil {
		details["result_error"] = err.Error()
		return details
	}

	details["result_size_bytes"] = len(raw)
	if len(raw) <= maxStepCIResultErrorBytes {
		details["result"] = output
		return details
	}

	details["result_omitted"] = true
	details["result_omitted_reason"] = "StepCI result is too large to store safely in a Temporal failure payload."
	return details
}

func summarizeStepCIFailure(output StepCICliReturns) StepCIFailureSummary {
	summary := StepCIFailureSummary{
		Passed:   output.Passed,
		Messages: output.Messages,
		Errors:   output.Errors,
	}

	for _, test := range output.Tests {
		for _, step := range test.Steps {
			if step.Passed || (step.Skipped && !step.Errored) {
				continue
			}

			failed := StepCIFailureStep{
				TestID:  step.TestID,
				Errored: step.Errored,
				Skipped: step.Skipped,
			}
			if step.Name != nil {
				failed.Name = *step.Name
			}
			if step.ErrorMessage != nil {
				failed.ErrorMessage = *step.ErrorMessage
			}
			summary.FailedSteps = append(summary.FailedSteps, failed)
		}
	}

	return summary
}

func RenderYAML(yamlString string, data map[string]any) (string, error) {
	handler := sprout.New(sprout.WithGroups(all.RegistryGroup()))
	funcs := handler.Build()

	tmpl, err := template.New("yaml").Delims("[[", "]]").Funcs(funcs).Parse(yamlString)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	result := html.UnescapeString(buf.String())
	return strings.TrimSpace(result), nil
}
