// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package validators

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The status-list URIs Capture production emits, probed 22/10/2026. The
// malformed shapes are derived from the valid value so each fixture differs
// from it in exactly one respect.
const (
	pidStatusListURI  = "https://tsl.credimi.io/token_status_list/EU/urn:eudi:pid:1/d50c88a6-f49c-4a2d-9731-ad638b873429"
	mdocStatusListURI = "https://tsl.credimi.io/token_status_list/EU/eu.europa.ec.eudi.pid.1/4d93fe90-16f6-4c44-b8e2-249de9b2100d"
)

// captureStatusURIShape pairs one `uri_*` shape with the value Capture emits
// for it.
type captureStatusURIShape struct {
	shape string
	uri   string
}

func captureStatusURIShapes(valid string) []captureStatusURIShape {
	return []captureStatusURIShape{
		{statusShapeURIMissingScheme, strings.TrimPrefix(valid, "https:")},
		{statusShapeURISpaceInHost, strings.Replace(valid, "tsl.credimi.io", "tsl.cre dimi.io", 1)},
		{statusShapeURIInvalidPercentEncoding, valid + "%GG"},
		{statusShapeURIIllegalCharacters, valid + "<token>|1"},
		{statusShapeURIEmpty, ""},
	}
}

func TestAnalyzeStatusURI(t *testing.T) {
	valid := pidStatusListURI
	for _, test := range []struct {
		name                   string
		uri                    string
		missingScheme          bool
		spaceInHost            bool
		invalidPercentEncoding bool
		illegalInPath          bool
		empty                  bool
		malformed              bool
	}{
		{name: "valid capture value", uri: valid},
		{
			name:          "capture malformed value",
			uri:           "not a uri",
			missingScheme: true,
			illegalInPath: true,
			malformed:     true,
		},
		{
			name:          "missing scheme",
			uri:           "//tsl.credimi.io/token_status_list/EU/urn:eudi:pid:1/d50c88a6-f49c-4a2d-9731-ad638b873429",
			missingScheme: true,
			malformed:     true,
		},
		{
			name:        "space in host",
			uri:         "https://tsl.cre dimi.io/token_status_list/EU/urn:eudi:pid:1/d50c88a6-f49c-4a2d-9731-ad638b873429",
			spaceInHost: true,
			malformed:   true,
		},
		{
			name:                   "invalid percent encoding",
			uri:                    "https://tsl.credimi.io/token_status_list/EU/urn:eudi:pid:1/d50c88a6-f49c-4a2d-9731-ad638b873429%GG",
			invalidPercentEncoding: true,
			malformed:              true,
		},
		{
			name:          "illegal characters",
			uri:           "https://tsl.credimi.io/token_status_list/EU/urn:eudi:pid:1/d50c88a6-f49c-4a2d-9731-ad638b873429<token>|1",
			illegalInPath: true,
			malformed:     true,
		},
		{name: "empty", uri: "", empty: true, malformed: true},
		{
			// Missed by the previous url.Parse + IsAbs + Host check.
			name:          "space in path",
			uri:           strings.Replace(valid, "/token_status_list", "/token status_list", 1),
			illegalInPath: true,
			malformed:     true,
		},
		{
			// Missed by the previous url.Parse + IsAbs + Host check too.
			name:          "pipe in path",
			uri:           valid + "|",
			illegalInPath: true,
			malformed:     true,
		},
		{name: "space in userinfo is not a space in the host", uri: "https://us er@tsl.credimi.io/list/1", malformed: true},
		{name: "userinfo", uri: "https://user@tsl.credimi.io/list/1"},
		{name: "port", uri: "https://tsl.credimi.io:8443/list/1"},
		{name: "escaped percent", uri: "https://tsl.credimi.io/list/%41?x=%2F#%7E"},
		{name: "relative reference", uri: "status.example/list/1", missingScheme: true, malformed: true},
		{name: "no authority", uri: "urn:eudi:pid:1", malformed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			defects := analyzeStatusURI(test.uri)
			require.Equal(t, test.missingScheme, defects.missingScheme, "missingScheme")
			require.Equal(t, test.spaceInHost, defects.spaceInHost, "spaceInHost")
			require.Equal(t, test.invalidPercentEncoding, defects.invalidPercentEncoding)
			require.Equal(t, test.illegalInPath, defects.illegalInPath, "illegalInPath")
			require.Equal(t, test.empty, defects.empty, "empty")
			require.Equal(t, test.malformed, defects.malformed(), "malformed")
		})
	}

	t.Run("an empty uri is not a missing scheme", func(t *testing.T) {
		require.False(t, matchesStatusURIShape("", statusShapeURIMissingScheme))
		require.True(t, matchesStatusURIShape("", statusShapeURIEmpty))
	})
}

