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
	"net/url"
	"os"

	"github.com/forkbombeu/credimi/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert/yaml"
)

var (
	yamlPath    string
	apiKey      string
	instanceURL string
)

// NewPipelineCmd creates the "pipeline" command, using the PocketBase URL.
func NewPipelineCmd() *cobra.Command {
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

			rec, err := findOrCreatePipeline(cmd.Context(), token, orgID, input)
			if err != nil {
				return err
			}

			return startPipeline(cmd.Context(), token, canonName, rec)
		},
	}

	addPipelineFlags(cmd)
	cmd.AddCommand(NewSchemaCmd())
	cmd.AddCommand(NewPipelineStoreCmd())
	return cmd
}

// NewPipelineStoreCmd creates the "pipeline store" subcommand
func NewPipelineStoreCmd() *cobra.Command {
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

			rec, err := createPipeline(cmd.Context(), token, orgID, input)
			if err != nil {
				return err
			}

			return printJSON(rec)
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

type pipelineQueueResponse struct {
	Mode              string   `json:"mode"`
	TicketID          string   `json:"ticket_id"`
	RunnerIDs         []string `json:"runner_ids"`
	Position          int      `json:"position"`
	LineLen           int      `json:"line_len"`
	WorkflowID        string   `json:"workflow_id"`
	RunID             string   `json:"run_id"`
	WorkflowNamespace string   `json:"workflow_namespace"`
	ErrorMessage      string   `json:"error_message"`
}

// Checks for existing pipeline, otherwise creates
func findOrCreatePipeline(
	ctx context.Context,
	token string,
	orgID string,
	input *PipelineCLIInput,
) (map[string]any, error) {
	// 1. Try to find existing
	filter := fmt.Sprintf(
		`owner="%s" && name="%s" && yaml="%s"`,
		orgID,
		input.Name,
		input.YAML,
	)

	findURL := utils.JoinURL(
		instanceURL,
		"api",
		"collections",
		"pipelines",
		"records",
	) + "?filter=" + url.QueryEscape(filter)

	req, _ := http.NewRequestWithContext(ctx, "GET", findURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	if len(list.Items) > 0 {
		return list.Items[0], nil
	}

	// 2. Create new
	payload := map[string]any{
		"owner":       orgID,
		"name":        input.Name,
		"description": "Pipeline started from CLI",
		"yaml":        input.YAML,
		"steps":       "[{}]",
	}

	body, _ := json.Marshal(payload)

	createReq, _ := http.NewRequestWithContext(
		ctx,
		"POST",
		utils.JoinURL(
			instanceURL,
			"api",
			"collections",
			"pipelines",
			"records",
		),
		bytes.NewReader(body),
	)
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	createResp, err := http.DefaultClient.Do(createReq)
	if err != nil {
		return nil, err
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(createResp.Body)
		return nil, fmt.Errorf("create failed: %s", b)
	}

	var created map[string]any
	return created, json.NewDecoder(createResp.Body).Decode(&created)
}

// Always creates new record
func createPipeline(
	ctx context.Context,
	token string,
	orgID string,
	input *PipelineCLIInput,
) (map[string]any, error) {
	payload := map[string]any{
		"owner":       orgID,
		"name":        input.Name,
		"description": "Pipeline stored from CLI",
		"yaml":        input.YAML,
		"steps":       "[{}]",
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		utils.JoinURL(
			instanceURL,
			"api",
			"collections",
			"pipelines",
			"records",
		),
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create pipeline: %s", b)
	}

	var out map[string]any
	return out, json.NewDecoder(resp.Body).Decode(&out)
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

// postPipelineRequest sends a POST request to a pipeline endpoint and returns status/body.
func postPipelineRequest(
	ctx context.Context,
	token string,
	path string,
	payload []byte,
) (int, []byte, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		utils.JoinURL(instanceURL, "api", "pipeline", path),
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create %s request: %w", path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to call pipeline %s endpoint: %w", path, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, nil
}

// decodeJSONPayload parses JSON responses and falls back to raw strings.
func decodeJSONPayload(respBody []byte) any {
	var respJSON any
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		return map[string]any{"raw": string(respBody)}
	}
	return respJSON
}

func startPipeline(ctx context.Context, token string, canonName string, rec map[string]any) error {
	payload := map[string]any{
		"yaml":                rec["yaml"].(string),
		"pipeline_identifier": fmt.Sprintf("%s/%s", canonName, rec["canonified_name"].(string)),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal start payload: %w", err)
	}

	queueStatus, queueBody, err := postPipelineRequest(ctx, token, "queue", body)
	if err != nil {
		return err
	}
	if queueStatus != http.StatusOK {
		return printJSON(map[string]any{
			"status":  queueStatus,
			"payload": decodeJSONPayload(queueBody),
		})
	}

	var queueResp pipelineQueueResponse
	if err := json.Unmarshal(queueBody, &queueResp); err != nil {
		return fmt.Errorf("failed to decode queue response: %w", err)
	}

	switch queueResp.Mode {
	case "queued":
		return printJSON(map[string]any{
			"mode":           queueResp.Mode,
			"ticket_id":      queueResp.TicketID,
			"runner_ids":     queueResp.RunnerIDs,
			"position":       queueResp.Position,
			"line_len":       queueResp.LineLen,
			"position_human": queueResp.Position + 1,
		})
	case "started":
		return printJSON(map[string]any{
			"mode":               queueResp.Mode,
			"workflow_id":        queueResp.WorkflowID,
			"run_id":             queueResp.RunID,
			"workflow_namespace": queueResp.WorkflowNamespace,
		})
	case "failed":
		return printJSON(map[string]any{
			"mode":          queueResp.Mode,
			"error_message": queueResp.ErrorMessage,
		})
	default:
		return printJSON(map[string]any{
			"mode":    queueResp.Mode,
			"payload": decodeJSONPayload(queueBody),
		})
	}
}

func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output as JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
