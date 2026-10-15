// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"fmt"
	"strings"
)

// issuanceCompletedStatus is the status Capture records on an issuer session
// once the Wallet has fetched the credential from the Credential Endpoint.
const issuanceCompletedStatus = "credential_issued"

// OID4VCIFixturesStoredValidator proves that the Wallet holds a credential for
// each named claim-set fixture without presenting it: presenting would spend
// it, because the reference Wallet uses each instance once.
//
// Each evidence entry pairs the issuer session that issued a fixture with the
// screenshots of the wallet step that accepted the offer. The issuer session
// proves the Wallet fetched the credential; the named screenshot, which the
// issuance flow takes only after the Wallet confirms the document was added,
// proves the Wallet stored it rather than discarding it after the fetch.
type OID4VCIFixturesStoredValidator struct{}

// ID returns the validator identifier.
func (OID4VCIFixturesStoredValidator) ID() string {
	return "oid4vci.fixtures_stored"
}

type fixtureStorageOutcome struct {
	status  Status
	message string
}

// Validate requires, for every fixture in params.fixtures, an entry whose
// issuer_session carries that fixture_id with status credential_issued and
// whose wallet_screenshots hold the params.stored_screenshot screenshot.
func (OID4VCIFixturesStoredValidator) Validate(ctx context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Fixtures         []string `json:"fixtures"`
		StoredScreenshot string   `json:"stored_screenshot"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Fixtures) == 0 {
		return Result{Status: StatusError, Message: "fixtures param is required"}
	}
	if strings.TrimSpace(params.StoredScreenshot) == "" {
		return Result{Status: StatusError, Message: "stored_screenshot param is required"}
	}

	entries, ok := input.Value.([]any)
	if !ok {
		entries = []any{input.Value}
	}
	best := map[string]fixtureStorageOutcome{}
	for index, rawEntry := range entries {
		entry, ok := normalizeJSONObject(rawEntry)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("fixture evidence %d is not an object", index),
			}
		}
		session, ok := normalizeJSONObject(entry["issuer_session"])
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("fixture evidence %d has no issuer_session object", index),
			}
		}
		fixture, _ := session["fixture_id"].(string)
		outcome := fixtureStorage(
			ctx,
			fixture,
			session,
			entry["wallet_screenshots"],
			params.StoredScreenshot,
		)
		if previous, seen := best[fixture]; !seen ||
			storageRank(outcome.status) > storageRank(previous.status) {
			best[fixture] = outcome
		}
	}

	for _, fixture := range params.Fixtures {
		outcome, seen := best[fixture]
		if !seen {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("no issuer session issued fixture %q", fixture),
			}
		}
		if outcome.status != StatusPass {
			return Result{Status: outcome.status, Message: outcome.message}
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"the Wallet fetched and stored a credential for each fixture: %s",
			strings.Join(params.Fixtures, ", "),
		),
	}
}

// fixtureStorage judges one issuance of a fixture.
func fixtureStorage(
	ctx context.Context,
	fixture string,
	session map[string]any,
	screenshots any,
	storedScreenshot string,
) fixtureStorageOutcome {
	if status, _ := session["status"].(string); status != issuanceCompletedStatus {
		return fixtureStorageOutcome{
			status: StatusFail,
			message: fmt.Sprintf(
				"fixture %q was not issued: its issuer session status is %q, expected %s",
				fixture,
				status,
				issuanceCompletedStatus,
			),
		}
	}
	stored := EvidenceScreenshotNamedValidator{}.Validate(ctx, Input{
		Value:  screenshots,
		Params: map[string]any{"name": storedScreenshot},
	})
	if stored.Status == StatusPass {
		return fixtureStorageOutcome{status: StatusPass}
	}
	return fixtureStorageOutcome{
		status: stored.Status,
		message: fmt.Sprintf(
			"fixture %q was fetched from the issuer, but the Wallet did not confirm storing it: %s",
			fixture,
			stored.Message,
		),
	}
}

// storageRank orders outcomes so a fixture issued several times is judged by
// its best issuance.
func storageRank(status Status) int {
	switch status {
	case StatusPass:
		return 3
	case StatusBlocked:
		return 2
	case StatusFail:
		return 1
	default:
		return 0
	}
}
