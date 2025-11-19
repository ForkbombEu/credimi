// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

type TestSetupData struct {
	IssuerID          string
	CreateIssuer      bool
	CreateCredentials []TestCredential
}

type TestCredential struct {
	Name        string
	IssuerID    string
	CredKey     string
	Format      string
	DisplayName string
	Locale      string
	Description string
	LogoURI     string
	Conformant  bool
}

func getOrgIDfromName(name string) (string, error) { //nolint
	app, err := tests.NewTestApp(testDataDir)
	if err != nil {
		return "", err
	}
	defer app.Cleanup()

	filter := fmt.Sprintf(`name="%s"`, name)

	record, err := app.FindFirstRecordByFilter("organizations", filter)
	if err != nil {
		return "", err
	}

	return record.Id, nil
}

func jsonBody(data map[string]any) *bytes.Reader {
	b, _ := json.Marshal(data)
	return bytes.NewReader(b)
}

func setupTestAppWithData(orgID string, setupData TestSetupData) func(t testing.TB) *tests.TestApp {
	return func(t testing.TB) *tests.TestApp {
		testApp, err := tests.NewTestApp(testDataDir)
		require.NoError(t, err)
		canonify.RegisterCanonifyHooks(testApp)
		IssuerTemporalInternalRoutes.Add(testApp)

		var issuerID string

		// Create issuer if needed
		if setupData.CreateIssuer {
			issuerCollection, err := testApp.FindCollectionByNameOrId("credential_issuers")
			require.NoError(t, err)

			issuerRecord := core.NewRecord(issuerCollection)
			if setupData.IssuerID != "" {
				issuerRecord.Set("id", setupData.IssuerID)
			}
			issuerRecord.Set("url", "https://test-issuer.example.com")
			issuerRecord.Set("name", "Test Issuer")
			issuerRecord.Set("owner", orgID)
			issuerRecord.Set("imported", true)
			err = testApp.Save(issuerRecord)
			require.NoError(t, err)
			issuerID = issuerRecord.Id
		}

		// Create credentials if needed
		if len(setupData.CreateCredentials) > 0 && issuerID != "" {
			credCollection, err := testApp.FindCollectionByNameOrId("credentials")
			require.NoError(t, err)

			for _, cred := range setupData.CreateCredentials {
				credRecord := core.NewRecord(credCollection)

				// Build credential JSON
				credJSON := map[string]any{
					"format": cred.Format,
					"display": []any{
						map[string]any{
							"name":        cred.DisplayName,
							"locale":      cred.Locale,
							"description": cred.Description,
						},
					},
				}

				// Add logo if provided
				if cred.LogoURI != "" {
					display := credJSON["display"].([]any)[0].(map[string]any)
					display["logo"] = map[string]any{
						"uri": cred.LogoURI,
					}
				}

				jsonBytes, _ := json.Marshal(credJSON)

				credRecord.Set("name", cred.CredKey)
				credRecord.Set("display_name", cred.DisplayName)
				credRecord.Set("locale", cred.Locale)
				credRecord.Set("description", cred.Description)
				credRecord.Set("logo", cred.LogoURI)
				credRecord.Set("format", cred.Format)
				credRecord.Set("json", string(jsonBytes))
				credRecord.Set("credential_issuer", issuerID)
				credRecord.Set("conformant", cred.Conformant)
				credRecord.Set("owner", orgID)

				err = testApp.Save(credRecord)
				require.NoError(t, err)
			}
		}

		return testApp
	}
}

