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
	fcafDeviceID   string
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
		StringVar(&fcafDeviceID, "device-id", "", "registered mobile device identifier for mobile-automation steps")
	run.Flags().
		DurationVar(&fcafTimeout, "timeout", 30*time.Minute, "maximum time allowed for each pipeline")
	run.Flags().DurationVar(&fcafInterval, "interval", 5*time.Second, "queue polling interval")
	syncCmd.Flags().
		StringVar(&fcafImportsDir, "imports-dir", "config_templates/fcaf/imports", "directory containing repository-backed FCAF imports")
	syncCmd.Flags().
		StringVar(&fcafDir, "dir", "config_templates/fcaf/wallet_solution/relying_party/pipelines", "directory containing pipeline YAML files")
	syncCmd.Flags().
		StringVar(&fcafDeviceID, "device-id", "", "registered mobile device identifier to write into mobile-automation pipelines")
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

// fcafSourceOrg is the canonical organization that owns the bundled FCAF
// import references; sync rewrites it to the authenticated organization.
const fcafSourceOrg = "forkbomb-bv-andrea"

type fcafCredentialImport struct {
	Name           string `yaml:"name"`
	CanonifiedName string `yaml:"canonified_name"`
	Description    string `yaml:"description"`
	LogoURL        string `yaml:"logo_url"`
	Published      bool   `yaml:"published"`
	Imported       bool   `yaml:"imported"`
	Conformant     bool   `yaml:"conformant"`
	YAML           string `yaml:"yaml"`
}

type fcafIssuerImport struct {
	Name           string `yaml:"name"`
	CanonifiedName string `yaml:"canonified_name"`
	Description    string `yaml:"description"`
	URL            string `yaml:"url"`
	HomepageURL    string `yaml:"homepage_url"`
	RepoURL        string `yaml:"repo_url"`
	LogoURL        string `yaml:"logo_url"`
	Published      bool   `yaml:"published"`
	Imported       bool   `yaml:"imported"`
}

type fcafWalletActionsManifest struct {
	Organization string                `yaml:"organization"`
	Wallet       string                `yaml:"wallet"`
	Version      string                `yaml:"version"`
	Actions      []fcafWalletActionRef `yaml:"actions"`
}

type fcafWalletActionRef struct {
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
	Tags     string `yaml:"tags"`
	File     string `yaml:"file"`
}

type fcafCredentialCode struct {
	Env struct {
		Host                      string `yaml:"host"`
		CredentialConfigurationID string `yaml:"credential_configuration_id"`
	} `yaml:"env"`
}

//nolint:gocyclo // Sync keeps pipeline, issuer, credential, and wallet-action failures contextual.
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
	orgID, orgCanonName, err := getMyOrganization(ctx, token)
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
		input = rewriteFCAFOrganization(input, orgCanonName)
		if fcafDeviceID != "" {
			input, readErr = setFCAFDeviceID(input, fcafDeviceID)
			if readErr != nil {
				return fmt.Errorf("set device ID in %s: %w", path, readErr)
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
			fcafDeviceID != "",
		); syncErr != nil {
			return fmt.Errorf("sync pipeline %s: %w", path, syncErr)
		}
		fmt.Printf("synced pipeline %s\n", name)
	}

	if err := syncFCAFCredentialIssuers(ctx, token, orgID); err != nil {
		return err
	}
	if err := syncFCAFCredentials(ctx, token, orgID); err != nil {
		return err
	}
	return syncFCAFWalletActions(ctx, token, orgID)
}

// rewriteFCAFOrganization points org-scoped canonical references
// (global_device_id, action_id, version_id) at the target organization.
func rewriteFCAFOrganization(input []byte, orgCanonName string) []byte {
	if orgCanonName == "" || orgCanonName == fcafSourceOrg {
		return input
	}
	return []byte(strings.ReplaceAll(string(input), fcafSourceOrg+"/", orgCanonName+"/"))
}

