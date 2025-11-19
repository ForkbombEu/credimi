// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package activities is a package that provides activities for the workflow engine.
// This file includes activities for checking credentials issuer metadata and running automation workflows.
package activities

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
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

type CheckCredentialsIssuerActivityPayload struct {
	BaseURL string `json:"base_url" yaml:"base_url" validate:"required"`
}

func NewCheckCredentialsIssuerActivity() *CheckCredentialsIssuerActivity {
	return &CheckCredentialsIssuerActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Parse the Credential issuer metadata (.well-known/openid-federation or .well-known/openid-credential-issuer)",
		},
	}
}

// Name returns the name of the CheckCredentialsIssuerActivity, which describes
// the purpose of this activity as checking the credential issuer metadata.
func (a *CheckCredentialsIssuerActivity) Name() string {
	return a.BaseActivity.Name
}

type FederationDecodingError struct {
	Payload string
	Code    string
	Err     error
}

func (e *FederationDecodingError) Error() string {
	return fmt.Sprintf("failed to decode openid-federation JWT payload: %v", e.Err)
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

	payload, err := workflowengine.DecodePayload[CheckCredentialsIssuerActivityPayload](
		input.Payload,
	)
	if err != nil {
		return result, a.NewMissingOrInvalidPayloadError(err)
	}

	cleanURL := TrimInput(payload.BaseURL)
	if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
		cleanURL = "https://" + cleanURL
	}
	cleanURL = strings.TrimRight(cleanURL, "/")
	if !strings.HasSuffix(cleanURL, "/.well-known/openid-credential-issuer") {
		// 1. Try federation
		federationURL := strings.TrimSuffix(
			cleanURL,
			"/.well-known/openid-federation",
		) + "/.well-known/openid-federation"
		federationJSON, err := fetchJSONFromURL(ctx, federationURL, true, a)
		if err == nil {
			return workflowengine.ActivityResult{
				Output: map[string]any{
					"rawJSON":  federationJSON,
					"base_url": payload.BaseURL,
					"source":   ".well-known/openid-federation",
				},
			}, nil
		}
		var decodeErr *FederationDecodingError
		if errors.As(err, &decodeErr) {
			return result, a.NewActivityError(
				errorcodes.Codes[errorcodes.DecodeFailed].Code,
				fmt.Sprintf(
					"Openid-federation well.known exists but JWT decoding failed: %v",
					decodeErr.Err,
				),
				decodeErr.Payload,
			)
		}
	}
	// 2. Fallback to credential issuer
	issuerURL := strings.TrimSuffix(
		cleanURL,
		"/.well-known/openid-credential-issuer",
	) + "/.well-known/openid-credential-issuer"
	issuerJSON, err := fetchJSONFromURL(ctx, issuerURL, false, a)
	if err != nil {
		return result, err
	}

	return workflowengine.ActivityResult{
		Output: map[string]any{
			"rawJSON":  issuerJSON,
			"base_url": payload.BaseURL,
			"source":   ".well-known/openid-credential-issuer",
		},
	}, nil
}

func fetchJSONFromURL(
	ctx context.Context,
	url string,
	isJWT bool,
	a *CheckCredentialsIssuerActivity,
) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.CreateHTTPRequestFailed]
		return "", a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			"GET",
			url,
		)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ExecuteHTTPRequestFailed]
		return "", a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			req,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errCode := errorcodes.Codes[errorcodes.IsNotCredentialIssuer]
		return "", a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: ", errCode.Description),
			url,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.ReadFromReaderFailed]
		return "", a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			resp.Body,
		)
	}

	if isJWT {
		parts := strings.Split(string(body), ".")
		if len(parts) < 2 {
			errCode := errorcodes.Codes[errorcodes.InvalidJWTFormat]
			return "", &FederationDecodingError{
				Payload: string(body),
				Code:    errCode.Code,
				Err:     errors.New(errCode.Description),
			}
		}

		payload := parts[1]
		decoded, err := base64.RawURLEncoding.DecodeString(payload)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.DecodeFailed]
			return "", &FederationDecodingError{
				Payload: payload,
				Code:    errCode.Code,
				Err:     err,
			}
		}
		// Unmarshal and extract openid_credential_issuer
		var data map[string]any
		if err := json.Unmarshal(decoded, &data); err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
			return "", &FederationDecodingError{
				Payload: string(decoded),
				Code:    errCode.Code,
				Err:     err,
			}
		}

		issuerMap, ok := data["metadata"].(map[string]any)["openid_credential_issuer"].(map[string]any)
		if !ok {
			errCode := errorcodes.Codes[errorcodes.InvalidJWTFormat]
			return "", &FederationDecodingError{
				Payload: string(decoded),
				Code:    errCode.Code,
				Err:     fmt.Errorf("openid_credential_issuer not found or not valid"),
			}
		}
		jsonBytes, err := json.Marshal(issuerMap)
		if err != nil {
			errCode := errorcodes.Codes[errorcodes.JSONMarshalFailed]
			return "", &FederationDecodingError{
				Payload: string(decoded),
				Code:    errCode.Code,
				Err:     err,
			}
		}
		return string(jsonBytes), nil
	}

	return string(body), nil
}
