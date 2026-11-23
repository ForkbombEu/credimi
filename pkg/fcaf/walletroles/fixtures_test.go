// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletroles

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const relyingPartySuite = "wallet_solution/relying_party"

func TestRepositoryFixturesCoverEveryRelyingPartyRole(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)

	issuance := contract.Injected(InjectedIssuance).Name
	for _, role := range contract.Roles {
		if !slices.Contains(role.Suites, relyingPartySuite) || role.Name == issuance {
			continue
		}
		t.Run(role.Name, func(t *testing.T) {
			fixtures, err := contract.Fixtures(repositoryRoot, relyingPartySuite, role.Name)
			require.NoError(t, err)
			require.NotNil(t, fixtures)
			if len(role.Parameters) == 0 {
				assert.Empty(t, fixtures)
				return
			}
			require.NotEmpty(t, fixtures)
			standaloneDone := false
			for _, fixture := range fixtures {
				assert.Equal(t, fixture.Scenario+"#"+fixture.MobileStepID, fixture.ID)
				assert.Equal(t, role.Parameters[0], fixture.Parameter)
				assert.Equal(t, "POST", fixture.Request.Method)
				assert.NotContains(t, fixture.Request.URL, "${")
				assert.NotEmpty(t, fixture.ValuePath)
				assert.NotNil(t, fixture.Fetches)
				assert.NotNil(t, fixture.TestIDs)
				for _, fetch := range fixture.Fetches {
					assert.Contains(t, fetch.URL, "${session_id}")
					assert.NotContains(t, fetch.URL, "${{")
				}
				if !fixture.Standalone {
					standaloneDone = true
				}
				assert.False(
					t,
					standaloneDone && fixture.Standalone,
					"standalone fixtures come first",
				)
			}
		})
	}
}

func TestRepositoryFixturesOfTheInjectedIssuance(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)

	fixtures, err := contract.Fixtures(repositoryRoot, relyingPartySuite, "fcaf-issue-pid")
	require.NoError(t, err)
	require.Len(t, fixtures, 2)

	tests := []struct {
		id              string
		format          string
		configurationID string
	}{
		{id: "pid-sd-jwt", format: "sd-jwt", configurationID: PIDSDJWTConfigurationID},
		{id: "pid-mdoc", format: "mdoc", configurationID: PIDMdocConfigurationID},
	}
	for i, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			fixture := fixtures[i]
			assert.Equal(t, tc.id, fixture.ID)
			assert.Equal(t, tc.format, fixture.PIDFormat)
			assert.Equal(t, "deeplink", fixture.Parameter)
			assert.Equal(t, "deeplink", fixture.ValuePath)
			assert.Equal(t, PIDIssuance(tc.format), fixture.Request)
			assert.Equal(t, map[string]any{
				"issuer_configuration_id":     PIDIssuerConfigurationID,
				"credential_configuration_id": tc.configurationID,
			}, fixture.Request.Body)
			assert.Equal(t, "https://capture-wallet.credimi.io/sessions", fixture.Request.URL)
			assert.Empty(t, fixture.Fetches)
			assert.Empty(t, fixture.TestIDs)
		})
	}
}

func TestRepositoryFixturesOfARequestRejection(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)

	fixtures, err := contract.Fixtures(
		repositoryRoot,
		relyingPartySuite,
		"fcaf-expect-request-rejected",
	)
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)

	first := fixtures[0]
	assert.True(t, first.Standalone)
	assert.Equal(t, "POST", first.Request.Method)
	assert.Equal(t, "https://capture-wallet.credimi.io/openid4vp/sessions", first.Request.URL)
	assert.Equal(t, "deeplink", first.ValuePath)
	require.NotEmpty(t, first.Fetches)
	assert.True(t, strings.HasSuffix(first.Fetches[0].URL, "/openid4vp/sessions/${session_id}"))
	assert.NotEmpty(t, first.TestIDs)
	assert.NotEmpty(t, first.PipelineOutputs)
}

