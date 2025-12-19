// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert/yaml"
)

var (
	yamlPath    string
	apiKey      string
	instanceURL string
)

// NewPipelineCmd creates the "pipeline" command, using the PocketBase URL.
func NewPipelineCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Start a pipeline workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := authenticate(cmd.Context())
			if err != nil {
				return err
			}

			orgID, canonName, err := getMyOrganization(cmd.Context(), token)
			if err != nil {
				return err
			}

			input, err := readPipelineInput()
			if err != nil {
				return err
			}

			rec, err := findOrCreatePipeline(app, orgID, input)
			if err != nil {
				return err
			}

			return startPipeline(cmd.Context(), token, canonName, rec)
		},
	}

	addPipelineFlags(cmd)
	cmd.AddCommand(NewSchemaCmd())
	cmd.AddCommand(NewPipelineStoreCmd(app))
	return cmd
}

// NewPipelineStoreCmd creates the "pipeline store" subcommand
func NewPipelineStoreCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Store a pipeline in the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := authenticate(cmd.Context())
			if err != nil {
				return err
			}

			orgID, _, err := getMyOrganization(cmd.Context(), token)
			if err != nil {
				return err
			}

			input, err := readPipelineInput()
			if err != nil {
				return err
			}

			rec, err := createPipeline(app, orgID, input)
			if err != nil {
				return err
			}

			return printJSON(rec.FieldsData())
		},
	}

	addPipelineFlags(cmd)
	return cmd
}

func addPipelineFlags(cmd *cobra.Command) {
	cmd.Flags().
		StringVarP(&yamlPath, "path", "p", "", "Path to YAML file (optional, otherwise reads from stdin)")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")
	cmd.MarkFlagRequired("api-key")
	cmd.Flags().
		StringVarP(&instanceURL, "instance", "i", "https://credimi.io", "URL of the PocketBase instance")
}

func authenticate(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		utils.JoinURL(instanceURL, "api", "apikey", "authenticate"),
		nil,
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Api-Key", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: %s", b)
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", err)
	}

	return out.Token, nil
}

func getMyOrganization(
	ctx context.Context,
	token string,
) (orgID string, canonName string, err error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		utils.JoinURL(instanceURL, "api", "organizations", "my"),
		nil,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create org request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to call get organization endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get organization: %s", string(body))
	}

	var orgData struct {
		Id             string `json:"id"`
		CanonifiedName string `json:"canonified_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&orgData); err != nil {
		return "", "", fmt.Errorf("failed to decode org response: %w", err)
	}

	return orgData.Id, orgData.CanonifiedName, nil
}

type WorkflowYAML struct {
	Name string `yaml:"name"`
}

func parsePipelineName(yamlBytes []byte) (string, error) {
	var wf WorkflowYAML
	if err := yaml.Unmarshal(yamlBytes, &wf); err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}
	return wf.Name, nil
}

type PipelineCLIInput struct {
	Name string
	YAML string
}

// Checks for existing pipeline, otherwise creates
func findOrCreatePipeline(
	app core.App,
	orgID string,
	input *PipelineCLIInput,
) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("pipelines")
	if err != nil {
		return nil, err
	}

	filter := fmt.Sprintf(`owner="%s" && name="%s" && yaml="%s"`, orgID, input.Name, input.YAML)
	rec, err := app.FindFirstRecordByFilter("pipelines", filter)
	if err == nil {
		return rec, nil
	}

	// Create new if not found
	rec = core.NewRecord(col)
	rec.Set("owner", orgID)
	rec.Set("name", input.Name)
	rec.Set("description", "Pipeline started from CLI")
	rec.Set("yaml", input.YAML)
	rec.Set("steps", "[{}]") // empty steps

	return rec, app.Save(rec)
}

// Always creates new record
func createPipeline(app core.App, orgID string, input *PipelineCLIInput) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("pipelines")
	if err != nil {
		return nil, err
	}

	rec := core.NewRecord(col)
	rec.Set("owner", orgID)
	rec.Set("name", input.Name)
	rec.Set("description", "Pipeline stored from CLI")
	rec.Set("yaml", input.YAML)
	rec.Set("steps", "[{}]") // empty steps

	return rec, app.Save(rec)
}

func readPipelineInput() (*PipelineCLIInput, error) {
	var yamlData []byte
	var err error

	if yamlPath != "" {
		yamlData, err = os.ReadFile(yamlPath)
	} else {
		yamlData, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return nil, err
	}

	name, err := parsePipelineName(yamlData)
	if err != nil {
		return nil, err
	}

	return &PipelineCLIInput{
		Name: name,
		YAML: string(yamlData),
	}, nil
}

func startPipeline(ctx context.Context, token string, canonName string, rec *core.Record) error {
	payload := map[string]any{
		"yaml":                rec.GetString("yaml"),
		"pipeline_identifier": fmt.Sprintf("%s/%s", canonName, rec.GetString("canonified_name")),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal start payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		utils.JoinURL(instanceURL, "api", "pipeline", "start"),
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create start request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call pipeline start endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var respJSON any
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		respJSON = map[string]any{"raw": string(respBody)}
	}

	return printJSON(map[string]any{
		"status":  resp.StatusCode,
		"payload": respJSON,
	})
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output as JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