// walkFCAFImports returns YAML files under a directory segment of the import tree.
func walkFCAFImports(root, segment string) ([]string, error) {
	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() && strings.Contains(filepath.ToSlash(path), segment) &&
			strings.HasSuffix(path, ".yaml") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// fcafListRecords lists collection records matching a PocketBase filter.
func fcafListRecords(
	ctx context.Context,
	token, collection, filter string,
) ([]map[string]any, error) {
	query := url.Values{}
	query.Set("filter", filter)
	query.Set("perPage", "20")
	endpoint := utils.JoinURL(
		instanceURL,
		"api",
		"collections",
		collection,
		"records",
	) + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list %s failed: HTTP %d: %s", collection, resp.StatusCode, body)
	}
	var records struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&records); err != nil {
		return nil, err
	}
	return records.Items, nil
}

// fcafCreateRecord creates a collection record.
func fcafCreateRecord(
	ctx context.Context,
	token, collection string,
	record map[string]any,
) error {
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		utils.JoinURL(instanceURL, "api", "collections", collection, "records"),
		strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"create %s failed: HTTP %d: %s",
			collection,
			resp.StatusCode,
			responseBody,
		)
	}
	return nil
}

// fcafPatchRecord updates a collection record by id.
func fcafPatchRecord(
	ctx context.Context,
	token, collection, id string,
	updates map[string]any,
) error {
	body, err := json.Marshal(updates)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPatch,
		utils.JoinURL(instanceURL, "api", "collections", collection, "records", id),
		strings.NewReader(string(body)),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"update %s failed: HTTP %d: %s",
			collection,
			resp.StatusCode,
			responseBody,
		)
	}
	return nil
}

func syncFCAFCredentialIssuers(ctx context.Context, token, orgID string) error {
	paths, err := walkFCAFImports(fcafImportsDir, "/credential-issuers/")
	if err != nil {
		return fmt.Errorf("walk FCAF credential issuers: %w", err)
	}
	for _, path := range paths {
		input, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read credential issuer %s: %w", path, readErr)
		}
		var issuer fcafIssuerImport
		if parseErr := yaml.Unmarshal(input, &issuer); parseErr != nil {
			return fmt.Errorf("parse credential issuer %s: %w", path, parseErr)
		}
		if issuer.Name == "" || issuer.URL == "" {
			return fmt.Errorf("credential issuer %s must define name and url", path)
		}
		if err := syncFCAFCredentialIssuer(ctx, token, orgID, issuer); err != nil {
			return fmt.Errorf("sync credential issuer %s: %w", path, err)
		}
		fmt.Printf("synced credential issuer %s\n", issuer.Name)
	}
	return nil
}

func syncFCAFCredentialIssuer(
	ctx context.Context,
	token, orgID string,
	issuer fcafIssuerImport,
) error {
	lookup := issuer.CanonifiedName
	if lookup == "" {
		lookup = issuer.Name
	}
	records, err := fcafListRecords(ctx, token, "credential_issuers",
		fmt.Sprintf(
			`owner="%s" && canonified_name="%s"`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(lookup),
		),
	)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"name":            issuer.Name,
		"canonified_name": issuer.CanonifiedName,
		"description":     issuer.Description,
		"url":             issuer.URL,
		"homepage_url":    issuer.HomepageURL,
		"repo_url":        issuer.RepoURL,
		"logo_url":        issuer.LogoURL,
		"published":       issuer.Published,
		"imported":        issuer.Imported,
	}
	switch len(records) {
	case 0:
		updates["owner"] = orgID
		return fcafCreateRecord(ctx, token, "credential_issuers", updates)
	case 1:
		id, ok := records[0]["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("credential issuer %q has no record id", lookup)
		}
		return fcafPatchRecord(ctx, token, "credential_issuers", id, updates)
	default:
		return fmt.Errorf(
			"expected one credential issuer named %q, found %d",
			lookup,
			len(records),
		)
	}
}

