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
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
	"github.com/go-sprout/sprout"
	"github.com/go-sprout/sprout/group/all"
)

const (
	maxStepCIResultErrorBytes  = 256 * 1024
	maxStepCIFailureReportSize = 32 * 1024
	stepCIGenericFailureReport = "One or more StepCI assertions failed."
	stepCIReportTruncatedNote  = "\n… (StepCI report truncated)"
	stepCIMaskedSecret         = "***"
	minStepCIMaskedSecretLen   = 4
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
				Code:    errCode.Code,
				Summary: "StepCI checks failed",
				Message: formatStepCIFailureReport(
					output,
					parseStepCIRequests(stdoutBuf.Bytes()),
					input.Secrets,
				),
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

// stepCIRequests holds the executed request of each step, indexed like
// StepCICliReturns.Tests[i].Steps[j]. Requests are kept out of StepResult so request
// headers and bodies, which carry resolved secrets, stay out of activity outputs and
// failure details.
type stepCIRequests [][]*stepCIRequest

type stepCIRequest struct {
	Protocol string          `json:"protocol"`
	URL      string          `json:"url"`
	Method   string          `json:"method"`
	Headers  json.RawMessage `json:"headers,omitempty"`
	Body     any             `json:"body,omitempty"`
}

func parseStepCIRequests(stdout []byte) stepCIRequests {
	var raw struct {
		Tests []struct {
			Steps []struct {
				Request *stepCIRequest `json:"request"`
			} `json:"steps"`
		} `json:"tests"`
	}
	if err := json.Unmarshal(stdout, &raw); err != nil {
		return nil
	}

	requests := make(stepCIRequests, len(raw.Tests))
	for i, test := range raw.Tests {
		requests[i] = make([]*stepCIRequest, len(test.Steps))
		for j, step := range test.Steps {
			requests[i][j] = step.Request
		}
	}
	return requests
}

func (r stepCIRequests) at(test, step int) *stepCIRequest {
	if test >= len(r) || step >= len(r[test]) {
		return nil
	}
	return r[test][step]
}

// formatStepCIFailureReport renders the StepCI result as the human-readable report the
// StepCI CLI prints (failed checks with expected and received values, the request, and the
// response), followed by step and runner errors that the runner does not include in its
// messages. Secret values are masked.
func formatStepCIFailureReport(
	output StepCICliReturns,
	requests stepCIRequests,
	secrets map[string]any,
) string {
	var report strings.Builder

	writeSection := func(title string, lines []string) {
		if len(lines) == 0 {
			return
		}
		if report.Len() > 0 {
			report.WriteString("\n\n")
		}
		report.WriteString(title)
		for _, line := range lines {
			report.WriteString("\n  - ")
			report.WriteString(line)
		}
	}

	messages, unplaced := withStepCIRequests(output, requests)
	if messages != "" {
		report.WriteString(messages)
	}
	for _, block := range unplaced {
		if report.Len() > 0 {
			report.WriteString("\n\n")
		}
		report.WriteString(block)
	}

	var stepErrors []string
	for _, test := range output.Tests {
		for _, step := range test.Steps {
			if !step.Errored || step.ErrorMessage == nil || *step.ErrorMessage == "" {
				continue
			}
			stepErrors = append(stepErrors, stepCIStepName(step)+": "+*step.ErrorMessage)
		}
	}
	writeSection("Step errors:", stepErrors)

	var runnerErrors []string
	for _, cliErr := range output.Errors {
		if msg := strings.TrimSpace(cliErr.Message); msg != "" {
			runnerErrors = append(runnerErrors, msg)
		}
	}
	writeSection("Errors:", runnerErrors)

	if report.Len() == 0 {
		return stepCIGenericFailureReport
	}
	return truncateStepCIReport(maskStepCISecrets(report.String(), secrets))
}

// withStepCIRequests joins the runner messages and inserts each failed step's request
// after the "  Method:" line of that step's "Step Failed: <name>" block. Requests that
// cannot be placed in the messages are returned as standalone blocks.
func withStepCIRequests(output StepCICliReturns, requests stepCIRequests) (string, []string) {
	type failedStep struct {
		name    string
		request *stepCIRequest
	}
	var failed []failedStep
	for i, test := range output.Tests {
		for j, step := range test.Steps {
			if step.Passed {
				continue
			}
			failed = append(
				failed,
				failedStep{name: stepCIStepName(step), request: requests.at(i, j)},
			)
		}
	}

	lines := make([]string, 0, len(output.Messages)+len(failed))
	next := 0
	current := -1
	for _, message := range output.Messages {
		lines = append(lines, message)
		if name, ok := strings.CutPrefix(message, "Step Failed: "); ok {
			current = -1
			for k := next; k < len(failed); k++ {
				if failed[k].name == name {
					current, next = k, k+1
					break
				}
			}
			continue
		}
		if current >= 0 && strings.HasPrefix(message, "  Method:") &&
			failed[current].request != nil {
			lines = append(
				lines,
				"  Request:\n"+indentStepCILines(
					renderStepCIRequest(failed[current].request),
					"    ",
				),
			)
			failed[current].request = nil
			current = -1
		}
	}

	var unplaced []string
	for _, step := range failed {
		if step.request != nil {
			unplaced = append(unplaced, "Request of "+step.name+":\n"+
				indentStepCILines(renderStepCIRequest(step.request), "    "))
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), unplaced
}

// renderStepCIRequest mirrors renderHTTPRequest of the StepCI CLI: request line, one
// "name: value" line per header in sent order, then a blank line and the body when the
// body is text.
func renderStepCIRequest(request *stepCIRequest) string {
	var out strings.Builder
	out.WriteString(strings.TrimSpace(request.Method + " " + request.URL + " " + request.Protocol))
	for _, header := range orderedStepCIHeaders(request.Headers) {
		out.WriteString("\n")
		out.WriteString(header)
	}
	if body, ok := request.Body.(string); ok && body != "" {
		out.WriteString("\n\n")
		out.WriteString(body)
	}
	return out.String()
}

// orderedStepCIHeaders decodes a JSON object of headers keeping the order the runner
// sent them in.
func orderedStepCIHeaders(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		return nil
	}

	var headers []string
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return headers
		}
		var value any
		if err := decoder.Decode(&value); err != nil {
			return headers
		}
		text, ok := value.(string)
		if !ok {
			encoded, err := json.Marshal(value)
			if err != nil {
				continue
			}
			text = string(encoded)
		}
		headers = append(headers, fmt.Sprintf("%v: %s", key, text))
	}
	return headers
}

