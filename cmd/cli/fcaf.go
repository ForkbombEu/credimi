// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/fcaf/catalog"
	"github.com/forkbombeu/credimi/pkg/fcaf/dsl"
	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const fcafSourceBaseURL = "https://github.com/eu-digital-identity-wallet/eudi-doc-functional-conformance-assessment/blob/88ab69a/docs/fcaf/suts/wallet_solution/relying_party/"

var (
	fcafDir       string
	fcafOutput    string
	fcafFilter    string
	fcafTestsFile string
	fcafRunnerID  string
	fcafTimeout   time.Duration
	fcafInterval  time.Duration
)

type fcafRun struct {
	Name        string
	Path        string
	Status      string
	Error       string
	StartedAt   time.Time
	FinishedAt  time.Time
	WorkflowID  string
	DetailsURL  string
	DetailsFile string
	Screenshots []string
	Tests       []fcafTestRun
}

type fcafTestRun struct {
	ID         string
	Title      string
	SourceURL  string
	Status     string
	Message    string
	Validators []fcafValidatorRun
}

type fcafValidatorRun struct {
	ID           string
	Validator    string
	Input        string
	Params       map[string]any
	Status       string
	Message      string
	EvidenceKeys []string
}

type fcafReport struct {
	GeneratedAt time.Time
	Runs        []fcafRun
	Passed      int
	Failed      int
	Blocked     int
	Errors      int
}

// NewFCAFCommand runs the reusable FCAF pipeline YAML files sequentially.
func NewFCAFCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fcaf",
		Short: "Run FCAF wallet pipelines and collect evidence",
	}
	run := &cobra.Command{
		Use:   "run",
		Short: "Run FCAF pipeline YAML files sequentially",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFCAF(cmd.Context())
		},
	}
	run.Flags().StringVar(&fcafDir, "dir", "config_templates/fcaf/wallet_solution/relying_party/pipelines", "directory containing pipeline YAML files")
	run.Flags().StringVar(&fcafOutput, "output", "", "optional local directory for JSON, screenshots, and HTML output")
	run.Flags().StringVar(&fcafFilter, "filter", "", "run only files whose name contains this value")
	run.Flags().StringVar(&fcafTestsFile, "tests-file", "", "YAML file containing a tests list of FCAF test IDs")
	run.Flags().StringVar(&fcafRunnerID, "runner-id", "", "registered mobile runner identifier for mobile-automation steps")
	run.Flags().DurationVar(&fcafTimeout, "timeout", 30*time.Minute, "maximum time allowed for each pipeline")
	run.Flags().DurationVar(&fcafInterval, "interval", 5*time.Second, "queue polling interval")
	run.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")
	run.Flags().StringVarP(&instanceURL, "instance", "i", "http://localhost:8090", "URL of the Credimi instance")
	cmd.AddCommand(run)
	return cmd
}

