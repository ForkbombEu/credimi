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
	require.Len(t, fixtures, 4)

	tests := []struct {
		id                    string
		issuer                string
		format                string
		issuerConfigurationID string
		configurationID       string
	}{
		{
			id:                    "pid-sd-jwt",
			issuer:                PIDIssuerKeyAttestation,
			format:                "sd-jwt",
			issuerConfigurationID: "eu-pid-device-bound",
			configurationID:       "urn:eu.europa.ec.eudi:pid:1.sd-jwt.key-attestation-required",
		},
		{
			id:                    "pid-mdoc",
			issuer:                PIDIssuerKeyAttestation,
			format:                "mdoc",
			issuerConfigurationID: "eu-pid-device-bound",
			configurationID:       "urn:eu.europa.ec.eudi:pid:1.mdoc.key-attestation-required",
		},
		{
			id:                    "pid-jwt-proof-sd-jwt",
			issuer:                PIDIssuerJWTProof,
			format:                "sd-jwt",
			issuerConfigurationID: "eu-pid-jwt-proof-only",
			configurationID:       "urn:eu.europa.ec.eudi:pid:1.sd-jwt.jwt-proof",
		},
		{
			id:                    "pid-jwt-proof-mdoc",
			issuer:                PIDIssuerJWTProof,
			format:                "mdoc",
			issuerConfigurationID: "eu-pid-jwt-proof-only",
			configurationID:       "urn:eu.europa.ec.eudi:pid:1.mdoc.jwt-proof",
		},
	}
	for i, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			fixture := fixtures[i]
			assert.Equal(t, tc.id, fixture.ID)
			assert.Equal(t, tc.format, fixture.PIDFormat)
			assert.Equal(t, tc.issuer, fixture.PIDIssuer)
			assert.Equal(t, "deeplink", fixture.Parameter)
			assert.Equal(t, "deeplink", fixture.ValuePath)
			assert.Equal(t, PIDIssuance(tc.issuer, tc.format), fixture.Request)
			assert.Equal(t, map[string]any{
				"issuer_configuration_id":     tc.issuerConfigurationID,
				"credential_configuration_id": tc.configurationID,
			}, fixture.Request.Body)
			assert.Equal(t, "https://capture-wallet.credimi.io/sessions", fixture.Request.URL)
			assert.Empty(t, fixture.Fetches)
			assert.Empty(t, fixture.TestIDs)
			assert.True(t, fixture.Complete)
		})
	}
}

func TestPIDIssuancePanicsOnAnUnknownIssuer(t *testing.T) {
	assert.Panics(t, func() { PIDIssuance("unknown", "sd-jwt") })
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
		{role: "fcaf-expect-credential-absent", want: []string{"fcaf-reset-and-onboard"}},
	}
	for _, tc := range tests {
		t.Run(tc.role, func(t *testing.T) {
			role, ok := contract.Role(tc.role)
			require.True(t, ok)
			assert.Equal(t, tc.want, contract.StartsFrom(role))
		})
	}
}

// repositoryFixture returns the role's fixture of that id.
func repositoryFixture(t *testing.T, contract *Contract, role, id string) Fixture {
	t.Helper()
	fixtures, err := contract.Fixtures(repositoryRoot, relyingPartySuite, role)
	require.NoError(t, err)
	for _, fixture := range fixtures {
		if fixture.ID == id {
			return fixture
		}
	}
	require.Failf(t, "fixture not found", "%s has no fixture %s", role, id)
	return Fixture{}
}