func TestOID4VPMalformedStatusURIShapesSDJWT(t *testing.T) {
	const documentNumber = "CREDIMI-DEMO-CASE"
	statusToken := func(uri string) string {
		return testSDJWT(t, map[string]any{
			"document_number": documentNumber,
			"status": map[string]any{"status_list": map[string]any{
				"uri": uri,
				"idx": 42,
			}},
		})
	}
	exchange := func(tokens ...string) map[string]any {
		entries := make([]any, 0, len(tokens))
		for _, token := range tokens {
			entries = append(entries, token)
		}
		return map[string]any{
			"authorization_request": map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id":       "pid",
					"format":   "dc+sd-jwt",
					"multiple": true,
					"meta":     map[string]any{"vct_values": []any{testPIDVCT}},
					"claims": []any{map[string]any{
						"path":   []any{"document_number"},
						"values": []any{documentNumber},
					}},
				}}},
			},
			"observed": map[string]any{"wallet_response": map[string]any{
				"value": map[string]any{"vp_token": map[string]any{"pid": entries}},
			}},
		}
	}
	expect := func(t *testing.T, value any, shape string, want Status) {
		t.Helper()
		result := OID4VPMalformedStatusCredentialAbsentValidator{}.Validate(
			context.Background(),
			Input{Value: value, Params: map[string]any{
				"vct":   testPIDVCT,
				"claim": "document_number",
				"value": documentNumber,
				"shape": shape,
			}},
		)
		require.Equal(t, want, result.Status, result.Message)
	}

	valid := statusToken(pidStatusListURI)
	for _, scenario := range captureStatusURIShapes(pidStatusListURI) {
		t.Run(scenario.shape, func(t *testing.T) {
			t.Run("retained", func(t *testing.T) {
				expect(t, exchange(statusToken(scenario.uri)), scenario.shape, StatusFail)
			})
			t.Run("only a valid duplicate", func(t *testing.T) {
				expect(t, exchange(valid), scenario.shape, StatusPass)
			})
			t.Run("empty vp_token", func(t *testing.T) {
				expect(t, exchange(), scenario.shape, StatusPass)
			})
		})
	}

	t.Run("shapes stay distinct", func(t *testing.T) {
		spaceInHost := statusToken(captureStatusURIShapes(pidStatusListURI)[1].uri)
		illegalCharacters := statusToken(captureStatusURIShapes(pidStatusListURI)[3].uri)
		expect(t, exchange(spaceInHost), statusShapeURIIllegalCharacters, StatusPass)
		expect(t, exchange(illegalCharacters), statusShapeURISpaceInHost, StatusPass)
		expect(t, exchange(statusToken("")), statusShapeURIMissingScheme, StatusPass)
	})

	t.Run("malformed uri", func(t *testing.T) {
		expect(t, exchange(statusToken("not a uri")), statusShapeMalformedURI, StatusFail)
		expect(t, exchange(statusToken(pidStatusListURI+"|")), statusShapeMalformedURI, StatusFail)
		expect(
			t,
			exchange(statusToken(strings.Replace(
				pidStatusListURI,
				"/token_status_list",
				"/token status_list",
				1,
			))),
			statusShapeMalformedURI,
			StatusFail,
		)
		expect(t, exchange(valid), statusShapeMalformedURI, StatusPass)
	})

	t.Run("unknown shape", func(t *testing.T) {
		expect(t, exchange(valid), "uri_totally_unknown", StatusError)
	})
}

func TestOID4VPMalformedStatusURIShapesMDoc(t *testing.T) {
	const documentNumber = "CREDIMI-DEMO-CASE"
	statusMDoc := func(uri string) string {
		return encodedMSOStatusDeviceResponse(
			t,
			map[string]any{"status_list": map[string]any{"idx": uint64(42), "uri": uri}},
			map[string]any{"document_number": documentNumber},
		)
	}
	exchange := func(tokens ...string) map[string]any {
		entries := make([]any, 0, len(tokens))
		for _, token := range tokens {
			entries = append(entries, token)
		}
		return map[string]any{
			"authorization_request": map[string]any{
				"dcql_query": map[string]any{"credentials": []any{map[string]any{
					"id":       "pid_mdoc",
					"format":   "mso_mdoc",
					"multiple": true,
					"meta":     map[string]any{"doctype_value": msoStatusTestDocType},
					"claims": []any{map[string]any{
						"path":   []any{msoStatusTestDocType, "document_number"},
						"values": []any{documentNumber},
					}},
				}}},
			},
			"observed": map[string]any{"wallet_response": map[string]any{
				"value": map[string]any{"vp_token": map[string]any{"pid_mdoc": entries}},
			}},
		}
	}
	expect := func(t *testing.T, value any, shape string, want Status) {
		t.Helper()
		result := OID4VPMalformedStatusCredentialAbsentValidator{}.Validate(
			context.Background(),
			Input{Value: value, Params: map[string]any{
				"doctype":   msoStatusTestDocType,
				"namespace": msoStatusTestDocType,
				"claim":     "document_number",
				"value":     documentNumber,
				"shape":     shape,
			}},
		)
		require.Equal(t, want, result.Status, result.Message)
	}

	valid := statusMDoc(mdocStatusListURI)
	for _, scenario := range captureStatusURIShapes(mdocStatusListURI) {
		t.Run(scenario.shape, func(t *testing.T) {
			t.Run("retained", func(t *testing.T) {
				expect(t, exchange(statusMDoc(scenario.uri)), scenario.shape, StatusFail)
			})
			t.Run("only a valid duplicate", func(t *testing.T) {
				expect(t, exchange(valid), scenario.shape, StatusPass)
			})
			t.Run("empty vp_token", func(t *testing.T) {
				expect(t, exchange(), scenario.shape, StatusPass)
			})
		})
	}

	t.Run("shapes stay distinct", func(t *testing.T) {
		spaceInHost := statusMDoc(captureStatusURIShapes(mdocStatusListURI)[1].uri)
		illegalCharacters := statusMDoc(captureStatusURIShapes(mdocStatusListURI)[3].uri)
		expect(t, exchange(spaceInHost), statusShapeURIIllegalCharacters, StatusPass)
		expect(t, exchange(illegalCharacters), statusShapeURISpaceInHost, StatusPass)
		expect(t, exchange(statusMDoc("")), statusShapeURIMissingScheme, StatusPass)
	})

	t.Run("malformed uri", func(t *testing.T) {
		expect(t, exchange(statusMDoc("not a uri")), statusShapeMalformedURI, StatusFail)
		expect(t, exchange(statusMDoc(mdocStatusListURI+"|")), statusShapeMalformedURI, StatusFail)
		expect(t, exchange(valid), statusShapeMalformedURI, StatusPass)
	})
}