func runFCAF(ctx context.Context) error {
	if strings.TrimSpace(apiKey) == "" {
		apiKey = strings.TrimSpace(os.Getenv("CREDIMI_API_KEY"))
	}
	if apiKey == "" || apiKey == "replace-with-local-api-key" {
		return fmt.Errorf("CREDIMI_API_KEY is required; generate a local key and set it in .env or use --api-key")
	}
	token, err := authenticate(ctx)
	if err != nil {
		return err
	}
	orgID, org, err := getMyOrganization(ctx, token)
	if err != nil {
		return err
	}
	if fcafTestsFile != "" {
		selection, err := selectionForFCAFTests(fcafTestsFile, fcafDir)
		if err != nil {
			return err
		}
		if err := runSelectedFCAFAssessment(ctx, token, orgID, org, selection); err != nil {
			if localFCAFOutputEnabled() {
				if reportErr := writeFCAFErrorReport(selection, err); reportErr != nil {
					return fmt.Errorf("FCAF run failed: %v; write failure report: %w", err, reportErr)
				}
			}
			return err
		}
		return nil
	}
	paths, err := filepath.Glob(filepath.Join(fcafDir, "*.yaml"))
	if err != nil {
		return err
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return fmt.Errorf("no YAML pipelines found in %s", fcafDir)
	}
	runs := make([]fcafRun, 0, len(paths))
	report := fcafReport{}
	for _, path := range paths {
		if fcafFilter != "" && !strings.Contains(filepath.Base(path), fcafFilter) {
			continue
		}
		run, err := runOneFCAF(ctx, token, orgID, org, path)
		if err != nil {
			run.Status, run.Error = "error", err.Error()
		}
		switch run.Status {
		case "completed", "success", "passed":
			report.Passed++
		case "failed", "canceled":
			report.Failed++
		case "queued", "starting", "running":
			report.Blocked++
		default:
			report.Errors++
		}
		runs = append(runs, run)
		if localFCAFOutputEnabled() {
			b, _ := json.MarshalIndent(run, "", "  ")
			name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) + ".json"
			if err := os.MkdirAll(fcafOutput, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(fcafOutput, name), b, 0o644); err != nil {
				return err
			}
		}
		fmt.Printf("%-8s %s\n", run.Status, run.Name)
	}
	if !localFCAFOutputEnabled() {
		return nil
	}
	return writeFCAFReport(fcafReport{GeneratedAt: time.Now(), Runs: runs, Passed: report.Passed, Failed: report.Failed, Blocked: report.Blocked, Errors: report.Errors})
}

func localFCAFOutputEnabled() bool {
	return strings.TrimSpace(fcafOutput) != ""
}

func writeFCAFErrorReport(selection fcafTestSelection, runErr error) error {
	runs := make([]fcafRun, 0, len(selection.TestIDs))
	for _, testID := range selection.TestIDs {
		test := selection.Tests[testID]
		runs = append(runs, fcafRun{
			Name:       test.Title,
			Path:       testID,
			Status:     "error",
			Error:      runErr.Error(),
			StartedAt:  time.Now(),
			FinishedAt: time.Now(),
			Tests: []fcafTestRun{{
				ID:        test.ID,
				Title:     test.Title,
				SourceURL: fcafSourceBaseURL + test.Source.Path,
				Status:    "error",
				Message:   runErr.Error(),
			}},
		})
	}
	return writeFCAFReport(fcafReport{
		GeneratedAt: time.Now(),
		Runs:        runs,
		Errors:      len(runs),
	})
}

type fcafTestsFileConfig struct {
	Tests []string `yaml:"tests"`
}

type fcafTestSelection struct {
	TestIDs []string
	Tests   map[string]dsl.TestDefinition
	Paths   []string
}

func pathsForFCAFTests(filename, pipelineDir string) ([]string, error) {
	selection, err := selectionForFCAFTests(filename, pipelineDir)
	if err != nil {
		return nil, err
	}
	return selection.Paths, nil
}

