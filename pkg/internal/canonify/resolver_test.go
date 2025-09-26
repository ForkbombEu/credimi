// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package canonify

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

type RecordSetup struct {
	Collection string
	Fields     map[string]any
}
type TestExpected struct {
	Result string
	Error  bool
}

type BuildPathTestCase struct {
	Name     string
	Setup    []RecordSetup
	Input    RecordSetup
	Expected TestExpected
}
type ResolveTestCase struct {
	Name       string
	Setup      []RecordSetup
	Collection string
	Path       string
	Expected   TestExpected
}

// createRecord creates a single record
func createRecord(t *testing.T, app core.App, setup RecordSetup) *core.Record {
	col, err := app.FindCollectionByNameOrId(setup.Collection)
	require.NoError(t, err, "failed to find collection %s", setup.Collection)

	rec := core.NewRecord(col)
	for k, v := range setup.Fields {
		rec.Set(k, v)
	}

	return rec
}

func createRecordAndSave(t *testing.T, app core.App, setup RecordSetup) {
	rec := createRecord(t, app, setup)
	require.NoError(t, app.Save(rec), "failed to save record in %s", setup.Collection)
}

func TestBuildPath(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err, "failed to create test app")

	testCases := []BuildPathTestCase{
		{
			Name:  "organization root",
			Setup: []RecordSetup{},
			Input: RecordSetup{
				Collection: "organizations",
				Fields: map[string]any{
					"name":            "OrgA",
					"canonified_name": "orga",
				},
			},
			Expected: TestExpected{
				Result: "/orga",
				Error:  false,
			},
		},
		{
			Name: "issuer under org",
			Setup: []RecordSetup{
				{
					Collection: "organizations",
					Fields: map[string]any{
						"id":              "orgbid123456789",
						"name":            "OrgB",
						"canonified_name": "orgb",
					},
				},
			},
			Input: RecordSetup{
				Collection: "credential_issuers",
				Fields: map[string]any{
					"name":            "IssuerX",
					"canonified_name": "issuerx",
					"url":             "https://issuerx.com",
					"owner":           "orgbid123456789",
				},
			},
			Expected: TestExpected{
				Result: "/orgb/issuerx",
				Error:  false,
			},
		},
		{
			Name: "complex hierarchy",
			Setup: []RecordSetup{
				{
					Collection: "organizations",
					Fields: map[string]any{
						"id":              "orgcid123456789",
						"name":            "OrgC",
						"canonified_name": "orgc",
					},
				},
				{
					Collection: "credential_issuers",
					Fields: map[string]any{
						"id":              "issueryid123456",
						"name":            "IssuerY",
						"canonified_name": "issuery",
						"url":             "https://issuery.com",
						"owner":           "orgcid123456789",
					},
				},
			},
			Input: RecordSetup{
				Collection: "credentials",
				Fields: map[string]any{
					"name":              "Cred Test",
					"canonified_name":   "cred-test",
					"credential_issuer": "issueryid123456",
					"owner":             "orgcid123456789",
				},
			},
			Expected: TestExpected{
				Result: "/orgc/issuery/cred-test",
				Error:  false,
			},
		},
		{
			Name:  "missing parent field",
			Setup: []RecordSetup{},
			Input: RecordSetup{
				Collection: "credential_issuers",
				Fields: map[string]any{
					"name":            "IssuerWithoutOwner",
					"canonified_name": "issuerwithoutowner",
					"url":             "https://issuer.com",
				},
			},
			Expected: TestExpected{
				Error: true,
			},
		},
		{
			Name:  "no path template for parent collection",
			Setup: []RecordSetup{},
			Input: RecordSetup{
				Collection: "credential_issuers",
				Fields: map[string]any{
					"name":            "IssuerBadParent",
					"canonified_name": "issuerbadparent",
					"url":             "https://issuer.com",
					"owner":           "nonexistent123",
				},
			},
			Expected: TestExpected{
				Error: true,
			},
		},
		{
			Name:  "parent record not found",
			Setup: []RecordSetup{},
			Input: RecordSetup{
				Collection: "credential_issuers",
				Fields: map[string]any{
					"name":            "IssuerMissingParent",
					"canonified_name": "issuermissingparent",
					"url":             "https://issuer.com",
					"owner":           "nonexistentparent123",
				},
			},
			Expected: TestExpected{
				Error: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Create setup records
			for _, setup := range tc.Setup {
				createRecordAndSave(t, app, setup)
			}

			// Create input record
			inputRec := createRecord(t, app, tc.Input)

			// Test BuildPath
			tpl := CanonifyPaths[tc.Input.Collection]
			got, err := BuildPath(app, inputRec, tpl, tc.Input.Fields["canonified_name"].(string))

			if tc.Expected.Error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.Expected.Result, got)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err, "failed to create test app")

	testCases := []ResolveTestCase{
		{
			Name: "resolve organization",
			Setup: []RecordSetup{
				{
					Collection: "organizations",
					Fields: map[string]any{
						"id":              "orgaid123456789",
						"name":            "OrgA",
						"canonified_name": "orga",
					},
				},
			},
			Collection: "organizations",
			Path:       "/orga",
			Expected: TestExpected{
				Result: "orgaid123456789",
				Error:  false,
			},
		},
		{
			Name: "resolve issuer",
			Setup: []RecordSetup{
				{
					Collection: "organizations",
					Fields: map[string]any{
						"id":              "orgbid123456789",
						"name":            "OrgB",
						"canonified_name": "orgb",
					},
				},
				{
					Collection: "credential_issuers",
					Fields: map[string]any{
						"id":              "issuerxid123456",
						"name":            "IssuerX",
						"canonified_name": "issuerx",
						"url":             "https://issuerx.com",
						"owner":           "orgbid123456789",
					},
				},
			},
			Collection: "credential_issuers",
			Path:       "/orgb/issuerx",
			Expected: TestExpected{
				Result: "issuerxid123456",
				Error:  false,
			},
		},
		{
			Name: "complex hierarchy",
			Setup: []RecordSetup{
				{
					Collection: "organizations",
					Fields: map[string]any{
						"id":              "orgcid123456789",
						"name":            "OrgC",
						"canonified_name": "orgc",
					},
				},
				{
					Collection: "credential_issuers",
					Fields: map[string]any{
						"id":              "issueryid123456",
						"name":            "IssuerY",
						"canonified_name": "issuery",
						"url":             "https://issuery.com",
						"owner":           "orgcid123456789",
					},
				},
				{
					Collection: "credentials",
					Fields: map[string]any{
						"id":                "credid123456789",
						"name":              "Cred Test",
						"canonified_name":   "cred-test",
						"credential_issuer": "issueryid123456",
						"owner":             "orgcid123456789",
					},
				},
			},
			Collection: "credentials",
			Path:       "/orgc/issuery/cred-test",
			Expected: TestExpected{
				Result: "credid123456789",
				Error:  false,
			},
		},
		{
			Name:       "nonexistent path",
			Setup:      []RecordSetup{},
			Collection: "credential_issuers",
			Path:       "/orgb/Unknown",
			Expected: TestExpected{
				Error: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// Create setup records
			for _, setup := range tc.Setup {
				createRecordAndSave(t, app, setup)
			}

			// Test Resolve
			got, err := Resolve(app, tc.Collection, tc.Path)

			if tc.Expected.Error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.Expected.Result, got.Id)
			}
		})
	}
}
