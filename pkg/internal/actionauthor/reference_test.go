// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/pocketbase/dbx"
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
	// Another organization's unpublished credential of the same issuer URL and
	// name, whose issuer's organization no longer exists: building its path
	// would fail, so a match on it would error instead of skipping it.
	ghostIssuer := save("credential_issuers", map[string]any{
		"owner": orgID,
		"name":  "Ghost issuer",
		"url":   "https://issuer.reference.test",
	})
	save("credentials", map[string]any{
		"owner":             "ghostorganization",
		"credential_issuer": ghostIssuer.Id,
		"name":              "pid-sd-jwt",
	})
	_, err = app.DB().NewQuery(
		"UPDATE credential_issuers SET owner = 'ghostorganization' WHERE id = {:id}",
	).Bind(dbx.Params{"id": ghostIssuer.Id}).Execute()
	require.NoError(t, err)
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
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	sixteen := make([]string, 0, 17)
	for i := range 16 {
		sixteen = append(sixteen, fmt.Sprintf("config-%d", i))
	}
	encodedSixteen, err := json.Marshal(map[string]any{
		"credential_issuer":            "https://issuer.reference.test",
		"credential_configuration_ids": append(sixteen, "pid-sd-jwt"),
	})
	require.NoError(t, err)
	tooManyConfigurations := string(encodedSixteen)
	encodedSixteen, err = json.Marshal(map[string]any{
		"credential_issuer":            "https://issuer.reference.test",
		"credential_configuration_ids": append(sixteen[:15:15], "pid-sd-jwt"),
	})
	require.NoError(t, err)
	sixteenConfigurations := string(encodedSixteen)

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
		"credential offer with sixteen configurations": {
			text: offerDeeplink("credential_offer", sixteenConfigurations),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{credentialSource},
				Decoded: map[string]string{
					"credential_issuer": "https://issuer.reference.test",
					"credential_configuration_ids": "config-0,config-1,config-2,config-3," +
						"config-4,config-5,config-6,config-7,config-8,config-9,config-10," +
						"config-11,config-12,config-13,config-14,pid-sd-jwt",
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
		"plain http credential offer uri": {
			text: offerDeeplink("credential_offer_uri", "http://issuer.reference.test/offer"),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{},
				Note:    "cannot read the offer: credential_offer_uri must be https",
			},
		},
		"credential offer with too many configurations": {
			text: offerDeeplink("credential_offer", tooManyConfigurations),
			want: ReferenceResolution{
				Kind:    ReferenceCredentialOffer,
				Sources: []ReferenceSource{},
				Note:    "cannot read the offer: too many credential configurations",
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
			got, err := ResolveReference(t.Context(), app, tc.orgID, tc.text, server.Client())
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	// The fetch runs within the caller's context.
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := ResolveReference(
		canceled, app, orgID, offerDeeplink("credential_offer_uri", server.URL+"/offer"),
		server.Client(),
	)
	require.NoError(t, err)
	assert.Empty(t, got.Sources)
	assert.Contains(t, got.Note, "cannot read the offer: ")
	assert.Contains(t, got.Note, context.Canceled.Error())
}
