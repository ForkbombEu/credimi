// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveReference(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	orgID := selection.Organization.Id
	save := func(collection string, fields map[string]any) *core.Record {
		c, err := app.FindCollectionByNameOrId(collection)
		require.NoError(t, err)
		record := core.NewRecord(c)
		for key, value := range fields {
			record.Set(key, value)
		}
		require.NoError(t, app.SaveNoValidate(record))
		return record
	}
	issuer := save("credential_issuers", map[string]any{
		"owner": orgID,
		"name":  "Reference issuer",
		"url":   "https://issuer.reference.test/",
	})
	credential := save("credentials", map[string]any{
		"owner":             orgID,
		"credential_issuer": issuer.Id,
		"name":              "pid-sd-jwt",
	})
	otherIssuer := save("credential_issuers", map[string]any{
		"owner": orgID,
		"name":  "Other reference issuer",
		"url":   "https://other.reference.test",
	})
	save("credentials", map[string]any{
		"owner":             orgID,
		"credential_issuer": otherIssuer.Id,
		"name":              "pid-sd-jwt",
	})
	verifier := save("verifiers", map[string]any{"owner": orgID, "name": "Reference verifier"})
	useCase := save("use_cases_verifications", map[string]any{
		"owner":    orgID,
		"verifier": verifier.Id,
		"name":     "Reference presentation",
	})
	credentialPath := namespace + "/" + issuer.GetString("canonified_name") + "/" +
		credential.GetString("canonified_name")
	useCasePath := namespace + "/" + verifier.GetString("canonified_name") + "/" +
		useCase.GetString("canonified_name")
	credentialSource := ReferenceSource{
		Label:  "pid-sd-jwt",
		Source: SourceSpec{Kind: SourceCredentialOffer, Identifier: credentialPath},
	}
	useCaseSource := ReferenceSource{
		Label:  "Reference presentation",
		Source: SourceSpec{Kind: SourceUseCaseVerification, Identifier: useCasePath},
	}

	offerJSON := func(issuer string) string {
		encoded, err := json.Marshal(map[string]any{
			"credential_issuer":            issuer,
			"credential_configuration_ids": []string{"pid-sd-jwt", "mdoc"},
		})
		require.NoError(t, err)
		return string(encoded)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/broken" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write([]byte(offerJSON("https://issuer.reference.test")))
	}))
	t.Cleanup(server.Close)
	offerDeeplink := func(key, value string) string {
		return "openid-credential-offer://?" + url.Values{key: {value}}.Encode()
	}
	matchedOffer := map[string]string{
		"credential_issuer":            "https://issuer.reference.test",
		"credential_configuration_ids": "pid-sd-jwt,mdoc",
	}
	noOfferMatch := "no Credimi credential matches this offer; a deeplink works once, " +
		"so replays need a source that creates a new one: a Credimi credential, " +
		"or an HTTP call that creates an issuer session"

	for name, tc := range map[string]struct {
		text  string
		orgID string
		want  ReferenceResolution
	}{
		"marketplace credential URL": {
			text: " https://credimi.io/marketplace/credentials/" + credentialPath + "/ ",
			want: ReferenceResolution{
				Kind:    ReferenceCredimiRecord,
				Sources: []ReferenceSource{credentialSource},
			},
		},
		"hub use case URL": {
			text: "https://credimi.io/hub/use_cases_verifications/" + useCasePath,
			want: ReferenceResolution{
				Kind:    ReferenceCredimiRecord,
				Sources: []ReferenceSource{useCaseSource},
			},
		},
		"hub URL of another organization's record": {
			text:  "https://credimi.io/hub/credentials/" + credentialPath,
			orgID: "another-org",
			want: ReferenceResolution{
				Kind:    ReferenceCredimiRecord,
				Sources: []ReferenceSource{},
				Note:    credentialPath + " not found in your organization's or published records",
			},
		},
		"other URL": {
			text: "https://credimi.io/my/wallets",
			want: ReferenceResolution{
				Kind:    ReferenceUnknown,
				Sources: []ReferenceSource{},
				Note:    "not a Credimi URL, identifier or deeplink",
			},
		},
		"credential path": {
			text: credentialPath,
			want: ReferenceResolution{
				Kind:    ReferenceCredimiRecord,
				Sources: []ReferenceSource{credentialSource},
			},
		},
		"use case path": {
			text: useCasePath,
			want: ReferenceResolution{
				Kind:    ReferenceCredimiRecord,
				Sources: []ReferenceSource{useCaseSource},
			},
		},
		"missing path": {
			text: namespace + "/missing/x",
			want: ReferenceResolution{
				Kind:    ReferenceCredimiRecord,
				Sources: []ReferenceSource{},
				Note: namespace + "/missing/x not found in your organization's " +
					"or published records",
			},
		},
		"credential offer": {
			text: offerDeeplink("credential_offer", offerJSON("https://issuer.reference.test/")),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{credentialSource},
				Decoded: map[string]string{
					"credential_issuer":            "https://issuer.reference.test/",
					"credential_configuration_ids": "pid-sd-jwt,mdoc",
				},
			},
		},
		"credential offer uri": {
			text: offerDeeplink("credential_offer_uri", server.URL+"/offer"),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{credentialSource},
				Decoded: matchedOffer,
			},
		},
		"credential offer of another organization": {
			text:  offerDeeplink("credential_offer_uri", server.URL+"/offer"),
			orgID: "another-org",
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{},
				Decoded: matchedOffer,
				Note:    noOfferMatch,
			},
		},
		"credential offer of an unknown issuer": {
			text: offerDeeplink("credential_offer", offerJSON("https://unknown.test")),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{},
				Decoded: map[string]string{
					"credential_issuer":            "https://unknown.test",
					"credential_configuration_ids": "pid-sd-jwt,mdoc",
				},
				Note: noOfferMatch,
			},
		},
		"unreadable credential offer uri": {
			text: offerDeeplink("credential_offer_uri", server.URL+"/broken"),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{},
				Note:    "cannot read the offer: credential_offer_uri answered HTTP 404",
			},
		},
		"credential offer that is not JSON": {
			text: offerDeeplink("credential_offer", "{"),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{},
				Note: "cannot read the offer: the offer is not JSON: " +
					"unexpected end of JSON input",
			},
		},
		"presentation request": {
			text: "openid4vp://?client_id=x509_san_dns%3Averifier.test&request_uri=https%3A%2F%2Fverifier.test%2Fr",
			want: ReferenceResolution{
				Kind:    ReferencePresentation,
				Sources: []ReferenceSource{},
				Decoded: map[string]string{
					"client_id":   "x509_san_dns:verifier.test",
					"request_uri": "https://verifier.test/r",
				},
				Note: "Credimi records do not identify verifiers by client_id; replays need " +
					"a Credimi use case verification, or an HTTP call that creates a verifier session",
			},
		},
		"haip presentation request": {
			text: "haip-vp://?client_id=verifier",
			want: ReferenceResolution{
				Kind:    ReferencePresentation,
				Sources: []ReferenceSource{},
				Decoded: map[string]string{"client_id": "verifier"},
				Note: "Credimi records do not identify verifiers by client_id; replays need " +
					"a Credimi use case verification, or an HTTP call that creates a verifier session",
			},
		},
		"text": {
			text: "get a PID",
			want: ReferenceResolution{
				Kind:    ReferenceUnknown,
				Sources: []ReferenceSource{},
				Note:    "not a Credimi URL, identifier or deeplink",
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.orgID == "" {
				tc.orgID = orgID
			}
			got, err := ResolveReference(app, tc.orgID, tc.text, server.Client())
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
