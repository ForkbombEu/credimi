// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

// FidesIssuersURL is the URL to fetch issuers from the Fides API.
const FidesIssuersURL = "https://credential-catalog.fides.community/api/public/credentialtype?" + params
const params = "includeAllDetails=false&size=200"

// FetchIssuersActivityResponse represents the response containing a list of issuers fetched from the Fides API.
type FetchIssuersActivityResponse struct{ Issuers []string }

// FidesResponse represents the structure of the response from the Fides API.
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

// CreateCredentialIssuersInput represents the input required to create credential issuers.
type CreateCredentialIssuersInput struct {
	Issuers []string
	DBPath  string
}

// FetchIssuersTaskQueue is the task queue name for fetching issuers.
const FetchIssuersTaskQueue = "FetchIssuersTaskQueue"
