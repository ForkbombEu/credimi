// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package walletroles

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClass(t *testing.T) {
	all := &Contract{AuthoringClassFields: ClassFields}
	sdJWT := map[string]any{"format": "dc+sd-jwt"}
	mdoc := map[string]any{"format": "mso_mdoc"}
	for _, testCase := range []struct {
		name     string
		contract *Contract
		body     any
		want     string
	}{
		{
			name:     "nested query, default scheme",
			contract: all,
			body: map[string]any{"presentation_request": map[string]any{
				"dcql_query": map[string]any{"credentials": []any{mdoc, sdJWT, sdJWT}},
			}},
			want: "2x dc+sd-jwt + mso_mdoc, openid4vp://",
		},
		{
			name:     "top-level query, credential sets, scheme, reply, redirect",
			contract: all,
			body: map[string]any{
				"dcql_query": map[string]any{
					"credentials":     []any{sdJWT},
					"credential_sets": []any{},
				},
				"scheme":            "haip-vp://",
				"response_scenario": map[string]any{"status": 400, "content_type": "application/json"},
				"redirect_uri":      "https://verifier.example/redirect",
				"request_delivery":  "by_value",
				"client_id_scheme":  "x509_san_dns",
			},
			want: "dc+sd-jwt, with credential_sets, haip-vp://, " +
				"verifier replies 400 application/json, with redirect_uri",
		},
		{
			name:     "a nested scheme is ignored",
			contract: all,
			body: map[string]any{"presentation_request": map[string]any{
				"scheme": "eu-eaap://", "dcql_query": nil,
			}},
			want: "no dcql credentials, openid4vp://",
		},
		{
			name:     "only listed fields",
			contract: &Contract{AuthoringClassFields: []string{ClassFieldScheme}},
			body: map[string]any{
				"dcql_query":   map[string]any{"credentials": []any{sdJWT}},
				"redirect_uri": "https://verifier.example/redirect",
			},
			want: "openid4vp://",
		},
		{
			name:     "nothing listed tells it apart",
			contract: &Contract{AuthoringClassFields: []string{ClassFieldRedirectURI}},
			body:     map[string]any{"presentation_request": map[string]any{}},
			want:     "any request",
		},
		{
			name:     "mdoc offer, default offer scheme",
			contract: all,
			body: map[string]any{
				"credential_configuration_id": "eu.europa.ec.eudi.pid_mdoc",
				"tx_code":                     "1234",
			},
			want: "credential offer, mso_mdoc, openid-credential-offer://",
		},
		{
			name:     "sd-jwt offer, offer scheme",
			contract: all,
			body: map[string]any{
				"credential_configuration_id": "eu.europa.ec.eudi.pid_vc_sd_jwt",
				"offer_scheme":                "haip-vci://",
			},
			want: "credential offer, dc+sd-jwt, haip-vci://",
		},
		{
			name:     "offer without listed offer fields",
			contract: &Contract{AuthoringClassFields: []string{ClassFieldScheme}},
			body:     map[string]any{"credential_configuration_id": "x_mdoc"},
			want:     "credential offer",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, testCase.contract.Class(testCase.body))
		})
	}
}

func TestAuthoringFixtures(t *testing.T) {
	fixtures := []Fixture{
		{ID: "s#late", Class: "a", Complete: true, TestIDs: []string{"T1", "T2", "T3"}},
		{ID: "s#first", Class: "a", Standalone: true, Complete: true, TestIDs: []string{"T1"}},
		{ID: "s#second", Class: "a", Standalone: true, Complete: true, TestIDs: []string{"T1"}},
		{ID: "s#untested", Class: "b", Standalone: true, Complete: true},
		{ID: "s#tested", Class: "b", Complete: true, TestIDs: []string{"T1", "T2"}},
		{ID: "s#only", Class: "c", Standalone: true, Complete: true},
		{ID: "s#incomplete", Class: "d", TestIDs: []string{"T1", "T2", "T3", "T4"}},
	}
	got := AuthoringFixtures(fixtures)
	ids := make([]string, 0, len(got))
	for _, fixture := range got {
		ids = append(ids, fixture.ID)
	}
	assert.Equal(t, []string{"s#first", "s#only", "s#tested", "s#incomplete"}, ids)
	assert.Empty(t, AuthoringFixtures(nil))
}

func TestAuthoringSet(t *testing.T) {
	contract, err := Load(repositoryRoot)
	require.NoError(t, err)
	const suite, roleName = "wallet_solution/relying_party", "fcaf-engagement-haip-vp"

	issuance := contract.Injected(InjectedIssuance).Name
	pids, err := contract.AuthoringSet(repositoryRoot, suite, issuance)
	require.NoError(t, err)
	all, err := contract.Fixtures(repositoryRoot, suite, issuance)
	require.NoError(t, err)
	assert.Equal(t, all, pids, "the PID issuance keeps every fixture, in order")
	assert.Equal(t, PIDIssuerKeyAttestation, pids[0].PIDIssuer)

	set, err := contract.AuthoringSet(repositoryRoot, suite, roleName)
	require.NoError(t, err)
	fixtures, err := contract.Fixtures(repositoryRoot, suite, roleName)
	require.NoError(t, err)
	pinned := ""
	for _, fixture := range fixtures {
		if !slices.ContainsFunc(set, func(f Fixture) bool { return f.ID == fixture.ID }) {
			pinned = fixture.ID
			break
		}
	}
	require.NotEmpty(t, pinned)
	for i := range contract.Roles {
		if contract.Roles[i].Name == roleName {
			contract.Roles[i].AuthoringFixtures = []string{pinned}
		}
	}
	withPin, err := contract.AuthoringSet(repositoryRoot, suite, roleName)
	require.NoError(t, err)
	require.Len(t, withPin, len(set)+1)
	assert.Equal(t, pinned, withPin[len(set)].ID)

	for i := range contract.Roles {
		if contract.Roles[i].Name == roleName {
			contract.Roles[i].AuthoringFixtures = []string{"unknown#step"}
		}
	}
	_, err = contract.AuthoringSet(repositoryRoot, suite, roleName)
	require.ErrorContains(t, err, `authoring fixture "unknown#step"`)
}