func TestRepositoryFixtureSetup(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)
	issuance := contract.Injected(InjectedIssuance).Name

	// A request of both PID formats starts from the mdoc PID and gets the
	// sd-jwt one in its setup.
	both := repositoryFixture(
		t,
		contract,
		"fcaf-present-all-credentials",
		"fcaf-wallet-solution-relying-party-dcql-all-credentials-without-credential-sets#present-all-credentials",
	)
	assert.Equal(t, "mdoc", both.PIDFormat)
	assert.Equal(t, []FixtureSetup{{Role: issuance, PIDFormat: "sd-jwt"}}, both.Setup)

	// A later step starts from the scenario's own earlier offers.
	candidate := repositoryFixture(t, contract, "fcaf-select-and-share-candidate",
		"fcaf-wallet-solution-relying-party-pid-candidate-selection#select-and-share-candidate")
	assert.False(t, candidate.Standalone)
	assert.True(t, candidate.Complete)
	// Its offers issue the sd-jwt PID its request asks for besides the start
	// state's: no injected PID is added.
	require.Len(t, candidate.Setup, 2)
	for i, fixtureID := range []string{"pid_default", "pid_person_b"} {
		setup := candidate.Setup[i]
		assert.Equal(t, "getcredential-generic-credential-without-authentication", setup.Role)
		assert.Empty(t, setup.PIDFormat)
		assert.Equal(t, "deeplink", setup.ValuePath)
		require.NotNil(t, setup.Request)
		assert.Equal(t, "POST", setup.Request.Method)
		assert.Equal(t, "https://capture-wallet.credimi.io/sessions", setup.Request.URL)
		assert.Equal(t, fixtureID, setup.Request.Body.(map[string]any)["fixture_id"])
	}

	// A standalone request of one format needs nothing more.
	fixtures, err := contract.Fixtures(
		repositoryRoot,
		relyingPartySuite,
		"fcaf-expect-request-rejected",
	)
	require.NoError(t, err)
	require.NotEmpty(t, fixtures)
	require.True(t, fixtures[0].Standalone)
	assert.Equal(t, []string{fixtures[0].PIDFormat}, RequestPIDFormats(fixtures[0].Request.Body))
	assert.NotNil(t, fixtures[0].Setup)
	assert.Empty(t, fixtures[0].Setup)
}

func TestRepositoryFixtureSetupSkipsRejectedOffers(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)
	const rejected = "fcaf-expect-credential-rejected"

	checked := 0
	for _, role := range []string{rejected, "fcaf-expect-request-rejected"} {
		fixtures, err := contract.Fixtures(repositoryRoot, relyingPartySuite, role)
		require.NoError(t, err)
		for _, fixture := range fixtures {
			if !strings.Contains(fixture.Scenario, "status-reference-rejection") {
				continue
			}
			checked++
			for _, setup := range fixture.Setup {
				assert.NotEqual(t, rejected, setup.Role, fixture.ID)
			}
		}
	}
	assert.Positive(t, checked)
}

func TestRepositoryFixtureCompleteness(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)

	// The PID this step presents was issued from a credential-offer step,
	// which no setup reproduces.
	afterOffer := repositoryFixture(
		t,
		contract,
		"fcaf-engagement-haip-vp",
		"fcaf-wallet-solution-relying-party-dcql-credential-formats#credential-formats-036-present-pid-sdjwt",
	)
	assert.False(t, afterOffer.Standalone)
	assert.False(t, afterOffer.Complete)

	standalone := repositoryFixture(t, contract, "fcaf-engagement-haip-vp",
		"fcaf-wallet-solution-relying-party-dcql-baseline#exercise-wallet")
	assert.True(t, standalone.Complete)
}

func TestFixtureSetupCountsOfferedPIDFormats(t *testing.T) {
	contract := &Contract{
		ReferencePrefix: "org/wallet/",
		Roles: []Role{
			{Name: "fcaf-present", Parameters: []string{"deeplink"}, Category: "verify-credential"},
			{Name: "fcaf-offer", Parameters: []string{"deeplink"}, IssuesCredential: true},
			{Name: "fcaf-issue-pid", Parameters: []string{"deeplink"}, Injected: InjectedIssuance},
		},
	}
	present, _ := contract.Role("fcaf-present")
	createOffer := map[string]any{
		"id":  "create-offer",
		"use": httpRequestUse,
		"with": map[string]any{
			"method": "POST",
			"url":    "https://issuer.example/sessions",
			"body":   map[string]any{"credential_configuration_id": "urn:pid:1.mdoc"},
		},
	}
	offer := map[string]any{"id": "offer", "use": mobileAutomationUse, "with": map[string]any{
		"action_id":  "org/wallet/fcaf-offer",
		"parameters": map[string]any{"deeplink": "${{ create-offer.outputs.body.deeplink }}"},
	}}
	scenario := fixtureScenario{byID: map[string]map[string]any{"create-offer": createOffer}}
	fixture := Fixture{PIDFormat: "sd-jwt", Request: FixtureRequest{Body: map[string]any{
		"presentation_request": map[string]any{"dcql_query": map[string]any{"credentials": []any{
			map[string]any{"id": "a", "format": "dc+sd-jwt"},
			map[string]any{"id": "b", "format": "mso_mdoc"},
		}}},
	}}}

	setup, complete := contract.fixtureSetup(
		fixture,
		present,
		scenario,
		[]map[string]any{createOffer, offer},
	)
	assert.True(t, complete)
	require.Len(t, setup, 1, "the mdoc offer stands for the mdoc PID")
	assert.Equal(t, "fcaf-offer", setup[0].Role)
}

