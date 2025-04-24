// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

const FidesIssuersUrl = "https://credential-catalog.fides.community/api/public/credentialtype?includeAllDetails=false&size=200"

type FetchIssuersActivityResponse struct{ Issuers []string }

type FidesResponse struct {
	Content []struct {
		IssuanceURL               string `json:"issuanceUrl"`
		CredentialConfigurationID string `json:"credentialConfigurationId"`
		IssuePortalURL            string `json:"issuePortalUrl,omitempty"`
	} `json:"content"`
	Page struct {
		Size          int `json:"size"`
		Number        int `json:"number"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
	} `json:"page"`
}

type CredentialWorkflowInput struct {
	BaseURL  string // Base URL for the credential issuer
	IssuerID string // ID of the credentials issuer from PB
}

type CredentialWorkflowResponse struct {
	Message string
}

type CreateCredentialIssuersInput struct {
	Issuers []string
	DBPath  string
}

const FetchIssuersTaskQueue = "FetchIssuersTaskQueue"