func indentStepCILines(text, prefix string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

func stepCIStepName(step StepResult) string {
	if step.Name != nil && *step.Name != "" {
		return *step.Name
	}
	if step.ID != nil && *step.ID != "" {
		return *step.ID
	}
	return step.TestID
}

// maskStepCISecrets replaces secret values in the report. Values shorter than
// minStepCIMaskedSecretLen are left alone: masking them would garble unrelated text.
func maskStepCISecrets(report string, secrets map[string]any) string {
	values := make([]string, 0, len(secrets))
	for _, value := range secrets {
		if text, ok := value.(string); ok && len(text) >= minStepCIMaskedSecretLen {
			values = append(values, text)
		}
	}
	if len(values) == 0 {
		return report
	}
	// Longest first, so a secret containing another secret is masked whole.
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	pairs := make([]string, 0, len(values)*2)
	for _, value := range values {
		pairs = append(pairs, value, stepCIMaskedSecret)
	}
	return strings.NewReplacer(pairs...).Replace(report)
}

func truncateStepCIReport(report string) string {
	if len(report) <= maxStepCIFailureReportSize {
		return report
	}
	cut := maxStepCIFailureReportSize - len(stepCIReportTruncatedNote)
	for cut > 0 && !utf8.RuneStart(report[cut]) {
		cut--
	}
	return report[:cut] + stepCIReportTruncatedNote
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
