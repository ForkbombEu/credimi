// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"os/exec"
	"strings"
	"text/template"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

// StepCIWorkflowActivity is an activity that runs a StepCI workflow
type StepCIWorkflowActivity struct{}

// Name returns the name of the StepCIWorkflowActivity, which describes
// the purpose of this activity as running an automation workflow of API calls.
func (StepCIWorkflowActivity) Name() string {
	return "Run an automation workflow of API calls"
}

// Configure injects the parsed template and token into the payload
func (a *StepCIWorkflowActivity) Configure(
	input *workflowengine.ActivityInput,
) error {
	yamlString := input.Config["template"]
	if yamlString == "" {
		return errors.New("missing required config: 'template'")
	}

	rendered, err := RenderYAML(yamlString, input.Payload)
	if err != nil {
		return fmt.Errorf("failed to render YAML: %w", err)
	}

	input.Payload["yaml"] = rendered

	return nil
}

// Execute runs the StepCI workflow activity. It takes the activity input,
// validates the presence of a YAML payload, and executes the StepCI runner
// binary with the provided configuration and secrets.
//
// Parameters:
//   - ctx: The context for managing the execution lifecycle.
//   - input: The input for the activity, containing the payload and configuration.
//
// Returns:
//   - workflowengine.ActivityResult: The result of the activity execution,
//     including any output produced by the StepCI runner.
//   - error: An error if the execution fails, including details about the failure.
//
// The method performs the following steps:
//  1. Validates the presence of a "yaml" key in the input payload.
//  2. Constructs a secret string from the input configuration, excluding the "template" key.
//  3. Determines the path to the StepCI runner binary.
//  4. Executes the binary with the YAML content and secret string as arguments.
//  5. Captures and parses the output from the binary, returning it as a JSON object.
//
// If any step fails, the method returns a failure result with an appropriate error message.
func (a *StepCIWorkflowActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	var result workflowengine.ActivityResult

	yamlContent, ok := input.Payload["yaml"].(string)
	if !ok || yamlContent == "" {
		return workflowengine.Fail(&result, "missing rendered YAML in payload")
	}

	var secretString bytes.Buffer
	for k, v := range input.Config {
		if k != "template" {
			if secretString.Len() > 0 {
				secretString.WriteString(" ")
			}
			secretString.WriteString(fmt.Sprintf("%s=%s", k, v))
		}
	}

	binDir := utils.GetEnvironmentVariable("BIN", ".bin", false)
	binName := "stepci-captured-runner"
	binPath := fmt.Sprintf("%s/%s", binDir, binName)
	// Build the arguments for the command
	args := []string{yamlContent, "-s", secretString.String()}

	cmd := exec.CommandContext(ctx, binPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return workflowengine.Fail(
			&result,
			fmt.Sprintf("stepci runner failed: %v\nOutput: %s", err, output),
		)
	}
	var outputJSON map[string]any

	if err := json.Unmarshal(output, &outputJSON); err != nil {
		return workflowengine.Fail(&result, fmt.Sprintf("failed to unmarshal JSON output: %v", err))
	}
	result.Output = outputJSON
	return result, nil
}

// RenderYAML takes a YAML template string and a data map, renders the template
// using the provided data, and returns the resulting string. The function
// supports custom template functions provided by the sprout library.
//
// Parameters:
//   - yamlString: A string containing the YAML template with placeholders.
//   - data: A map containing key-value pairs to populate the template.
//
// Returns:
//   - A string containing the rendered YAML with placeholders replaced by
//     corresponding values from the data map.
//   - An error if the template parsing or execution fails.
//
// The function also decodes any HTML entities in the rendered string and trims
// leading/trailing whitespace or extra newlines from the result.
func RenderYAML(yamlString string, data map[string]interface{}) (string, error) {
	handler := sprout.New(
		sprout.WithGroups(all.RegistryGroup()),
	)
	funcs := handler.Build()

	tmpl, err := template.New("yaml").Delims("[[", "]]").
		Funcs(funcs).
		Parse(yamlString)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	// Decode HTML entities from the rendered string
	result := html.UnescapeString(buf.String())

	// Remove any leading/trailing whitespace or extra newlines from the result
	return strings.TrimSpace(result), nil
}
