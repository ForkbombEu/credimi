// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"errors"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestStoreCredentialIssuerActivityUpserts(t *testing.T) {
	app := newCredimiTestApp(t)
	act := NewStoreCredentialIssuerActivity(app)

	created, err := executeActivity(t, act, StoreCredentialIssuerInput{
		URL:   "https://issuer.example.com",
		OrgID: testOrgAID,
		Name:  "Issuer",
		Logo:  "https://issuer.example.com/logo.png",
	})
	require.NoError(t, err)
	output, ok := created.Output.(map[string]any)
	require.True(t, ok)
	issuerID, ok := output["id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, issuerID)

	issuer, err := app.FindRecordById("credential_issuers", issuerID)
	require.NoError(t, err)
	require.Equal(t, "Issuer", issuer.GetString("name"))
	require.Equal(t, "https://issuer.example.com/logo.png", issuer.GetString("logo_url"))
	require.True(t, issuer.GetBool("imported"))

	updated, err := executeActivity(t, act, StoreCredentialIssuerInput{
		URL:   "https://issuer.example.com",
		OrgID: testOrgAID,
		Name:  "Renamed Issuer",
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"id": issuerID}, updated.Output)

	issuer, err = app.FindRecordById("credential_issuers", issuerID)
	require.NoError(t, err)
	require.Equal(t, "Renamed Issuer", issuer.GetString("name"))
	require.Equal(t, "https://issuer.example.com/logo.png", issuer.GetString("logo_url"))
}

func TestStoreCredentialIssuerActivityRequiresURL(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(t, NewStoreCredentialIssuerActivity(app), StoreCredentialIssuerInput{
		OrgID: testOrgAID,
	})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}

func TestStoreCredentialIssuerActivitySaveFailure(t *testing.T) {
	app := newCredimiTestApp(t)
	app.OnRecordCreate("credential_issuers").BindFunc(func(*core.RecordEvent) error {
		return errors.New("save failed")
	})
	_, err := executeActivity(t, NewStoreCredentialIssuerActivity(app), StoreCredentialIssuerInput{
		URL:   "https://issuer.example.com",
		OrgID: testOrgAID,
	})
	requireActivityError(t, err, errorcodes.DatabaseOperationFailed, false)
}

func TestStoreIssuerCredentialActivityCreatesCredential(t *testing.T) {
	app := newCredimiTestApp(t)
	issuer := createTestIssuer(t, app, testOrgAID, "Issuer", "https://issuer.example.com")

	result, err := executeActivity(
		t,
		NewStoreIssuerCredentialActivity(app),
		StoreIssuerCredentialInput{
			IssuerID: issuer.Id,
			CredKey:  "cred-1",
			Credential: map[string]any{
				"format": "jwt",
				"display": []any{
					map[string]any{
						"name":        "Credential One",
						"description": "desc",
						"logo":        map[string]any{"uri": "https://logo.example.com/logo.png"},
					},
				},
			},
			Conformant: true,
			OrgID:      testOrgAID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"key": "cred-1"}, result.Output)

	record, err := app.FindFirstRecordByFilter(
		"credentials",
		"name = {:key} && credential_issuer = {:issuerID}",
		map[string]any{"key": "cred-1", "issuerID": issuer.Id},
	)
	require.NoError(t, err)
	require.Equal(t, "Credential One", record.GetString("display_name"))
	require.Equal(t, "jwt", record.GetString("format"))
	require.Equal(t, "https://logo.example.com/logo.png", record.GetString("logo_url"))
	require.True(t, record.GetBool("imported"))
	require.True(t, record.GetBool("conformant"))
}

func TestStoreIssuerCredentialActivityUpdatesExistingRecord(t *testing.T) {
	app := newCredimiTestApp(t)
	issuer := createTestIssuer(t, app, testOrgAID, "Issuer", "https://issuer.example.com")
	cred := createTestCredential(t, app, testOrgAID, issuer.Id, "cred-2")
	cred.Set("display_name", "Old Name")
	cred.Set("logo_url", "https://old.logo")
	cred.Set("json", `{"display":[{"name":"Old Name","logo":{"uri":"https://old.logo"}}]}`)
	require.NoError(t, app.Save(cred))

	_, err := executeActivity(t, NewStoreIssuerCredentialActivity(app), StoreIssuerCredentialInput{
		IssuerID: issuer.Id,
		CredKey:  "cred-2",
		Credential: map[string]any{
			"display": []any{
				map[string]any{
					"name": "New Name",
					"logo": map[string]any{"uri": "https://new.logo"},
				},
			},
		},
		OrgID: testOrgAID,
	})
	require.NoError(t, err)

	updated, err := app.FindRecordById("credentials", cred.Id)
	require.NoError(t, err)
	require.Equal(t, "New Name", updated.GetString("display_name"))
	require.Equal(t, "https://new.logo", updated.GetString("logo_url"))
}

