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

type CliError struct {
	Message string  `json:"message"`
	Stack   *string `json:"stack,omitempty"`
}

type StepCIWorkflowActivity struct {
	workflowengine.BaseActivity
}

// StepCIWorkflowActivityPayload is the input for the StepCIWorkflowActivity
type StepCIWorkflowActivityPayload struct {
	Yaml    string            `json:"yaml"              yaml:"yaml"`
	Data    map[string]any    `json:"data,omitempty"    yaml:"data,omitempty"`
	Secrets map[string]string `json:"secrets,omitempty" yaml:"secrets,omitempty"`
	Env     string            `json:"env,omitempty"     yaml:"env,omitempty"`
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
			errCode.Code,
			errCode.Description+": 'template'",
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
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
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

	secrets := make(map[string]string)
	if payload.Secrets != nil {
		secrets = payload.Secrets
	}
	secretBytes, err := json.Marshal(secrets)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
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

		return result, a.NewActivityError(
			errorcodes.Codes[errorcodes.CommandExecutionFailed].Code,
			err.Error(),
			stderrBuf.String(),
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
		return result, a.NewActivityError(
			errorcodes.Codes[errorcodes.StepCIRunFailed].Code,
			"stepci run failed",
			output,
		)
	}

	return result, nil
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
