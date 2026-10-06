// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"
)

const fixtureTokenPrefix = "${fixture."

const (
	DefaultIssuerURL   = "https://issuer-backend.eudiw.dev"
	DefaultVerifierURL = "https://capture-wallet.credimi.io"
	DefaultLogChecker  = "eudiw"
)

// ApplyFixture substitutes service-profile values throughout a workflow.
// Keeping this at the parsed workflow boundary lets one FCAF scenario run
// against multiple issuer/verifier fixtures without duplicating the scenario.
func ApplyFixture(workflow *WorkflowDefinition) error {
	if workflow == nil {
		return nil
	}
	fixture := map[string]string{
		"issuer_url":   DefaultIssuerURL,
		"verifier_url": DefaultVerifierURL,
		"log_checker":  DefaultLogChecker,
	}
	for key, value := range workflow.Runtime.Fixture {
		fixture[key] = value
	}
	data, err := json.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("marshal workflow for fixture substitution: %w", err)
	}
	text := string(data)
	for key, value := range fixture {
		// Tokens only appear inside JSON strings, so the value is inserted
		// JSON-escaped: it can change string contents, never workflow structure.
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode fixture value %q: %w", key, err)
		}
		text = strings.ReplaceAll(
			text,
			fixtureTokenPrefix+key+"}",
			string(encoded[1:len(encoded)-1]),
		)
	}
	var substituted WorkflowDefinition
	if err := json.Unmarshal([]byte(text), &substituted); err != nil {
		return fmt.Errorf("unmarshal workflow after fixture substitution: %w", err)
	}
	*workflow = substituted
	return nil
}