func TestStoreIssuerCredentialActivityKeepsUserEditedDisplay(t *testing.T) {
	app := newCredimiTestApp(t)
	issuer := createTestIssuer(t, app, testOrgAID, "Issuer", "https://issuer.example.com")
	cred := createTestCredential(t, app, testOrgAID, issuer.Id, "cred-4")
	cred.Set("display_name", "User Name")
	cred.Set("logo_url", "https://user.logo")
	cred.Set("json", `{"display":[{"name":"Old Name","logo":{"uri":"https://old.logo"}}]}`)
	require.NoError(t, app.Save(cred))

	_, err := executeActivity(t, NewStoreIssuerCredentialActivity(app), StoreIssuerCredentialInput{
		IssuerID: issuer.Id,
		CredKey:  "cred-4",
		Credential: map[string]any{
			"display": []any{map[string]any{"name": "New Name"}},
		},
		OrgID: testOrgAID,
	})
	require.NoError(t, err)

	updated, err := app.FindRecordById("credentials", cred.Id)
	require.NoError(t, err)
	require.Equal(t, "User Name", updated.GetString("display_name"))
	require.Equal(t, "https://user.logo", updated.GetString("logo_url"))
}

func TestStoreIssuerCredentialActivityInvalidPayload(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(t, NewStoreIssuerCredentialActivity(app), StoreIssuerCredentialInput{
		CredKey: "cred-1",
		OrgID:   testOrgAID,
	})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}

func TestStoreIssuerCredentialActivityInvalidSavedJSON(t *testing.T) {
	app := newCredimiTestApp(t)
	issuer := createTestIssuer(t, app, testOrgAID, "Issuer", "https://issuer.example.com")
	cred := createTestCredential(t, app, testOrgAID, issuer.Id, "cred-3")
	cred.Set("display_name", "Old Name")
	cred.Set("logo_url", "https://old.logo")
	cred.Set("json", `not-json`)
	require.NoError(t, app.Save(cred))

	_, err := executeActivity(t, NewStoreIssuerCredentialActivity(app), StoreIssuerCredentialInput{
		IssuerID: issuer.Id,
		CredKey:  "cred-3",
		Credential: map[string]any{
			"display": []any{map[string]any{"name": "New Name"}},
		},
		Conformant: true,
		OrgID:      testOrgAID,
	})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, true)
}

func TestStoreIssuerCredentialActivitySaveFailure(t *testing.T) {
	app := newCredimiTestApp(t)
	issuer := createTestIssuer(t, app, testOrgAID, "Issuer", "https://issuer.example.com")
	app.OnRecordCreate("credentials").BindFunc(func(*core.RecordEvent) error {
		return errors.New("save failed")
	})
	_, err := executeActivity(t, NewStoreIssuerCredentialActivity(app), StoreIssuerCredentialInput{
		IssuerID:   issuer.Id,
		CredKey:    "cred-5",
		Credential: map[string]any{},
		OrgID:      testOrgAID,
	})
	requireActivityError(t, err, errorcodes.DatabaseOperationFailed, false)
}

func TestParseCredentialDisplay(t *testing.T) {
	cases := []struct {
		name       string
		input      map[string]any
		wantName   string
		wantLocale string
		wantLogo   string
		wantDesc   string
	}{
		{
			name: "full display with logo",
			input: map[string]any{
				"display": []any{
					map[string]any{
						"name":        "University Degree",
						"locale":      "en-US",
						"description": "A degree credential",
						"logo":        map[string]any{"uri": "https://example.com/logo.png"},
					},
				},
			},
			wantName:   "University Degree",
			wantLocale: "en-US",
			wantLogo:   "https://example.com/logo.png",
			wantDesc:   "A degree credential",
		},
		{
			name: "credential metadata display",
			input: map[string]any{
				"credential_metadata": map[string]any{
					"display": []any{
						map[string]any{
							"name":        "Mobile Driving Licence",
							"locale":      "en-US",
							"description": "ISO mDL credential",
							"logo": map[string]any{
								"uri": "https://example.com/mdl-logo.png",
							},
						},
					},
				},
			},
			wantName:   "Mobile Driving Licence",
			wantLocale: "en-US",
			wantLogo:   "https://example.com/mdl-logo.png",
			wantDesc:   "ISO mDL credential",
		},
		{
			name: "credential metadata display with url logo fallback",
			input: map[string]any{
				"credential_metadata": map[string]any{
					"display": []any{
						map[string]any{
							"name": "SD-JWT Credential",
							"logo": map[string]any{"url": "https://example.com/sd-jwt-logo.png"},
						},
					},
				},
			},
			wantName: "SD-JWT Credential",
			wantLogo: "https://example.com/sd-jwt-logo.png",
		},
		{
			name: "display without logo",
			input: map[string]any{
				"display": []any{
					map[string]any{
						"name":        "Simple Credential",
						"locale":      "en-GB",
						"description": "No logo provided",
					},
				},
			},
			wantName:   "Simple Credential",
			wantLocale: "en-GB",
			wantDesc:   "No logo provided",
		},
		{
			name: "display without description",
			input: map[string]any{
				"display": []any{map[string]any{"name": "Name Only", "locale": "fr-FR"}},
			},
			wantName:   "Name Only",
			wantLocale: "fr-FR",
		},
		{
			name:  "empty display list",
			input: map[string]any{"display": []any{}},
		},
		{
			name:  "missing display field",
			input: map[string]any{"format": "jwt_vc_json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, locale, logo, description := parseCredentialDisplay(tc.input)
			require.Equal(t, tc.wantName, name)
			require.Equal(t, tc.wantLocale, locale)
			require.Equal(t, tc.wantLogo, logo)
			require.Equal(t, tc.wantDesc, description)
		})
	}
}