// resolveCredentialIssuerID resolves an issuer record id by canonified name.
func resolveCredentialIssuerID(
	ctx context.Context,
	token, orgID, canonifiedName string,
) (string, error) {
	if canonifiedName == "" {
		return "", nil
	}
	records, err := fcafListRecords(ctx, token, "credential_issuers",
		fmt.Sprintf(
			`owner="%s" && canonified_name="%s"`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(canonifiedName),
		),
	)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", nil
	}
	id, ok := records[0]["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("credential issuer %q has no record id", canonifiedName)
	}
	return id, nil
}

func syncFCAFCredentials(ctx context.Context, token, orgID string) error {
	paths, err := walkFCAFImports(fcafImportsDir, "/credentials/")
	if err != nil {
		return fmt.Errorf("walk FCAF credentials: %w", err)
	}
	for _, path := range paths {
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
		issuerID, resolveErr := resolveCredentialIssuerID(
			ctx,
			token,
			orgID,
			filepath.Base(filepath.Dir(path)),
		)
		if resolveErr != nil {
			return fmt.Errorf("resolve issuer for credential %s: %w", path, resolveErr)
		}
		if err := syncFCAFCredential(ctx, token, orgID, issuerID, definition); err != nil {
			return fmt.Errorf("sync credential %s: %w", path, err)
		}
		fmt.Printf("synced credential %s\n", definition.Name)
	}
	return nil
}

func syncFCAFCredential(
	ctx context.Context,
	token, orgID, issuerID string,
	definition fcafCredentialImport,
) error {
	records, err := fcafListRecords(ctx, token, "credentials",
		fmt.Sprintf(
			`owner="%s" && name="%s"`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(definition.Name),
		),
	)
	if err != nil {
		return err
	}
	updates := map[string]any{
		"name":            definition.Name,
		"description":     definition.Description,
		"yaml":            definition.YAML,
		"canonified_name": definition.CanonifiedName,
		"logo_url":        definition.LogoURL,
		"published":       definition.Published,
		"imported":        definition.Imported,
		"conformant":      definition.Conformant,
	}
	if issuerID != "" {
		updates["credential_issuer"] = issuerID
	}
	var code fcafCredentialCode
	if err := yaml.Unmarshal([]byte(definition.YAML), &code); err != nil {
		return fmt.Errorf("parse credential code: %w", err)
	}
	if strings.Contains(code.Env.Host, "capture-wallet") &&
		code.Env.CredentialConfigurationID != "" {
		deeplink, offerErr := regenerateCredentialOffer(ctx, code)
		if offerErr != nil {
			return offerErr
		}
		updates["deeplink"] = deeplink
	}
	switch len(records) {
	case 0:
		updates["owner"] = orgID
		return fcafCreateRecord(ctx, token, "credentials", updates)
	case 1:
		id, ok := records[0]["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("credential %q has no record id", definition.Name)
		}
		return fcafPatchRecord(ctx, token, "credentials", id, updates)
	default:
		return fmt.Errorf(
			"expected one credential record named %q, found %d",
			definition.Name,
			len(records),
		)
	}
}

func regenerateCredentialOffer(ctx context.Context, code fcafCredentialCode) (string, error) {
	offerBody, err := json.Marshal(
		map[string]string{"credential_configuration_id": code.Env.CredentialConfigurationID},
	)
	if err != nil {
		return "", err
	}
	offerReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		code.Env.Host,
		strings.NewReader(string(offerBody)),
	)
	if err != nil {
		return "", err
	}
	offerReq.Header.Set("Content-Type", "application/json")
	offerResp, err := http.DefaultClient.Do(offerReq)
	if err != nil {
		return "", fmt.Errorf("regenerate credential offer: %w", err)
	}
	defer offerResp.Body.Close()
	if offerResp.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(offerResp.Body)
		return "", fmt.Errorf(
			"regenerate credential offer failed: HTTP %d: %s",
			offerResp.StatusCode,
			responseBody,
		)
	}
	var offer struct {
		Deeplink string `json:"deeplink"`
	}
	if err := json.NewDecoder(offerResp.Body).Decode(&offer); err != nil {
		return "", err
	}
	if offer.Deeplink == "" {
		return "", fmt.Errorf(
			"regenerated credential offer for %q has no deeplink",
			code.Env.CredentialConfigurationID,
		)
	}
	return offer.Deeplink, nil
}