func TestCredentialIssuersAPI(t *testing.T) {
	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	scenarios := []tests.ApiScenario{
		{
			Name:   "store new credential",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "university-degree",
				"credential": map[string]any{
					"format": "jwt_vc_json",
					"display": []any{
						map[string]any{
							"name":        "University Degree",
							"locale":      "en-US",
							"description": "A university degree credential",
							"logo": map[string]any{
								"uri": "https://example.com/logo.png",
							},
						},
					},
				},
				"conformant": true,
				"orgID":      orgID,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"key":"university-degree"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
				IssuerID:     "issuerid1234567",
			}),
		},
		{
			Name:   "update existing credential",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "university-degree",
				"credential": map[string]any{
					"format": "jwt_vc_json",
					"display": []any{
						map[string]any{
							"name":        "University Degree",
							"locale":      "en-US",
							"description": "An updated university degree credential",
							"logo": map[string]any{
								"uri": "https://example.com/new-logo.png",
							},
						},
					},
				},
				"conformant": true,
				"orgID":      orgID,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"key":"university-degree"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Description: "A university degree credential",
						LogoURI:     "https://example.com/logo.png",
						Conformant:  true,
					},
				},
			}),
		},
		{
			Name:   "store credential without logo",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "simple-credential",
				"credential": map[string]any{
					"format": "vc+sd-jwt",
					"display": []any{
						map[string]any{
							"name":   "Simple Credential",
							"locale": "en-US",
						},
					},
				},
				"conformant": false,
				"orgID":      orgID,
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"key":"simple-credential"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
			}),
		},
		{
			Name:   "cleanup credentials - delete stale ones",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/cleanup-credentials",
			Body: jsonBody(map[string]any{
				"issuerID":  "issuerid1234567",
				"validKeys": []string{"university-degree"},
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"deleted"`,
				`"simple-credential"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Conformant:  true,
					},
					{
						CredKey:     "simple-credential",
						Format:      "vc+sd-jwt",
						DisplayName: "Simple Credential",
						Locale:      "en-US",
						Conformant:  false,
					},
				},
			}),
		},
		{
			Name:   "cleanup credentials - no deletions when all valid",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/cleanup-credentials",
			Body: jsonBody(map[string]any{
				"issuerID":  "issuerid1234567",
				"validKeys": []string{"university-degree"},
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"deleted":null`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{IssuerID: "issuerid1234567",
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Conformant:  true,
					},
				},
			}),
		},
		{
			Name:   "cleanup credentials - delete all",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/cleanup-credentials",
			Body: jsonBody(map[string]any{
				"issuerID":  "issuerid1234567",
				"validKeys": []string{},
			}),
			ExpectedStatus: 200,
			ExpectedContent: []string{
				`"deleted"`,
				`"university-degree"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				IssuerID:     "issuerid1234567",
				CreateIssuer: true,
				CreateCredentials: []TestCredential{
					{
						IssuerID:    "issuerid1234567",
						CredKey:     "university-degree",
						Format:      "jwt_vc_json",
						DisplayName: "University Degree",
						Locale:      "en-US",
						Conformant:  true,
					},
				},
			}),
		},
		{
			Name:   "store credential with wrong orgID",
			Method: http.MethodPost,
			URL:    "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body: jsonBody(map[string]any{
				"issuerID": "issuerid1234567",
				"credKey":  "invalid-org-credential",
				"credential": map[string]any{
					"format": "jwt_vc_json",
					"display": []any{
						map[string]any{
							"name":        "Invalid Org Credential",
							"locale":      "en-US",
							"description": "Should fail because of wrong orgID",
						},
					},
				},
				"conformant": true,
				"orgID":      "nonexistent-org-id", // <-- invalid orgID
			}),
			ExpectedStatus: 500,
			ExpectedContent: []string{
				`"credentials"`,
				`"failed to save credentials"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
				IssuerID:     "issuerid1234567",
			}),
		},
		{
			Name:           "store credential with invalid JSON body",
			Method:         http.MethodPost,
			URL:            "/api/credentials_issuers/store-or-update-extracted-credentials",
			Body:           bytes.NewReader([]byte(`{invalid json}`)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"reason":"Invalid JSON format for the expected type"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
			}),
		},
		{
			Name:           "cleanup credentials with invalid JSON body",
			Method:         http.MethodPost,
			URL:            "/api/credentials_issuers/cleanup-credentials",
			Body:           bytes.NewReader([]byte(`{invalid json}`)),
			ExpectedStatus: 400,
			ExpectedContent: []string{
				`"reason":"Invalid JSON format for the expected type"`,
			},
			TestAppFactory: setupTestAppWithData(orgID, TestSetupData{
				CreateIssuer: true,
			}),
		},
	}

	for _, scenario := range scenarios {
		scenario.Test(t)
	}
}

func TestParseCredentialDisplay(t *testing.T) {
	tests := []struct {
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
						"logo": map[string]any{
							"uri": "https://example.com/logo.png",
						},
					},
				},
			},
			wantName:   "University Degree",
			wantLocale: "en-US",
			wantLogo:   "https://example.com/logo.png",
			wantDesc:   "A degree credential",
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
			wantLogo:   "",
			wantDesc:   "No logo provided",
		},
		{
			name: "display without description",
			input: map[string]any{
				"display": []any{
					map[string]any{
						"name":   "Name Only",
						"locale": "fr-FR",
					},
				},
			},
			wantName:   "Name Only",
			wantLocale: "fr-FR",
			wantLogo:   "",
			wantDesc:   "",
		},
		{
			name: "empty display list",
			input: map[string]any{
				"display": []any{},
			},
			wantName:   "",
			wantLocale: "",
			wantLogo:   "",
			wantDesc:   "",
		},
		{
			name: "missing display field",
			input: map[string]any{
				"format": "jwt_vc_json",
			},
			wantName:   "",
			wantLocale: "",
			wantLogo:   "",
			wantDesc:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotLocale, gotLogo, gotDesc := parseCredentialDisplay(tt.input)

			require.Equal(t, tt.wantName, gotName, "name mismatch")
			require.Equal(t, tt.wantLocale, gotLocale, "locale mismatch")
			require.Equal(t, tt.wantLogo, gotLogo, "logo mismatch")
			require.Equal(t, tt.wantDesc, gotDesc, "description mismatch")
		})
	}
}
