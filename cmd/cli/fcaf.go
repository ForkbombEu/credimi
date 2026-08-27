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
	"sort"
	"strings"
	"time"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	fcafDir        string
	fcafOutput     string
	fcafFilter     string
	fcafRunnerID   string
	fcafTimeout    time.Duration
	fcafInterval   time.Duration
	fcafImportsDir string
)

type fcafRun struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	WorkflowID  string    `json:"workflow_id,omitempty"`
	DetailsURL  string    `json:"details_url,omitempty"`
	DetailsFile string    `json:"details_file,omitempty"`
	Screenshots []string  `json:"screenshots,omitempty"`
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
	syncCmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize FCAF pipelines and credential definitions without running them",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return syncFCAF(cmd.Context())
		},
	}
	run.Flags().
		StringVar(&fcafDir, "dir", "config_templates/fcaf/wallet_solution/relying_party/pipelines", "directory containing pipeline YAML files")
	run.Flags().
		StringVar(&fcafOutput, "output", "", "optional local directory for JSON, screenshots, and HTML output")
	run.Flags().
		StringVar(&fcafFilter, "filter", "", "run only files whose name contains this value")
	run.Flags().
		StringVar(&fcafRunnerID, "runner-id", "", "registered mobile runner identifier for mobile-automation steps")
	run.Flags().
		DurationVar(&fcafTimeout, "timeout", 30*time.Minute, "maximum time allowed for each pipeline")
	run.Flags().DurationVar(&fcafInterval, "interval", 5*time.Second, "queue polling interval")
	syncCmd.Flags().
		StringVar(&fcafImportsDir, "imports-dir", "config_templates/fcaf/imports", "directory containing repository-backed FCAF imports")
	syncCmd.Flags().
		StringVar(&fcafRunnerID, "runner-id", "", "registered mobile runner identifier to write into mobile-automation pipelines")
	run.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")
	run.Flags().
		StringVarP(&instanceURL, "instance", "i", "http://localhost:8090", "URL of the Credimi instance")
	syncCmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")
	syncCmd.Flags().
		StringVarP(&instanceURL, "instance", "i", "http://localhost:8090", "URL of the Credimi instance")
	cmd.AddCommand(run)
	cmd.AddCommand(syncCmd)
	return cmd
}

type fcafCredentialImport struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	YAML        string `yaml:"yaml"`
}

type fcafCredentialCode struct {
	Env struct {
		Host                      string `yaml:"host"`
		CredentialConfigurationID string `yaml:"credential_configuration_id"`
	} `yaml:"env"`
}

//nolint:gocyclo // Sync keeps pipeline, credential, and wallet-action failures contextual.
func syncFCAF(ctx context.Context) error {
	if strings.TrimSpace(apiKey) == "" {
		apiKey = strings.TrimSpace(os.Getenv("CREDIMI_API_KEY"))
	}
	if apiKey == "" || apiKey == "replace-with-local-api-key" {
		return fmt.Errorf(
			"CREDIMI_API_KEY is required; generate a local key and set it in .env or use --api-key",
		)
	}
	token, err := authenticate(ctx)
	if err != nil {
		return err
	}
	orgID, _, err := getMyOrganization(ctx, token)
	if err != nil {
		return err
	}

	pipelinePaths, err := filepath.Glob(filepath.Join(fcafDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("find FCAF pipelines: %w", err)
	}
	sort.Strings(pipelinePaths)
	for _, path := range pipelinePaths {
		input, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read pipeline %s: %w", path, readErr)
		}
		if fcafRunnerID != "" {
			input, readErr = setFCAFRunnerID(input, fcafRunnerID)
			if readErr != nil {
				return fmt.Errorf("set runner ID in %s: %w", path, readErr)
			}
		}
		name, parseErr := parsePipelineName(input)
		if parseErr != nil {
			return fmt.Errorf("parse pipeline %s: %w", path, parseErr)
		}
		if _, syncErr := findOrCreatePipelineWithValidation(
			ctx,
			token,
			orgID,
			&PipelineCLIInput{Name: name, YAML: string(input)},
			fcafRunnerID != "",
		); syncErr != nil {
			return fmt.Errorf("sync pipeline %s: %w", path, syncErr)
		}
		fmt.Printf("synced pipeline %s\n", name)
	}

	credentialPaths, err := filepath.Glob(
		filepath.Join(fcafImportsDir, "**", "credentials", "**", "*.yaml"),
	)
	if err != nil {
		return fmt.Errorf("find FCAF credentials: %w", err)
	}
	// filepath.Glob does not support **; walk the import tree explicitly.
	credentialPaths = credentialPaths[:0]
	err = filepath.WalkDir(
		fcafImportsDir,
		func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && strings.Contains(filepath.ToSlash(path), "/credentials/") &&
				strings.HasSuffix(path, ".yaml") {
				credentialPaths = append(credentialPaths, path)
			}
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("walk FCAF imports: %w", err)
	}
	sort.Strings(credentialPaths)
	for _, path := range credentialPaths {
		input, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read credential %s: %w", path, readErr)
		}
		var definition fcafCredentialImport
		if parseErr := yaml.Unmarshal(input, &definition); parseErr != nil {
			return fmt.Errorf("parse credential %s: %w", path, parseErr)
		}
		if definition.Name == "" || definition.YAML == "" {
			return fmt.Errorf("credential %s must define name and yaml", path)
		}
		if err := syncFCAFCredential(ctx, token, orgID, definition); err != nil {
			return fmt.Errorf("sync credential %s: %w", path, err)
		}
		fmt.Printf("synced credential %s\n", definition.Name)
	}

	walletActionPaths := make([]string, 0)
	err = filepath.WalkDir(
		fcafImportsDir,
		func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !entry.IsDir() && strings.Contains(filepath.ToSlash(path), "/wallet/") &&
				strings.HasSuffix(path, ".yaml") {
				walletActionPaths = append(walletActionPaths, path)
			}
			return nil
		},
	)
	if err != nil {
		return fmt.Errorf("walk FCAF wallet actions: %w", err)
	}
	sort.Strings(walletActionPaths)
	for _, path := range walletActionPaths {
		input, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read wallet action %s: %w", path, readErr)
		}
		name, parseErr := parseWalletActionName(input)
		if parseErr != nil {
			continue
		}
		updated, syncErr := syncFCAFWalletAction(ctx, token, orgID, name, string(input))
		if syncErr != nil {
			return syncErr
		}
		if updated {
			fmt.Printf("synced wallet action %s\n", name)
		}
	}
	return nil
}

