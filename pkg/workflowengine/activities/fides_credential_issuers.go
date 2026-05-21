// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/forkbombeu/credimi/pkg/workflowengine"
)

const (
	FidesCredentialIssuersURL = "https://fides-issuer-catalog.vercel.app/api/public/issuer"
)

type ParseFidesCredentialIssuersActivity struct {
	workflowengine.BaseActivity
}

type ParseFidesCredentialIssuersActivityPayload struct {
	Data any `json:"data" yaml:"data" validate:"required"`
}

type ParseFidesCredentialIssuersActivityResponse struct {
	Issuers    []string `json:"issuers"`
	PageNumber int      `json:"page_number"`
	TotalPages int      `json:"total_pages"`
}

type fidesCredentialIssuersResponse struct {
	Content []struct {
		IssuanceProtocol    string `json:"issuanceProtocol"`
		CredentialIssuerURL string `json:"credentialIssuerUrl"`
		OID4VCIMetadataURL  string `json:"oid4vciMetadataUrl"`
	} `json:"content"`
	Size          int `json:"size"`
	Number        int `json:"number"`
	TotalElements int `json:"totalElements"`
	TotalPages    int `json:"totalPages"`
}

func NewParseFidesCredentialIssuersActivity() *ParseFidesCredentialIssuersActivity {
	return &ParseFidesCredentialIssuersActivity{
		BaseActivity: workflowengine.BaseActivity{
			Name: "Parse Fides credential issuer catalog response",
		},
	}
}

func (a *ParseFidesCredentialIssuersActivity) Name() string {
	return a.BaseActivity.Name
}

func (a *ParseFidesCredentialIssuersActivity) Execute(
	ctx context.Context,
	input workflowengine.ActivityInput,
) (workflowengine.ActivityResult, error) {
	payload, err := workflowengine.DecodePayload[ParseFidesCredentialIssuersActivityPayload](
		input.Payload,
	)
	if err != nil {
		return workflowengine.ActivityResult{}, a.NewMissingOrInvalidPayloadError(err)
	}

	parsed, err := parseFidesCredentialIssuersResponse(payload.Data)
	if err != nil {
		errCode := errorcodes.Codes[errorcodes.JSONUnmarshalFailed]
		return workflowengine.ActivityResult{}, a.NewActivityError(
			errCode.Code,
			fmt.Sprintf("%s: %v", errCode.Description, err),
			payload.Data,
		)
	}

	return workflowengine.ActivityResult{
		Output: ParseFidesCredentialIssuersActivityResponse{
			Issuers:    extractCredentialIssuerIdentifiers(parsed),
			PageNumber: parsed.Number,
			TotalPages: parsed.TotalPages,
		},
	}, nil
}

func parseFidesCredentialIssuersResponse(data any) (fidesCredentialIssuersResponse, error) {
	var parsed fidesCredentialIssuersResponse
	raw, err := json.Marshal(data)
	if err != nil {
		return parsed, err
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return parsed, err
	}
	return parsed, nil
}

func extractCredentialIssuerIdentifiers(root fidesCredentialIssuersResponse) []string {
	issuers := make([]string, 0, len(root.Content))
	for _, item := range root.Content {
		issuerURL := item.OID4VCIMetadataURL
		if issuerURL == "" && item.IssuanceProtocol == "oid4vci" {
			issuerURL = item.CredentialIssuerURL
		}
		if issuerURL == "" {
			continue
		}
		issuers = append(issuers, credentialIssuerIdentifierFromMetadataURL(issuerURL))
	}
	return issuers
}

func credentialIssuerIdentifierFromMetadataURL(rawURL string) string {
	const wellKnownPath = "/.well-known/openid-credential-issuer"

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawURL
	}

	switch {
	case parsed.Path == wellKnownPath:
		parsed.Path = ""
	case strings.HasPrefix(parsed.Path, wellKnownPath+"/"):
		parsed.Path = strings.TrimPrefix(parsed.Path, wellKnownPath)
	case strings.HasSuffix(parsed.Path, wellKnownPath):
		parsed.Path = strings.TrimSuffix(parsed.Path, wellKnownPath)
	default:
		return rawURL
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}
