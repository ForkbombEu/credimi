// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package activities is a package that provides activities for the workflow engine.
// This file includes activities for checking credentials issuer metadata and running automation workflows.
package activities

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

// Credential is a struct that represents the credential issuer metadata
// as defined in the OpenID4VP specification. It includes various fields
// such as credential definition, supported signing algorithms, cryptographic
// binding methods, display options, format, proof types, and scope.

// CheckCredentialsIssuerActivity is an activity that checks the credential issuer
type CheckCredentialsIssuerActivity struct {
	workflowengine.BaseActivity
}

func NewCheckCredentialsIssuerActivity() *CheckCredentialsIssuerActivity {
	return &CheckCredentialsIssuerActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Parse the Credential issuer metadata (.well-known/openid-credential-issuer)",
		},
	}
}

// Name returns the name of the CheckCredentialsIssuerActivity, which describes
// the purpose of this activity as checking the credential issuer metadata.
func (a *CheckCredentialsIssuerActivity) Name() string {
	return "Parse the Credential issuer metadata (.well-known/openid-credential-issuer)"
}

// Execute performs the CheckCredentialsIssuerActivity by validating the provided
// base URL from the input configuration, constructing the issuer URL, and making
// an HTTP GET request to verify if the endpoint is a valid credential issuer.
//
// Parameters:
//   - ctx: The context for managing request deadlines and cancellations.
//   - input: The ActivityInput containing the configuration map with a "base_url" key.
//
// Returns:
//   - ActivityResult: Contains the raw JSON response from the credential issuer
//     and the validated base URL if successful.
//   - error: An error if the activity fails due to missing configuration, invalid
//     base URL, HTTP request issues, or non-OK response status.
//
// The function ensures the base URL is properly formatted with a scheme (http/https),
// appends the OpenID credential issuer path, and validates the response. If any
// step fails, it returns a failure result with an appropriate error message.
func (a *CheckCredentialsIssuerActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	result := workflowengine.ActivityResult{}

	baseURL, ok := input.Config["base_url"]
	if !ok || strings.TrimSpace(baseURL) == "" {
		errCode := errorcodes.Codes[errorcodes.MissingOrInvalidConfig]
		if !ok {
			return result, a.NewActivityError(
				errCode.Code,
				fmt.Sprintf("%s: 'baseURL'", errCode.Description),
			)
		}
	}
	cleanURL := strings.TrimSpace(baseURL)
	if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
		cleanURL = "https://" + cleanURL
	}

	issuerURL := strings.TrimRight(cleanURL, "/")
	if !strings.HasSuffix(issuerURL, "/.well-known/openid-credential-issuer") {
		issuerURL += "/.well-known/openid-credential-issuer"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", issuerURL, nil)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CreateHTTPRequestFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			"GET",
			issuerURL,
		)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			req,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errCode := errorcodes.Codes[errorcodes.IsNotCredentialIssuer]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: ", errCode.Description),
			issuerURL,
		)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ReadFromReaderFailed]
		return result, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.Body,
		)
	}

	return workflowengine.ActivityResult{
		Output: map[string]any{
			"rawJSON":  string(bodyBytes),
			"base_url": baseURL,
		},
	}, nil
}