func selectionForFCAFTests(filename, pipelineDir string) (fcafTestSelection, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fcafTestSelection{}, fmt.Errorf("read FCAF tests file: %w", err)
	}
	var config fcafTestsFileConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fcafTestSelection{}, fmt.Errorf("parse FCAF tests file: %w", err)
	}
	if len(config.Tests) == 0 {
		return fcafTestSelection{}, fmt.Errorf("FCAF tests file %s contains no tests", filename)
	}
	catalogRoot := filepath.Dir(pipelineDir)
	cat, err := catalog.Load(catalogRoot)
	if err != nil {
		return fcafTestSelection{}, fmt.Errorf("load FCAF catalog: %w", err)
	}
	selection := fcafTestSelection{
		TestIDs: make([]string, 0, len(config.Tests)),
		Tests:   make(map[string]dsl.TestDefinition, len(config.Tests)),
		Paths:   make([]string, 0, len(config.Tests)),
	}
	paths := make([]string, 0, len(config.Tests))
	seen := map[string]struct{}{}
	for _, testID := range config.Tests {
		testID = strings.TrimSpace(testID)
		if testID == "" {
			return fcafTestSelection{}, fmt.Errorf("FCAF tests file contains an empty test ID")
		}
		test, ok := cat.Tests[testID]
		if !ok {
			return fcafTestSelection{}, fmt.Errorf("FCAF test ID %q not found", testID)
		}
		if len(test.Preconditions) == 0 {
			return fcafTestSelection{}, fmt.Errorf("FCAF test %q has no preconditions to run", testID)
		}
		pipelineName := ""
		for _, ref := range test.Preconditions {
			precondition, ok := cat.Preconditions[ref.Ref]
			if !ok {
				return fcafTestSelection{}, fmt.Errorf("FCAF test %q references unknown precondition %q", testID, ref.Ref)
			}
			if precondition.PipelineID != "" {
				pipelineName = filepath.Base(strings.TrimSuffix(precondition.PipelineID, "/")) + ".yaml"
				break
			}
		}
		if pipelineName == "" {
			return fcafTestSelection{}, fmt.Errorf("FCAF test %q has no pipeline precondition", testID)
		}
		path := filepath.Join(pipelineDir, pipelineName)
		if _, err := os.Stat(path); err != nil {
			return fcafTestSelection{}, fmt.Errorf("pipeline for FCAF test %q not found at %s: %w", testID, path, err)
		}
		selection.TestIDs = append(selection.TestIDs, testID)
		selection.Tests[testID] = test
		if _, exists := seen[path]; !exists {
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	selection.Paths = paths
	return selection, nil
}

func runSelectedFCAFAssessment(ctx context.Context, token, orgID, org string, selection fcafTestSelection) error {
	// Run each selected test independently so a slow or failed mobile workflow
	// cannot hold the entire suite's report hostage.
	if len(selection.TestIDs) > 1 {
		for _, testID := range selection.TestIDs {
			single := selection
			single.TestIDs = []string{testID}
			if err := runSelectedFCAFAssessment(ctx, token, orgID, org, single); err != nil {
				continue
			}
		}
		return nil
	}

	for _, path := range selection.Paths {
		input, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read selected FCAF pipeline %s: %w", path, err)
		}
		if fcafRunnerID != "" {
			input, err = setFCAFRunnerID(input, fcafRunnerID)
			if err != nil {
				return fmt.Errorf("set runner ID in %s: %w", path, err)
			}
		}
		name, err := parsePipelineName(input)
		if err != nil {
			return err
		}
		if _, err := findOrCreatePipeline(ctx, token, orgID, &PipelineCLIInput{Name: name, YAML: string(input)}); err != nil {
			return fmt.Errorf("import selected FCAF pipeline %s: %w", path, err)
		}
	}
	body, err := json.Marshal(map[string]any{
		"test_ids":  selection.TestIDs,
		"suite":     "wallet_solution/relying_party",
		"runner_id": fcafRunnerID,
		// The report is only complete after the assessment workflow finishes.
		// The API returns the final report for both successful (200) and failed
		// (409) assessments when this is enabled.
		"wait_for_completion": true,
	})
	if err != nil {
		return err
	}
	status, response, err := postFCAF(ctx, token, body)
	if err != nil {
		return err
	}
	assessmentError := ""
	if status != http.StatusOK && status != http.StatusConflict {
		assessmentError = fmt.Sprintf("FCAF assessment returned HTTP %d: %s", status, response)
	}
	var envelope map[string]any
	if err := json.Unmarshal(response, &envelope); err != nil {
		if assessmentError == "" {
			return fmt.Errorf("decode FCAF assessment response: %w", err)
		}
		envelope = map[string]any{
			"status": status,
			"error":  string(response),
		}
	}
	if localFCAFOutputEnabled() {
		if err := os.MkdirAll(fcafOutput, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(fcafOutput, "fcaf-assessment.json"), response, 0o644); err != nil {
			return err
		}
	}

	workflowID, _ := envelope["workflow_id"].(string)
	runID, _ := envelope["run_id"].(string)
	detailsURL := ""
	if workflowID != "" && runID != "" {
		detailsURL = utils.JoinURL(instanceURL, "my", "tests", "runs", workflowID, runID)
	}
	reportData, _ := envelope["report"].(map[string]any)
	executed, _ := reportData["executed_tests"].([]any)
	executedByID := map[string]map[string]any{}
	for _, raw := range executed {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		id, _ := item["test_id"].(string)
		executedByID[id] = item
	}

	runs := make([]fcafRun, 0, len(selection.TestIDs))
	for _, testID := range selection.TestIDs {
		test := selection.Tests[testID]
		item := executedByID[testID]
		run := fcafRun{
			Name:       test.Title,
			Path:       testID,
			Status:     "not_run",
			StartedAt:  time.Now(),
			FinishedAt: time.Now(),
			WorkflowID: workflowID,
			DetailsURL: detailsURL,
			Tests: []fcafTestRun{{
				ID:        test.ID,
				Title:     test.Title,
				SourceURL: fcafSourceBaseURL + test.Source.Path,
				Status:    "not_run",
			}},
		}
		if item != nil {
			status, _ := item["status"].(string)
			if status != "" {
				run.Status = status
				run.Tests[0].Status = status
			}
			run.Tests[0].Message, _ = item["message"].(string)
			if outcome, ok := item["outcome"].(map[string]any); ok {
				if message, ok := outcome["reason"].(string); ok && run.Tests[0].Message == "" {
					run.Tests[0].Message = message
				}
			}
			if assertions, ok := item["assertions"].([]any); ok {
				for _, raw := range assertions {
					assertion, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					id, _ := assertion["id"].(string)
					definition := findFCAFAssertion(test.Assertions, id)
					validator := fcafValidatorRun{ID: id, Validator: definition.Validator, Input: definition.Input, Params: definition.Params}
					validator.Status, _ = assertion["status"].(string)
					validator.Message, _ = assertion["message"].(string)
					if keys, ok := assertion["evidence_keys"].([]any); ok {
						for _, key := range keys {
							if value, ok := key.(string); ok {
								validator.EvidenceKeys = append(validator.EvidenceKeys, value)
							}
						}
					}
					run.Tests[0].Validators = append(run.Tests[0].Validators, validator)
				}
			}
		}
		if assessmentError != "" && run.Tests[0].Message == "" {
			run.Tests[0].Message = assessmentError
		}
		run.FinishedAt = time.Now()
		runs = append(runs, run)
		if localFCAFOutputEnabled() {
			data, _ := json.MarshalIndent(run, "", "  ")
			name := strings.ReplaceAll(testID, "/", "-") + ".json"
			if err := os.WriteFile(filepath.Join(fcafOutput, name), data, 0o644); err != nil {
				return err
			}
		}
	}
	passed, failed, blocked, errors := 0, 0, 0, 0
	for _, run := range runs {
		switch run.Status {
		case "passed", "pass", "completed":
			passed++
		case "failed", "fail":
			failed++
		case "blocked":
			blocked++
		default:
			errors++
		}
	}
	if localFCAFOutputEnabled() {
		if err := writeFCAFReport(fcafReport{GeneratedAt: time.Now(), Runs: runs, Passed: passed, Failed: failed, Blocked: blocked, Errors: errors}); err != nil {
			return err
		}
	}
	if assessmentError != "" && !strings.Contains(assessmentError, "CRE229") {
		if workflowID, runID := assessmentWorkflowIDs(assessmentError); workflowID != "" && runID != "" {
			if details, err := pollFCAFExecution(ctx, token, org, workflowID, runID); err == nil {
				if localFCAFOutputEnabled() {
					path := filepath.Join(fcafOutput, selection.TestIDs[0]+"-execution.json")
					data, _ := json.MarshalIndent(details, "", "  ")
					_ = os.WriteFile(path, data, 0o644)
					runs[0].DetailsFile = path
				}
				runs[0].Status = "failed"
				runs[0].Tests[0].Status = "failed"
				runs[0].Tests[0].Message = "Pipeline execution reached a terminal result; see execution details."
			}
		}
	}
	if assessmentError != "" {
		for i := range runs {
			if runs[i].Status == "not_run" {
				runs[i].Status = "error"
				runs[i].Tests[0].Status = "error"
			}
		}
		errors = len(runs) - passed - failed - blocked
	}
	if assessmentError != "" {
		return fmt.Errorf("%s", assessmentError)
	}
	return nil
}