func TestRepositoryStartsFrom(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)

	tests := []struct {
		role string
		want []string
	}{
		{role: "fcaf-reset-and-onboard", want: []string{}},
		{role: "fcaf-expect-credential-rejected", want: []string{"fcaf-reset-and-onboard"}},
		{
			role: "fcaf-engagement-haip-vp",
			want: []string{"fcaf-reset-and-onboard", "fcaf-issue-pid"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			role, ok := contract.Role(tc.role)
			require.True(t, ok)
			assert.Equal(t, tc.want, contract.StartsFrom(role))
		})
	}
}

func TestFixturesRefusesAnUnknownRole(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)

	_, err = contract.Fixtures(repositoryRoot, relyingPartySuite, "fcaf-no-such-role")
	require.ErrorContains(t, err, `role "fcaf-no-such-role" is not in the contract`)
}

// fixtureScenarioYAML exercises every qualification rule: a qualifying
// standalone step, a later qualifying step, and steps that each break one rule.
const fixtureScenarioYAML = `name: Synthetic
runtime:
  fixture:
    verifier_url: https://verifier.example
steps:
- id: create-ok
  use: http-request
  with:
    method: POST
    url: ${fixture.verifier_url}/openid4vp/sessions
    headers: { Content-Type: application/json }
    body:
      presentation_request:
        dcql_query:
          credentials: [{ id: a, format: dc+sd-jwt }, { id: b, format: mso_mdoc }]
- id: run-ok
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-ok.outputs.body.deeplink }}" }
- id: fetch-ok
  use: http-request
  with: { method: GET, url: "${fixture.verifier_url}/openid4vp/sessions/${{create-ok.outputs.body.session_id}}" }
- id: fetch-other
  use: http-request
  with: { method: GET, url: "${fixture.verifier_url}/did.json" }
- id: create-authorized
  use: http-request
  with:
    method: POST
    url: ${fixture.verifier_url}/openid4vp/sessions
    headers: { Authorization: Bearer x }
- id: run-authorized
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-authorized.outputs.body.deeplink }}" }
- id: create-chained
  use: http-request
  with:
    method: POST
    url: ${fixture.verifier_url}/openid4vp/sessions
    body: { nonce: "${{ fetch-ok.outputs.body.nonce }}" }
- id: run-chained
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-chained.outputs.body.deeplink }}" }
- id: create-unknown-fixture
  use: http-request
  with: { method: POST, url: "${fixture.missing}/sessions" }
- id: run-unknown-fixture
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-unknown-fixture.outputs.body.deeplink }}" }
- id: create-get
  use: http-request
  with: { method: GET, url: "${fixture.verifier_url}/sessions" }
- id: run-get
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-get.outputs.body.deeplink }}" }
- id: run-two-parameters
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-ok.outputs.body.deeplink }}", other: x }
- id: run-other-role
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-other
    parameters: { deeplink: "${{ create-ok.outputs.body.deeplink }}" }
- id: create-later
  use: http-request
  with: { method: POST, url: "${fixture.issuer_url}/sessions", headers: { accept: application/json } }
- id: run-later
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-role
    parameters: { deeplink: "${{ create-later.outputs.body.credential_offer.uri }}" }
- id: validate
  use: fcaf-validation
  with:
    test_ids: [T_OK, T_FOREIGN, T_UNKNOWN]
    test_id: T_OK
    pipeline_outputs:
      pipeline.synthetic:
        output:
          session: ${{ fetch-ok.outputs.body }}
          shots: ${{ run-ok.outputs.flow_output.maestro_screenshot_urls }}
      pipeline.foreign:
        output:
          did: ${{ fetch-other.outputs.body }}
`

func fixtureTestYAML(id string, from ...string) string {
	evidence := ""
	for i, binding := range from {
		evidence += "  e" + string(rune('a'+i)) + ": { from: " + binding + " }\n"
	}
	return "id: " + id + `
title: Synthetic
suite: { sut: wallet_solution, role: relying_party }
normative_references: [{ title: Synthetic }]
evidence:
` + evidence + `assertions:
- id: present
  validator: evidence.screenshot_named
  input: evidence.ea
  params: { name: done }
verdict:
  pass_when: all_assertions_pass
`
}

