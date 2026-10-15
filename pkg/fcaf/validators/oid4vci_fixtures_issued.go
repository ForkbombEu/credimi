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

// OID4VCIFixturesIssuedValidator proves that the Wallet received a credential
// for each named claim-set fixture, from the issuer sessions that issued them.
// It is the possession evidence for sources whose precondition is a Wallet
// holding several credentials: proving possession by presenting a credential
// would spend it, because the reference Wallet uses each instance once.
type OID4VCIFixturesIssuedValidator struct{}

// ID returns the validator identifier.
func (OID4VCIFixturesIssuedValidator) ID() string {
	return "oid4vci.fixtures_issued"
}

// Validate requires, for every fixture in params.fixtures, an issuer session
// record carrying that fixture_id whose status is credential_issued.
func (OID4VCIFixturesIssuedValidator) Validate(_ context.Context, input Input) Result {
	params, err := DecodeParams[struct {
		Fixtures []string `json:"fixtures"`
	}](input.Params)
	if err != nil {
		return Result{Status: StatusError, Message: err.Error()}
	}
	if len(params.Fixtures) == 0 {
		return Result{Status: StatusError, Message: "fixtures param is required"}
	}

	records, ok := input.Value.([]any)
	if !ok {
		records = []any{input.Value}
	}
	issued := map[string]bool{}
	statuses := map[string][]string{}
	for index, entry := range records {
		record, ok := normalizeJSONObject(entry)
		if !ok {
			return Result{
				Status:  StatusFail,
				Message: fmt.Sprintf("issuer session %d is not an object", index),
			}
		}
		fixture, _ := record["fixture_id"].(string)
		status, _ := record["status"].(string)
		statuses[fixture] = append(statuses[fixture], status)
		if status == issuanceCompletedStatus {
			issued[fixture] = true
		}
	}

	for _, fixture := range params.Fixtures {
		if issued[fixture] {
			continue
		}
		if seen := statuses[fixture]; len(seen) > 0 {
			return Result{
				Status: StatusFail,
				Message: fmt.Sprintf(
					"fixture %q was not issued: its issuer session status is %s, expected %s",
					fixture,
					strings.Join(seen, ", "),
					issuanceCompletedStatus,
				),
			}
		}
		return Result{
			Status:  StatusFail,
			Message: fmt.Sprintf("no issuer session issued fixture %q", fixture),
		}
	}
	return Result{
		Status: StatusPass,
		Message: fmt.Sprintf(
			"the Wallet received a credential for each fixture: %s",
			strings.Join(params.Fixtures, ", "),
		),
	}
}