func parseWalletActionName(input []byte) (string, error) {
	parts := strings.SplitN(string(input), "---", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("wallet action is missing YAML front matter")
	}
	var metadata struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal([]byte(parts[0]), &metadata); err != nil {
		return "", err
	}
	if metadata.Name == "" {
		return "", fmt.Errorf("wallet action has no name")
	}
	return metadata.Name, nil
}

func syncFCAFWalletAction(ctx context.Context, token, orgID, name, code string) (bool, error) {
	filter := url.Values{}
	filter.Set(
		"filter",
		fmt.Sprintf(
			`owner="%s" && name="%s"`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(name),
		),
	)
	filter.Set("perPage", "20")
	endpoint := utils.JoinURL(
		instanceURL,
		"api",
		"collections",
		"wallet_actions",
		"records",
	) + "?" + filter.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("find wallet action failed: HTTP %d: %s", resp.StatusCode, body)
	}
	var records struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return false, err
	}
	if len(records.Items) == 0 {
		return false, nil
	}
	if len(records.Items) != 1 {
		return false, fmt.Errorf(
			"expected one wallet action named %q, found %d",
			name,
			len(records.Items),
		)
	}
	id, ok := records.Items[0]["id"].(string)
	if !ok || id == "" {
		return false, fmt.Errorf("wallet action %q has no record id", name)
	}
	body, err := json.Marshal(map[string]string{"code": code})
	if err != nil {
		return false, err
	}
	patchURL := utils.JoinURL(instanceURL, "api", "collections", "wallet_actions", "records", id)
	patchReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		patchURL,
		strings.NewReader(string(body)),
	)
	if err != nil {
		return false, err
	}
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		return false, err
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(patchResp.Body)
		return false, fmt.Errorf(
			"update wallet action failed: HTTP %d: %s",
			patchResp.StatusCode,
			responseBody,
		)
	}
	return true, nil
}

func syncFCAFCredential(
	ctx context.Context,
	token, orgID string,
	definition fcafCredentialImport,
) error {
	filter := url.Values{}
	filter.Set(
		"filter",
		fmt.Sprintf(
			`owner="%s" && name="%s"`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(definition.Name),
		),
	)
	filter.Set("perPage", "20")
	endpoint := utils.JoinURL(
		instanceURL,
		"api",
		"collections",
		"credentials",
		"records",
	) + "?" + filter.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("find credential failed: HTTP %d: %s", resp.StatusCode, body)
	}
	var records struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return err
	}
	if len(records.Items) != 1 {
		return fmt.Errorf(
			"expected one credential record named %q, found %d",
			definition.Name,
			len(records.Items),
		)
	}
	id, ok := records.Items[0]["id"].(string)
	if !ok || id == "" {
		return fmt.Errorf("credential %q has no record id", definition.Name)
	}
	var code fcafCredentialCode
	if err := yaml.Unmarshal([]byte(definition.YAML), &code); err != nil {
		return fmt.Errorf("parse credential code: %w", err)
	}
	updates := map[string]any{
		"name":        definition.Name,
		"description": definition.Description,
		"yaml":        definition.YAML,
	}
	if strings.Contains(code.Env.Host, "capture-wallet") &&
		code.Env.CredentialConfigurationID != "" {
		offerBody, err := json.Marshal(
			map[string]string{"credential_configuration_id": code.Env.CredentialConfigurationID},
		)
		if err != nil {
			return err
		}
		offerReq, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			code.Env.Host,
			strings.NewReader(string(offerBody)),
		)
		if err != nil {
			return err
		}
		offerReq.Header.Set("Content-Type", "application/json")
		offerResp, err := http.DefaultClient.Do(offerReq)
		if err != nil {
			return fmt.Errorf("regenerate credential offer: %w", err)
		}
		defer offerResp.Body.Close()
		if offerResp.StatusCode != http.StatusCreated {
			responseBody, _ := io.ReadAll(offerResp.Body)
			return fmt.Errorf(
				"regenerate credential offer failed: HTTP %d: %s",
				offerResp.StatusCode,
				responseBody,
			)
		}
		var offer struct {
			Deeplink string `json:"deeplink"`
		}
		if err := json.NewDecoder(offerResp.Body).Decode(&offer); err != nil {
			return err
		}
		if offer.Deeplink == "" {
			return fmt.Errorf(
				"regenerated credential offer for %q has no deeplink",
				definition.Name,
			)
		}
		updates["deeplink"] = offer.Deeplink
	}
	body, err := json.Marshal(updates)
	if err != nil {
		return err
	}
	patchURL := utils.JoinURL(instanceURL, "api", "collections", "credentials", "records", id)
	patchReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		patchURL,
		strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	patchReq.Header.Set("Authorization", "Bearer "+token)
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(patchReq)
	if err != nil {
		return err
	}
	defer patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(patchResp.Body)
		return fmt.Errorf(
			"update credential failed: HTTP %d: %s",
			patchResp.StatusCode,
			responseBody,
		)
	}
	return nil
}