func TestRequestPIDFormats(t *testing.T) {
	request := func(credentials ...any) any {
		return map[string]any{"presentation_request": map[string]any{
			"dcql_query": map[string]any{"credentials": credentials},
		}}
	}
	sdJWT := func(vcts ...any) any {
		credential := map[string]any{"format": "dc+sd-jwt"}
		if vcts != nil {
			credential["meta"] = map[string]any{"vct_values": vcts}
		}
		return credential
	}
	mdoc := map[string]any{"format": "mso_mdoc"}
	assert.Equal(
		t,
		[]string{"sd-jwt", "mdoc"},
		RequestPIDFormats(request(mdoc, sdJWT("urn:eudi:pid:1"))),
	)
	assert.Equal(t, []string{"sd-jwt"}, RequestPIDFormats(request(sdJWT())))
	assert.Equal(
		t,
		[]string{"mdoc"},
		RequestPIDFormats(request(mdoc, sdJWT("urn:credimi:degree:1"))),
	)
	assert.Empty(t, RequestPIDFormats(map[string]any{"issuer_configuration_id": "x"}))
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
		ReferencePrefix:      "org/wallet/",
		AuthoringClassFields: ClassFields,
		Roles: []Role{
			{Name: "fcaf-role", Parameters: []string{"deeplink"}, Category: "verify-credential"},
			{Name: "fcaf-bare"},
			{Name: "fcaf-issue-pid", Parameters: []string{"deeplink"}, Injected: InjectedIssuance},
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
				Setup:    []FixtureSetup{{Role: "fcaf-issue-pid", PIDFormat: "sd-jwt"}},
				Complete: true,
				Class:    "dc+sd-jwt + mso_mdoc, openid4vp://",
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
				Setup:           []FixtureSetup{},
				Complete:        true,
				Class:           "credential offer, dc+sd-jwt, openid-credential-offer://",
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

func TestWithRequestScheme(t *testing.T) {
	haip := map[string]any{"scheme": HAIPRequestScheme, "dcql": map[string]any{"q": []any{"a"}}}
	tests := []struct {
		name   string
		body   any
		scheme string
		want   any
	}{
		{
			name:   "haip to openid4vp",
			body:   haip,
			scheme: OpenID4VPRequestScheme,
			want: map[string]any{
				"scheme": OpenID4VPRequestScheme,
				"dcql":   map[string]any{"q": []any{"a"}},
			},
		},
		{name: "no scheme asked", body: haip, scheme: "", want: haip},
		{name: "haip asked", body: haip, scheme: HAIPRequestScheme, want: haip},
		{
			name:   "other scheme",
			body:   map[string]any{"scheme": "eu-eaap://"},
			scheme: OpenID4VPRequestScheme,
			want:   map[string]any{"scheme": "eu-eaap://"},
		},
		{
			name:   "missing scheme",
			body:   map[string]any{"dcql": "x"},
			scheme: OpenID4VPRequestScheme,
			want:   map[string]any{"dcql": "x"},
		},
		{
			name:   "nested scheme",
			body:   map[string]any{"inner": map[string]any{"scheme": HAIPRequestScheme}},
			scheme: OpenID4VPRequestScheme,
			want:   map[string]any{"inner": map[string]any{"scheme": HAIPRequestScheme}},
		},
		{name: "not an object", body: "x", scheme: OpenID4VPRequestScheme, want: "x"},
		{name: "nil", body: nil, scheme: OpenID4VPRequestScheme, want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, WithRequestScheme(tc.body, tc.scheme))
		})
	}
	assert.Equal(t, HAIPRequestScheme, haip["scheme"], "the input is not modified")

	copied := WithRequestScheme(haip, "").(map[string]any)
	copied["dcql"].(map[string]any)["q"].([]any)[0] = "b"
	assert.Equal(t, "a", haip["dcql"].(map[string]any)["q"].([]any)[0], "the copy is deep")
}

