// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package activities

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/lib"

	workflowengine "github.com/forkbombeu/didimo/pkg/workflow_engine"
	"github.com/forkbombeu/didimo/pkg/workflow_engine/workflows/credentials_config"
)

type Credential struct {
	CredentialDefinition                 *credentials_config.OpenidCredentialIssuerSchemaJsonCredentialConfigurationsSupportedValueCredentialDefinition                      `json:"credential_definition,omitempty"`
	CredentialSigningAlgValuesSupported  []credentials_config.OpenidCredentialIssuerSchemaJsonCredentialConfigurationsSupportedValueCredentialSigningAlgValuesSupportedElem  `json:"credential_signing_alg_values_supported,omitempty"`
	CryptographicBindingMethodsSupported []credentials_config.OpenidCredentialIssuerSchemaJsonCredentialConfigurationsSupportedValueCryptographicBindingMethodsSupportedElem `json:"cryptographic_binding_methods_supported,omitempty"`
	Display                              []credentials_config.OpenidCredentialIssuerSchemaJsonCredentialConfigurationsSupportedValueDisplayElem                              `json:"display,omitempty"`
	Format                               string                                                                                                                              `json:"format"`
	ProofTypesSupported                  credentials_config.OpenidCredentialIssuerSchemaJsonCredentialConfigurationsSupportedValueProofTypesSupported                        `json:"proof_types_supported,omitempty"`
	Scope                                *string                                                                                                                             `json:"scope,omitempty"`
}

type CheckCredentialsIssuerActivity struct{}

func (a *CheckCredentialsIssuerActivity) Name() string {
	return "Parse the Credential issuer metadata (.well-known/openid-credential-issuer)"
}

func (a *CheckCredentialsIssuerActivity) Execute(ctx context.Context, input workflowengine.ActivityInput) (workflowengine.ActivityResult, error) {
	baseURL, ok := input.Config["base_url"]
	if !ok || strings.TrimSpace(baseURL) == "" {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, "Missing baseURL in config")
	}

	if !strings.HasPrefix(baseURL, "https://") && !strings.HasPrefix(baseURL, "http://") {
		baseURL = "https://" + baseURL
	}

	issuerURL := strings.TrimRight(baseURL, "/") + "/.well-known/openid-credential-issuer"
	req, err := http.NewRequestWithContext(ctx, "GET", issuerURL, nil)
	if err != nil {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, fmt.Sprintf("Request creation failed: %v", err))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, fmt.Sprintf("Could not reach issuer: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, fmt.Sprintf("Not a credential issuer, status: %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return workflowengine.Fail(&workflowengine.ActivityResult{}, "Error reading response from credential issuer")
	}

	return workflowengine.ActivityResult{
		Output: map[string]any{
			"rawJSON":  string(bodyBytes),
			"base_url": baseURL,
		},
	}, nil
}