func TestFixturesQualifyOnlySelfContainedSteps(t *testing.T) {
	root := t.TempDir()
	suiteRoot := filepath.Join(root, filepath.FromSlash(relyingPartySuite))
	require.NoError(t, os.MkdirAll(filepath.Join(suiteRoot, "scenarios"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(suiteRoot, "tests"), 0o755))
	write := func(path, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(suiteRoot, path), []byte(content), 0o600))
	}
	write("scenarios/synthetic.yaml", fixtureScenarioYAML)
	write("tests/T_OK.yaml", fixtureTestYAML(
		"T_OK",
		"pipeline.synthetic.outputs.session",
		"pipeline.synthetic.outputs.shots",
	))
	write("tests/T_FOREIGN.yaml", fixtureTestYAML(
		"T_FOREIGN",
		"pipeline.synthetic.outputs.session",
		"pipeline.foreign.outputs.did",
	))

	contract := &Contract{
		ReferencePrefix: "org/wallet/",
		Roles: []Role{
			{Name: "fcaf-role", Parameters: []string{"deeplink"}},
			{Name: "fcaf-bare"},
		},
	}

	bare, err := contract.Fixtures(root, relyingPartySuite, "fcaf-bare")
	require.NoError(t, err)
	assert.Equal(t, []Fixture{}, bare)

	fixtures, err := contract.Fixtures(root, relyingPartySuite, "fcaf-role")
	require.NoError(t, err)
	require.Len(t, fixtures, 2)

	tests := []struct {
		name string
		got  Fixture
		want Fixture
	}{
		{
			name: "standalone",
			got:  fixtures[0],
			want: Fixture{
				ID:           "synthetic#run-ok",
				Scenario:     "synthetic",
				MobileStepID: "run-ok",
				Standalone:   true,
				PIDFormat:    "mdoc",
				Parameter:    "deeplink",
				Request: FixtureRequest{
					Method:  "POST",
					URL:     "https://verifier.example/openid4vp/sessions",
					Headers: map[string]string{"Content-Type": "application/json"},
					Body: map[string]any{"presentation_request": map[string]any{
						"dcql_query": map[string]any{"credentials": []any{
							map[string]any{"id": "a", "format": "dc+sd-jwt"},
							map[string]any{"id": "b", "format": "mso_mdoc"},
						}},
					}},
				},
				ValuePath: "deeplink",
				Fetches: []FixtureFetch{{
					StepID: "fetch-ok",
					URL:    "https://verifier.example/openid4vp/sessions/${session_id}",
				}},
				TestIDs: []string{"T_OK"},
				PipelineOutputs: map[string]any{"pipeline.synthetic": map[string]any{
					"output": map[string]any{
						"session": "${{ fetch-ok.outputs.body | optional }}",
						"shots":   "${{ run-ok.outputs.flow_output.maestro_screenshot_urls | optional }}",
					},
				}},
			},
		},
		{
			name: "later",
			got:  fixtures[1],
			want: Fixture{
				ID:           "synthetic#run-later",
				Scenario:     "synthetic",
				MobileStepID: "run-later",
				Parameter:    "deeplink",
				Request: FixtureRequest{
					Method:  "POST",
					URL:     "https://issuer-backend.eudiw.dev/sessions",
					Headers: map[string]string{"accept": "application/json"},
				},
				ValuePath:       "credential_offer.uri",
				Fetches:         []FixtureFetch{},
				TestIDs:         []string{},
				PipelineOutputs: map[string]any{},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.got)
		})
	}
}

func TestOptionalizeExpressions(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "expression", value: "${{ a.outputs.b }}", want: "${{ a.outputs.b | optional }}"},
		{
			name:  "already optional",
			value: "${{ a.outputs.b | optional }}",
			want:  "${{ a.outputs.b | optional }}",
		},
		{name: "embedded", value: "x ${{ a.outputs.b }}", want: "x ${{ a.outputs.b }}"},
		{
			name:  "nested",
			value: map[string]any{"k": []any{"${{a.outputs.b}}", 1}},
			want:  map[string]any{"k": []any{"${{ a.outputs.b | optional }}", 1}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, OptionalizeExpressions(tc.value))
		})
	}
}