var assessmentWorkflowPattern = regexp.MustCompile(`workflowID: ([^,]+), runID: ([^)]+)`)

func assessmentWorkflowIDs(message string) (string, string) {
	matches := assessmentWorkflowPattern.FindStringSubmatch(message)
	if len(matches) != 3 {
		return "", ""
	}
	return matches[1], matches[2]
}

func pollFCAFExecution(ctx context.Context, token, namespace, workflowID, runID string) (map[string]any, error) {
	deadline := time.NewTimer(fcafTimeout)
	defer deadline.Stop()
	for {
		endpoint := utils.JoinURL(instanceURL, "api", "pipeline", "execution-details", namespace, workflowID, runID)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		pollKey := strings.TrimSpace(os.Getenv("CREDIMI_INTERNAL_ADMIN_KEY"))
		if pollKey == "" {
			pollKey = apiKey
		}
		req.Header.Set("Credimi-Api-Key", pollKey)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			data, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var details map[string]any
				if readErr != nil {
					return nil, readErr
				}
				if err := json.Unmarshal(data, &details); err != nil {
					return nil, err
				}
				return details, nil
			}
		}
		select {
		case <-deadline.C:
			return nil, fmt.Errorf("timed out polling FCAF pipeline execution")
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(fcafInterval):
		}
	}
}

