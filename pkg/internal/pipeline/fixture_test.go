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
	require.Equal(
		t,
		"https://issuer.example",
		wf.Steps[0].With.Payload["body"].(map[string]any)["issuer"],
	)
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

func TestApplyFixtureUsesDefaultVerifier(t *testing.T) {
	wf, err := ParseWorkflow(`name: fixture
steps:
  - id: request
    use: http-request
    with: { url: "${fixture.verifier_url}/requests" }
`)
	require.NoError(t, err)
	require.NoError(t, ApplyFixture(wf))
	require.Equal(t, "https://capture-wallet.credimi.io", DefaultVerifierURL)
	require.Equal(t, DefaultVerifierURL+"/requests", wf.Steps[0].With.Payload["url"])
}

func TestApplyFixtureValuesCannotChangeWorkflowStructure(t *testing.T) {
	injected := `a"},"use":"container-run","with":{"payload":{"image":"alpine","cmd":["id"]}},"metadata":{"z":"b`
	wf, err := ParseWorkflow(`name: audit
runtime:
  fixture:
    p: '` + injected + `'
steps:
  - id: s1
    use: http-request
    with:
      payload:
        url: https://example.invalid
    metadata:
      z: "${fixture.p}"
`)
	require.NoError(t, err)
	require.NoError(t, ApplyFixture(wf))
	require.Len(t, wf.Steps, 1)
	require.Equal(t, "http-request", wf.Steps[0].Use)
	require.Equal(t, map[string]any{"url": "https://example.invalid"}, wf.Steps[0].With.Payload)
	require.Equal(t, injected, wf.Steps[0].Metadata["z"])
}
