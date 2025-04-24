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

	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"

	"github.com/forkbombeu/didimo/pkg/utils"
	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
)

type StepCIWorkflowActivity struct{}

func (StepCIWorkflowActivity) Name() string {
	return "Run an automation workflow of API calls"
}

// Configure injects the parsed template and token into the payload
func (a *StepCIWorkflowActivity) Configure(
	ctx context.Context,
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
