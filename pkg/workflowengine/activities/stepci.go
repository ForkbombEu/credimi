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

	rendered, err := RenderYAML(yamlString, input.Payload)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.TemplateRenderFailed]
		return a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
		)
	}

	input.Payload["yaml"] = rendered
	return nil
}

func (a *StepCIWorkflowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	yamlContent, ok := input.Payload["yaml"].(string)
	if !ok || yamlContent == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidPayload]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: 'yaml", errCode.Description),
		)
	}

	filtered := make(map[string]string)
	for k, v := range input.Config {
		if k != "template" {
			filtered[k] = v
		}
	}

	jsonBytes, err := json.Marshal(filtered)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
		)
	}

	binDir := utils.GetEnvironmentVariable("BIN", ".bin")
	binName := "stepci-captured-runner"
	binPath := fmt.Sprintf("%s/%s", binDir, binName)
	args := []string{yamlContent, "-s", string(jsonBytes)}

	cmd := exec.CommandContext(ctx, binPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CommandExecutionFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf(errCode.Description+": %v", err),
			stderrStr, // pass only stderr here
		)
	}
	var outputJSON StepCICliReturns
	if err := json.Unmarshal(stdoutBuf.Bytes(), &outputJSON); err != nil {
		result.Output = stdoutStr
		return result, nil //nolint:all
	}

	result.Output = outputJSON
	if !outputJSON.Passed {
		errCode := errorcodes.Codes[errorcodes.StepCIRunFailed]
		return result, a.NewActivityError(
			errCode.Code,
			errCode.Description,
			outputJSON,
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