func syncFCAFWalletActions(ctx context.Context, token, orgID string) error {
	manifestPath, err := findWalletActionsManifest(fcafImportsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	input, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read wallet actions manifest: %w", err)
	}
	var manifest fcafWalletActionsManifest
	if err := yaml.Unmarshal(input, &manifest); err != nil {
		return fmt.Errorf("parse wallet actions manifest: %w", err)
	}
	if manifest.Wallet == "" {
		return fmt.Errorf("wallet actions manifest must define wallet")
	}
	walletID, err := resolveWalletID(ctx, token, orgID, manifest.Wallet)
	if err != nil {
		return err
	}
	bundleRoot := filepath.Dir(manifestPath)
	for _, action := range manifest.Actions {
		if action.Name == "" {
			continue
		}
		code, readErr := os.ReadFile(filepath.Join(bundleRoot, action.File))
		if readErr != nil {
			return fmt.Errorf("read wallet action %s: %w", action.Name, readErr)
		}
		if err := syncFCAFWalletAction(
			ctx,
			token,
			orgID,
			walletID,
			action,
			string(code),
		); err != nil {
			return err
		}
		fmt.Printf("synced wallet action %s\n", action.Name)
	}
	return nil
}

func findWalletActionsManifest(root string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(root, "*", "wallet-actions.yaml"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", os.ErrNotExist
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("found %d wallet-actions manifests, expected one", len(matches))
	}
	return matches[0], nil
}

func resolveWalletID(ctx context.Context, token, orgID, walletName string) (string, error) {
	records, err := fcafListRecords(ctx, token, "wallets",
		fmt.Sprintf(
			`owner="%s" && (canonified_name="%s" || name="%s")`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(walletName),
			pocketBaseFilterString(walletName),
		),
	)
	if err != nil {
		return "", err
	}
	if len(records) == 0 {
		return "", fmt.Errorf(
			"wallet %q not found in the target organization; register the wallet or runner first",
			walletName,
		)
	}
	id, ok := records[0]["id"].(string)
	if !ok || id == "" {
		return "", fmt.Errorf("wallet %q has no record id", walletName)
	}
	return id, nil
}

func syncFCAFWalletAction(
	ctx context.Context,
	token, orgID, walletID string,
	action fcafWalletActionRef,
	code string,
) error {
	records, err := fcafListRecords(ctx, token, "wallet_actions",
		fmt.Sprintf(
			`owner="%s" && name="%s"`,
			pocketBaseFilterString(orgID),
			pocketBaseFilterString(action.Name),
		),
	)
	if err != nil {
		return err
	}
	category := action.Category
	if category == "" {
		category = "other"
	}
	updates := map[string]any{
		"name":     action.Name,
		"code":     code,
		"category": category,
	}
	if action.Tags != "" {
		updates["tags"] = action.Tags
	}
	switch len(records) {
	case 0:
		updates["owner"] = orgID
		updates["wallet"] = walletID
		return fcafCreateRecord(ctx, token, "wallet_actions", updates)
	case 1:
		id, ok := records[0]["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("wallet action %q has no record id", action.Name)
		}
		return fcafPatchRecord(ctx, token, "wallet_actions", id, updates)
	default:
		return fmt.Errorf(
			"expected one wallet action named %q, found %d",
			action.Name,
			len(records),
		)
	}
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
	input = rewriteFCAFOrganization(input, org)
	if fcafDeviceID != "" {
		input, err = setFCAFDeviceID(input, fcafDeviceID)
		if err != nil {
			return run, fmt.Errorf("set device ID in %s: %w", path, err)
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
		query.Set("device_ids", strings.Join(queued.DeviceIDs, ","))
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

func setFCAFDeviceID(data []byte, deviceID string) ([]byte, error) {
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
	global := find(runtime, "global_device_id")
	if global == nil {
		runtime.Content = append(runtime.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "global_device_id"},
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: deviceID},
		)
	} else {
		global.Kind = yaml.ScalarNode
		global.Tag = "!!str"
		global.Value = deviceID
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