func runFCAF(ctx context.Context) error {
	if strings.TrimSpace(apiKey) == "" {
		apiKey = strings.TrimSpace(os.Getenv("CREDIMI_API_KEY"))
	}
	if apiKey == "" || apiKey == "replace-with-local-api-key" {
		return fmt.Errorf(
			"CREDIMI_API_KEY is required; generate a local key and set it in .env or use --api-key",
		)
	}
	token, err := authenticate(ctx)
	if err != nil {
		return err
	}
	orgID, org, err := getMyOrganization(ctx, token)
	if err != nil {
		return err
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
	return writeFCAFReport(
		fcafReport{
			GeneratedAt: time.Now(),
			Runs:        runs,
			Passed:      report.Passed,
			Failed:      report.Failed,
			Blocked:     report.Blocked,
			Errors:      report.Errors,
		},
	)
}

func localFCAFOutputEnabled() bool {
	return strings.TrimSpace(fcafOutput) != ""
}

func runOneFCAF(ctx context.Context, token, orgID, org, path string) (fcafRun, error) {
	started := time.Now()
	run := fcafRun{
		Name:      strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
		Path:      path,
		StartedAt: started,
	}
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
	rec, err := findOrCreatePipeline(
		ctx,
		token,
		orgID,
		&PipelineCLIInput{Name: parsed, YAML: string(input)},
	)
	if err != nil {
		return run, err
	}
	identifier := fmt.Sprintf("%s/%s", org, rec["canonified_name"])
	body, _ := json.Marshal(
		map[string]string{"pipeline_identifier": identifier, "yaml": string(input)},
	)
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
		if queued.Status == "completed" || queued.Status == "failed" ||
			queued.Status == "canceled" ||
			queued.Status == "not_found" {
			run.Status = queued.Status
			run.Error = queued.ErrorMessage
			run.FinishedAt = time.Now()
			if queued.RunURL != "" && localFCAFOutputEnabled() {
				run.Screenshots, run.DetailsFile = collectFCAFImages(
					ctx,
					token,
					queued.RunURL,
					fcafOutput,
					run.Name,
				)
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

func getPipeline(
	ctx context.Context,
	token, path string,
	query ...url.Values,
) (int, []byte, error) {
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

func collectFCAFImages(
	ctx context.Context,
	token, detailsURL, output, runName string,
) ([]string, string) {
	status, body, err := getAbsolute(ctx, token, detailsURL)
	if err != nil || status < 200 || status >= 300 {
		return nil, ""
	}
	detailsFile := strings.TrimSuffix(
		filepath.Base(runName),
		filepath.Ext(runName),
	) + ".details.json"
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
	seenURLs := make(map[string]struct{}, len(urls))
	imageNumber := 0
	for _, imageURL := range urls {
		lower := strings.ToLower(imageURL)
		if !strings.Contains(lower, ".png") && !strings.Contains(lower, ".jpg") &&
			!strings.Contains(lower, ".jpeg") {
			continue
		}
		if _, seen := seenURLs[imageURL]; seen {
			continue
		}
		seenURLs[imageURL] = struct{}{}
		status, data, err := getAbsolute(ctx, token, imageURL)
		if err != nil || status < 200 || status >= 300 {
			continue
		}
		name := fmt.Sprintf(
			"%s-evidence-%03d%s",
			strings.TrimSuffix(filepath.Base(runName), filepath.Ext(runName)),
			imageNumber,
			filepath.Ext(imageURL),
		)
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