// roleTestsScenarioYAML runs two roles on the same request: T_SHARED reads
// only the session both fixtures fetch, T_A also the first role's
// screenshots, and T_MULTI the sessions of two requests.
const roleTestsScenarioYAML = `name: Shared
runtime:
  fixture:
    verifier_url: https://verifier.example
steps:
- id: create-a
  use: http-request
  with:
    method: POST
    url: ${fixture.verifier_url}/openid4vp/sessions
    body: { presentation_request: { dcql_query: { credentials: [{ id: a, format: dc+sd-jwt }] } } }
- id: run-a
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-a
    parameters: { deeplink: "${{ create-a.outputs.body.deeplink }}" }
- id: fetch-a
  use: http-request
  with: { method: GET, url: "${fixture.verifier_url}/openid4vp/sessions/${{create-a.outputs.body.session_id}}" }
- id: run-b
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-b
    parameters: { deeplink: "${{ create-a.outputs.body.deeplink }}" }
- id: create-c
  use: http-request
  with:
    method: POST
    url: ${fixture.verifier_url}/openid4vp/sessions
    body: { presentation_request: { dcql_query: { credentials: [{ id: c, format: mso_mdoc }] } } }
- id: run-c
  use: mobile-automation
  with:
    action_id: org/wallet/fcaf-c
    parameters: { deeplink: "${{ create-c.outputs.body.deeplink }}" }
- id: fetch-c
  use: http-request
  with: { method: GET, url: "${fixture.verifier_url}/openid4vp/sessions/${{create-c.outputs.body.session_id}}" }
- id: validate
  use: fcaf-validation
  with:
    test_ids: [T_SHARED, T_A, T_MULTI]
    pipeline_outputs:
      pipeline.a:
        output:
          session: ${{ fetch-a.outputs.body }}
          shots: ${{ run-a.outputs.flow_output.maestro_screenshot_urls }}
      pipeline.c:
        output:
          session: ${{ fetch-c.outputs.body }}
`

func TestRoleTests(t *testing.T) {
	root := t.TempDir()
	suiteRoot := filepath.Join(root, filepath.FromSlash(relyingPartySuite))
	require.NoError(t, os.MkdirAll(filepath.Join(suiteRoot, "scenarios"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(suiteRoot, "tests"), 0o755))
	write := func(path, content string) {
		require.NoError(t, os.WriteFile(filepath.Join(suiteRoot, path), []byte(content), 0o600))
	}
	write("scenarios/shared.yaml", roleTestsScenarioYAML)
	write("tests/T_SHARED.yaml", fixtureTestYAML("T_SHARED", "pipeline.a.outputs.session"))
	write("tests/T_A.yaml", fixtureTestYAML(
		"T_A",
		"pipeline.a.outputs.session",
		"pipeline.a.outputs.shots",
	))
	write("tests/T_MULTI.yaml", fixtureTestYAML(
		"T_MULTI",
		"pipeline.a.outputs.session",
		"pipeline.c.outputs.session",
	))
	suites := []string{relyingPartySuite}
	contract := &Contract{
		ReferencePrefix:      "org/wallet/",
		AuthoringClassFields: ClassFields,
		Roles: []Role{
			{Name: "fcaf-a", Parameters: []string{"deeplink"}, Suites: suites},
			{Name: "fcaf-b", Parameters: []string{"deeplink"}, Suites: suites},
			{Name: "fcaf-c", Parameters: []string{"deeplink"}, Suites: suites},
			{Name: "fcaf-elsewhere", Parameters: []string{"deeplink"}},
		},
	}

	roleTests, err := contract.RoleTests(root, relyingPartySuite)
	require.NoError(t, err)
	assert.Equal(t, map[string][]string{
		"fcaf-a": {"T_A", "T_SHARED"},
		"fcaf-b": {"T_SHARED"},
	}, roleTests, "a shared test belongs to both roles, a multi-step test to none")

	again, err := contract.RoleTests(root, relyingPartySuite)
	require.NoError(t, err)
	assert.Equal(t, roleTests, again)
}
