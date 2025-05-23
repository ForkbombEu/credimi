// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package workflow contains activities for managing credential issuers.
package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	// Importing modernc.org/sqlite for its side effects to enable SQLite database driver.
	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/lib"
)

// FetchIssuersActivity retrieves a list of issuers by recursively fetching data from the API.
func FetchIssuersActivity(ctx context.Context) (FetchIssuersActivityResponse, error) {
	// Start with offset 0.
	hrefs, err := fetchIssuersRecursive(ctx, 0)
	if err != nil {
		return FetchIssuersActivityResponse{}, err
	}
	return FetchIssuersActivityResponse{Issuers: hrefs}, nil
}

func fetchIssuersRecursive(ctx context.Context, after int) ([]string, error) {
	var url string
	if after > 0 {
		url = fmt.Sprintf("%s&page=%d", FidesIssuersURL, after)
	} else {
		url = FidesIssuersURL
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	var root FidesResponse
	if err = json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	hrefs := extractHrefsFromAPIResponse(root)

	if root.Page.Number >= root.Page.TotalPages || len(hrefs) < 200 {
		return hrefs, nil
	}

	nextHrefs, err := fetchIssuersRecursive(ctx, after+1)
	if err != nil {
		return nil, err
	}

	return append(hrefs, nextHrefs...), nil
}

func extractHrefsFromAPIResponse(root FidesResponse) []string {
	hrefs := []string{}
	for _, item := range root.Content {
		trimmedHref := removeWellKnownSuffix(item.IssuanceURL)
		hrefs = append(hrefs, trimmedHref)
	}
	return hrefs
}

// CreateCredentialIssuersActivity inserts a list of credential issuers into the database if they do not already exist.
func CreateCredentialIssuersActivity(
	ctx context.Context,
	input CreateCredentialIssuersInput,
) error {
	db, err := sql.Open("sqlite", input.DBPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	for _, issuer := range input.Issuers {
		exists, err := checkIfCredentialIssuerExist(ctx, db, issuer)
		if err != nil {
			return fmt.Errorf("failed to check if issuer exists: %w", err)
		}
		if exists {
			continue
		}
		_, err = db.ExecContext(ctx, "INSERT INTO credential_issuers(url) VALUES (?)", issuer)
		if err != nil {
			return fmt.Errorf("failed to insert issuer into database: %w", err)
		}
	}

	return nil
}

func checkIfCredentialIssuerExist(ctx context.Context, db *sql.DB, url string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM credential_issuers WHERE url = ?", url).
		Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to query database: %w", err)
	}
	return count > 0, nil
}

func removeWellKnownSuffix(urlStr string) string {
	const suffix = "/.well-known/openid-credential-issuer"
	if strings.HasSuffix(urlStr, suffix) {
		return strings.TrimSuffix(urlStr, suffix)
	}
	return urlStr
}
