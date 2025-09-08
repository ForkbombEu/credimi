// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	yamlPath string
	apiKey   string
)

// NewPipelineCmd creates the "pipeline" command, using the PocketBase URL.
func NewPipelineCmd(pbURL string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Start a pipeline workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			if pbURL == "" {
				return fmt.Errorf("PocketBase URL not provided")
			}

			// --- authenticate (if apiKey provided) ---
			var bearerToken string
			if apiKey != "" {
				authReq, err := http.NewRequest("GET", fmt.Sprintf("%s/api/apikey/authenticate", pbURL), nil)
				if err != nil {
					return fmt.Errorf("failed to create auth request: %w", err)
				}
				authReq.Header.Set("X-Api-Key", apiKey)

				authResp, err := http.DefaultClient.Do(authReq)
				if err != nil {
					return fmt.Errorf("failed to call authenticate endpoint: %w", err)
				}
				defer authResp.Body.Close()

				if authResp.StatusCode != http.StatusOK {
					body, _ := io.ReadAll(authResp.Body)
					return fmt.Errorf("authentication failed: %s", string(body))
				}

				var authData struct {
					Message string `json:"message"`
					Token   string `json:"token"`
				}
				if err := json.NewDecoder(authResp.Body).Decode(&authData); err != nil {
					return fmt.Errorf("failed to decode authentication response: %w", err)
				}
				bearerToken = authData.Token
			}

			// --- read YAML ---
			var yamlData []byte
			var err error
			if yamlPath != "" {
				yamlData, err = os.ReadFile(yamlPath)
				if err != nil {
					return fmt.Errorf("failed to read yaml file: %w", err)
				}
			} else {
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("no YAML provided: use -p <file> or pipe data into stdin")
				}
				yamlData, err = io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("failed to read stdin: %w", err)
				}
			}

			payload := map[string]any{"yaml": string(yamlData)}

			body, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/pipeline/start", pbURL), bytes.NewBuffer(body))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			if bearerToken != "" {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", bearerToken))
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			fmt.Println("Response:", string(respBody))
			return nil
		},
	}

	// flags
	cmd.Flags().StringVarP(&yamlPath, "path", "p", "", "Path to YAML file (optional, otherwise reads from stdin)")
	cmd.Flags().StringVarP(&apiKey, "api-key", "k", "", "API key for authentication")

	return cmd
}