func findFCAFAssertion(assertions []dsl.AssertionDefinition, id string) dsl.AssertionDefinition {
	for _, assertion := range assertions {
		if assertion.ID == id {
			return assertion
		}
	}
	return dsl.AssertionDefinition{ID: id}
}

func postFCAF(ctx context.Context, token string, body []byte) (int, []byte, error) {
	target := utils.JoinURL(instanceURL, "api", "fcaf", "run")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(body)))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

func runOneFCAF(ctx context.Context, token, orgID, org, path string) (fcafRun, error) {
	started := time.Now()
	run := fcafRun{Name: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Path: path, StartedAt: started}
	input, err := os.ReadFile(path)
	if err != nil {
		return run, err
	}
	if fcafRunnerID != "" {
		input, err = setFCAFRunnerID(input, fcafRunnerID)
		if err != nil {
			return run, fmt.Errorf("set runner ID in %s: %w", path, err)
		}
	}
	parsed, err := parsePipelineName(input)
	if err != nil {
		return run, err
	}
	run.Name = parsed
	rec, err := findOrCreatePipeline(ctx, token, orgID, &PipelineCLIInput{Name: parsed, YAML: string(input)})
	if err != nil {
		return run, err
	}
	identifier := fmt.Sprintf("%s/%s", org, rec["canonified_name"])
	body, _ := json.Marshal(map[string]string{"pipeline_identifier": identifier, "yaml": string(input)})
	status, response, err := postPipelineRequest(ctx, token, "queue", body)
	if err != nil {
		return run, err
	}
	if status != http.StatusOK {
		return run, fmt.Errorf("queue returned HTTP %d: %s", status, response)
	}
	var queued pipelineQueueResponse
	if err := json.Unmarshal(response, &queued); err != nil {
		return run, err
	}
	run.DetailsURL = queued.RunURL
	if queued.TicketID == "" {
		return run, fmt.Errorf("queue returned no ticket_id")
	}
	deadline := time.NewTimer(fcafTimeout)
	defer deadline.Stop()
	lastQueueStatusWarning := ""
	for {
		if queued.Status == "completed" || queued.Status == "failed" || queued.Status == "canceled" || queued.Status == "not_found" {
			run.Status = queued.Status
			run.Error = queued.ErrorMessage
			run.FinishedAt = time.Now()
			if queued.RunURL != "" && localFCAFOutputEnabled() {
				run.Screenshots, run.DetailsFile = collectFCAFImages(ctx, token, queued.RunURL, fcafOutput, run.Name)
			}
			return run, nil
		}
		select {
		case <-ctx.Done():
			return run, ctx.Err()
		case <-deadline.C:
			if lastQueueStatusWarning != "" {
				return run, fmt.Errorf(
					"timed out waiting for ticket %s; last polling error: %s",
					queued.TicketID,
					lastQueueStatusWarning,
				)
			}
			return run, fmt.Errorf("timed out waiting for ticket %s", queued.TicketID)
		case <-time.After(fcafInterval):
		}
		query := url.Values{}
		query.Set("runner_ids", strings.Join(queued.RunnerIDs, ","))
		status, response, err = getPipeline(ctx, token, "queue/"+queued.TicketID, query)
		if err != nil {
			if ctx.Err() != nil {
				return run, ctx.Err()
			}
			lastQueueStatusWarning = err.Error()
			continue
		}
		if status != http.StatusOK {
			if retryableFCAFQueueStatus(status) {
				lastQueueStatusWarning = fmt.Sprintf("HTTP %d: %s", status, response)
				continue
			}
			return run, fmt.Errorf("queue status returned HTTP %d: %s", status, response)
		}
		lastQueueStatusWarning = ""
		if err := json.Unmarshal(response, &queued); err != nil {
			return run, err
		}
		if queued.RunURL != "" {
			run.DetailsURL = queued.RunURL
		}
	}
}

func retryableFCAFQueueStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= http.StatusInternalServerError
}

func setFCAFRunnerID(data []byte, runnerID string) ([]byte, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, err
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("pipeline YAML root is not a mapping")
	}
	root := document.Content[0]
	find := func(mapping *yaml.Node, key string) *yaml.Node {
		for i := 0; i+1 < len(mapping.Content); i += 2 {
			if mapping.Content[i].Value == key {
				return mapping.Content[i+1]
			}
		}
		return nil
	}
	runtime := find(root, "runtime")
	if runtime == nil {
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "runtime"},
			&yaml.Node{Kind: yaml.MappingNode},
		)
		runtime = root.Content[len(root.Content)-1]
	}
	if runtime.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("runtime is not a mapping")
	}
	global := find(runtime, "global_runner_id")
	if global == nil {
		runtime.Content = append(runtime.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "global_runner_id"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: runnerID},
		)
	} else {
		global.Kind = yaml.ScalarNode
		global.Tag = "!!str"
		global.Value = runnerID
	}
	return yaml.Marshal(&document)
}

func getPipeline(ctx context.Context, token, path string, query ...url.Values) (int, []byte, error) {
	target := utils.JoinURL(instanceURL, "api", "pipeline", path)
	if len(query) > 0 {
		target += "?" + query[0].Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

func collectFCAFImages(ctx context.Context, token, detailsURL, output, runName string) ([]string, string) {
	status, body, err := getAbsolute(ctx, token, detailsURL)
	if err != nil || status < 200 || status >= 300 {
		return nil, ""
	}
	detailsFile := strings.TrimSuffix(filepath.Base(runName), filepath.Ext(runName)) + ".details.json"
	if err := os.WriteFile(filepath.Join(output, detailsFile), body, 0o644); err != nil {
		detailsFile = ""
	}
	var value any
	if json.Unmarshal(body, &value) != nil {
		return nil, detailsFile
	}
	urls := make([]string, 0)
	collectStrings(value, &urls)
	result := make([]string, 0)
	imageNumber := 0
	for _, imageURL := range urls {
		lower := strings.ToLower(imageURL)
		if !strings.Contains(lower, ".png") && !strings.Contains(lower, ".jpg") && !strings.Contains(lower, ".jpeg") {
			continue
		}
		status, data, err := getAbsolute(ctx, token, imageURL)
		if err != nil || status < 200 || status >= 300 {
			continue
		}
		name := fmt.Sprintf("%s-evidence-%03d%s", strings.TrimSuffix(filepath.Base(runName), filepath.Ext(runName)), imageNumber, filepath.Ext(imageURL))
		if err := os.WriteFile(filepath.Join(output, name), data, 0o644); err == nil {
			result = append(result, name)
			imageNumber++
		}
	}
	return result, detailsFile
}

func collectStrings(value any, out *[]string) {
	switch v := value.(type) {
	case string:
		if strings.HasPrefix(v, "http") {
			*out = append(*out, v)
		}
	case []any:
		for _, item := range v {
			collectStrings(item, out)
		}
	case map[string]any:
		for _, item := range v {
			collectStrings(item, out)
		}
	}
}

func getAbsolute(ctx context.Context, token, target string) (int, []byte, error) {
	if !strings.HasPrefix(target, "http") {
		target = utils.JoinURL(instanceURL, strings.TrimPrefix(target, "/"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	return resp.StatusCode, b, err
}

func writeFCAFReport(report fcafReport) error {
	f, err := os.Create(filepath.Join(fcafOutput, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	const page = `<!doctype html><html><head><meta charset="utf-8"><title>FCAF run report</title><style>body{font:14px sans-serif;line-height:1.4;margin:2rem;color:#202124}h1{margin-bottom:.25rem}.summary{display:flex;gap:1rem;margin:1rem 0}.summary span{border:1px solid #ccc;border-radius:4px;padding:.5rem .75rem}.passed{color:#137333}.failed,.error{color:#b3261e}.blocked{color:#8a4b00}table{border-collapse:collapse;width:100%;table-layout:fixed}td,th{border:1px solid #d8d8d8;padding:.6rem;text-align:left;vertical-align:top}th:nth-child(1){width:24%}th:nth-child(2){width:9%}th:nth-child(3){width:13%}th:nth-child(4){width:26%}th:nth-child(5){width:28%}img{max-width:180px;max-height:140px;object-fit:contain;border:1px solid #ddd;margin:.25rem}code{word-break:break-all}</style></head><body><h1>FCAF pipeline run report</h1><p>Generated {{.GeneratedAt}}</p><div class="summary"><span class="passed">Passed: {{.Passed}}</span><span class="failed">Failed: {{.Failed}}</span><span class="blocked">Blocked: {{.Blocked}}</span><span class="error">Errors: {{.Errors}}</span></div><table><tr><th>Pipeline</th><th>Status</th><th>Duration</th><th>Run details</th><th>Evidence</th></tr>{{range .Runs}}<tr><td><strong>{{.Name}}</strong><br><code>{{.Path}}</code></td><td class="{{statusClass .Status}}">{{.Status}}</td><td>{{duration .StartedAt .FinishedAt}}</td><td>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}{{if .WorkflowID}}<div>Workflow ID: <code>{{.WorkflowID}}</code></div>{{end}}{{if .DetailsFile}}<a href="{{.DetailsFile}}">details JSON</a><br>{{end}}{{if .DetailsURL}}<a href="{{.DetailsURL}}">Open Credimi workflow</a>{{else}}No workflow link yet{{end}}</td><td>{{if .Screenshots}}{{range .Screenshots}}<a href="{{.}}"><img src="{{.}}" alt="evidence screenshot"></a>{{end}}{{else}}No downloaded screenshots{{end}}</td></tr>{{end}}</table></body></html>`
	funcs := template.FuncMap{"statusClass": func(status string) string {
		if status == "completed" || status == "success" || status == "passed" {
			return "passed"
		}
		if status == "failed" || status == "canceled" {
			return "failed"
		}
		if status == "error" {
			return "error"
		}
		return "blocked"
	}, "duration": func(start, finish time.Time) string {
		if finish.IsZero() {
			return "-"
		}
		return finish.Sub(start).Round(time.Second).String()
	}}
	return template.Must(template.New("fcaf").Funcs(funcs).Parse(page)).Execute(f, report)
}
