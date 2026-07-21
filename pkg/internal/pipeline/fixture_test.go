// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package pipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyFixtureSubstitutesServiceValues(t *testing.T) {
	wf, err := ParseWorkflow(`name: fixture
runtime:
  fixture:
    issuer_url: https://issuer.example
    verifier_url: https://verifier.example
    log_checker: capture
steps:
  - id: request
    use: http-request
    with:
      url: ${fixture.verifier_url}/requests
      body:
        issuer: ${fixture.issuer_url}
        checker: ${fixture.log_checker}
`)
	require.NoError(t, err)
	require.NoError(t, ApplyFixture(wf))
	require.Equal(t, "https://verifier.example/requests", wf.Steps[0].With.Payload["url"])
	require.Equal(t, "https://issuer.example", wf.Steps[0].With.Payload["body"].(map[string]any)["issuer"])
	require.Equal(t, "capture", wf.Steps[0].With.Payload["body"].(map[string]any)["checker"])
}

func TestApplyFixtureLeavesUnknownTokensForDiagnostics(t *testing.T) {
	wf, err := ParseWorkflow(`name: fixture
runtime:
  fixture: { verifier_url: https://verifier.example }
steps:
  - id: request
    use: http-request
    with: { url: "${fixture.unknown}/requests" }
`)
	require.NoError(t, err)
	require.NoError(t, ApplyFixture(wf))
	require.Equal(t, "${fixture.unknown}/requests", wf.Steps[0].With.Payload["url"])
}

func TestApplyFixtureUsesRealServicesByDefault(t *testing.T) {
	wf, err := ParseWorkflow(`name: fixture
steps:
  - id: request
    use: http-request
    with: { url: "${fixture.verifier_url}/requests" }
`)
	require.NoError(t, err)
	require.NoError(t, ApplyFixture(wf))
	require.Equal(t, DefaultVerifierURL+"/requests", wf.Steps[0].With.Payload["url"])
}
