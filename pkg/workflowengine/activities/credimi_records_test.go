// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package activities

import (
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/errorcodes"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/require"
)

const resolvePrivateYAML = "DUMMY_PRIVATE_YAML"

type resolveFixture struct {
	ownerNamespace        string
	privatePipelinePath   string
	publicPipelinePath    string
	privateCredentialPath string
	publicCredentialPath  string
}

// setupResolveFixture seeds userB's organization with an unpublished and a
// published pipeline and an unpublished and a published credential.
func setupResolveFixture(t *testing.T, app *tests.TestApp) resolveFixture {
	t.Helper()
	org, err := app.FindRecordById("organizations", testOrgBID)
	require.NoError(t, err)
	orgPath := org.GetString("canonified_name")

	newPipeline := func(name string, published bool) string {
		coll, err := app.FindCollectionByNameOrId("pipelines")
		require.NoError(t, err)
		pipeline := core.NewRecord(coll)
		pipeline.Set("owner", org.Id)
		pipeline.Set("name", name)
		pipeline.Set("description", name)
		pipeline.Set("yaml", resolvePrivateYAML)
		pipeline.Set("published", published)
		require.NoError(t, app.Save(pipeline))
		return orgPath + "/" + pipeline.GetString("canonified_name")
	}
	newCredential := func(issuerName, credentialName string, published bool) string {
		issuer := createTestIssuer(t, app, org.Id, issuerName, "https://"+issuerName+".example")
		issuer.Set("published", published)
		require.NoError(t, app.Save(issuer))
		credential := createTestCredential(t, app, org.Id, issuer.Id, credentialName)
		credential.Set("published", published)
		require.NoError(t, app.Save(credential))
		return orgPath + "/" + issuer.GetString("canonified_name") + "/" +
			credential.GetString("canonified_name")
	}

	return resolveFixture{
		ownerNamespace:        orgPath,
		privatePipelinePath:   newPipeline("privpipe", false),
		publicPipelinePath:    newPipeline("pubpipe", true),
		privateCredentialPath: newCredential("iss", "cred", false),
		publicCredentialPath:  newCredential("pubiss", "pubcred", true),
	}
}

func createTestIssuer(t *testing.T, app *tests.TestApp, ownerID, name, url string) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)
	issuer := core.NewRecord(coll)
	issuer.Set("owner", ownerID)
	issuer.Set("name", name)
	issuer.Set("url", url)
	require.NoError(t, app.Save(issuer))
	return issuer
}

func createTestCredential(
	t *testing.T,
	app *tests.TestApp,
	ownerID string,
	issuerID string,
	name string,
) *core.Record {
	t.Helper()
	coll, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)
	credential := core.NewRecord(coll)
	credential.Set("owner", ownerID)
	credential.Set("credential_issuer", issuerID)
	credential.Set("name", name)
	require.NoError(t, app.Save(credential))
	return credential
}

func TestResolveRecordActivity(t *testing.T) {
	cases := []struct {
		name       string
		path       func(resolveFixture) string
		collection string
		namespace  func(resolveFixture) string
		wantErr    string
		wantYAML   bool
	}{
		{
			name:       "owner organization reads its unpublished pipeline",
			path:       func(f resolveFixture) string { return f.privatePipelinePath },
			collection: "pipelines",
			namespace:  func(f resolveFixture) string { return f.ownerNamespace },
			wantYAML:   true,
		},
		{
			name:       "another organization reads a published pipeline",
			path:       func(f resolveFixture) string { return f.publicPipelinePath },
			collection: "pipelines",
			namespace:  func(resolveFixture) string { return testOrgANamespace },
			wantYAML:   true,
		},
		{
			name:       "another organization cannot read an unpublished pipeline",
			path:       func(f resolveFixture) string { return f.privatePipelinePath },
			collection: "pipelines",
			namespace:  func(resolveFixture) string { return testOrgANamespace },
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "unknown owner namespace cannot read an unpublished pipeline",
			path:       func(f resolveFixture) string { return f.privatePipelinePath },
			collection: "pipelines",
			namespace:  func(resolveFixture) string { return "missing-org" },
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "missing record",
			path:       func(f resolveFixture) string { return f.ownerNamespace + "/missing" },
			collection: "pipelines",
			namespace:  func(f resolveFixture) string { return f.ownerNamespace },
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "a credential path never resolves as a wallet action",
			path:       func(f resolveFixture) string { return f.privateCredentialPath },
			collection: "wallet_actions",
			namespace:  func(f resolveFixture) string { return f.ownerNamespace },
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "a published credential path never resolves as a wallet action",
			path:       func(f resolveFixture) string { return f.publicCredentialPath },
			collection: "wallet_actions",
			namespace:  func(resolveFixture) string { return testOrgANamespace },
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "credentials cannot be requested",
			path:       func(f resolveFixture) string { return f.privateCredentialPath },
			collection: "credentials",
			namespace:  func(f resolveFixture) string { return f.ownerNamespace },
			wantErr:    errorcodes.MissingOrInvalidPayload,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			fx := setupResolveFixture(t, app)

			result, err := executeActivity(t, NewResolveRecordActivity(app), ResolveRecordInput{
				CanonifiedName: tc.path(fx),
				Collection:     tc.collection,
				OwnerNamespace: tc.namespace(fx),
			})
			if tc.wantErr != "" {
				requireActivityError(t, err, tc.wantErr, tc.wantErr == errorcodes.RecordNotFound)
				require.NotContains(t, err.Error(), resolvePrivateYAML)
				return
			}
			require.NoError(t, err)
			output, ok := result.Output.(map[string]any)
			require.True(t, ok)
			require.Equal(t, resolvePrivateYAML, output["yaml"])
		})
	}
}

func TestGetCredentialOfferActivity(t *testing.T) {
	cases := []struct {
		name        string
		seed        func(t *testing.T, app *tests.TestApp) *core.Record
		identifier  string
		wantErr     string
		wantOutput  map[string]any
		wantSecrets map[string]any
	}{
		{
			name:       "missing credential",
			identifier: "nonexistent",
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "credential with deeplink",
			identifier: testOrgANamespace + "/issuer-123/cred123",
			seed: func(t *testing.T, app *tests.TestApp) *core.Record {
				issuer := createTestIssuer(
					t,
					app,
					testOrgAID,
					"Issuer 123",
					"https://issuer.example",
				)
				credential := createTestCredential(t, app, testOrgAID, issuer.Id, "cred123")
				credential.Set("deeplink", "https://deeplink.example/offer")
				return credential
			},
			wantOutput: map[string]any{
				"credential_offer": "https://deeplink.example/offer",
				"dynamic":          false,
			},
		},
		{
			name:       "credential without deeplink builds a credential offer URI",
			identifier: testOrgANamespace + "/issuer-456/cred456",
			seed: func(t *testing.T, app *tests.TestApp) *core.Record {
				issuer := createTestIssuer(
					t,
					app,
					testOrgAID,
					"Issuer 456",
					"https://issuer.example",
				)
				return createTestCredential(t, app, testOrgAID, issuer.Id, "cred456")
			},
			wantOutput: map[string]any{
				"credential_offer": "openid-credential-offer://?credential_offer=%7B%22credential_configuration_ids%22%3A%5B%22cred456%22%5D%2C%22credential_issuer%22%3A%22https%3A%2F%2Fissuer.example%22%7D",
				"dynamic":          false,
			},
		},
		{
			name:       "dynamic credential with code and secrets",
			identifier: testOrgANamespace + "/issuer-789/cred789",
			seed: func(t *testing.T, app *tests.TestApp) *core.Record {
				issuer := createTestIssuer(
					t,
					app,
					testOrgAID,
					"Issuer 789",
					"https://issuer.example",
				)
				credential := createTestCredential(t, app, testOrgAID, issuer.Id, "cred789")
				credential.Set("yaml", "print('hello world')")
				credential.Set("secrets", "token: credential-secret\n")
				return credential
			},
			wantOutput: map[string]any{
				"dynamic": true,
				"code":    "print('hello world')",
			},
			wantSecrets: map[string]any{"token": "credential-secret"},
		},
		{
			name:       "credential with empty code is not dynamic",
			identifier: testOrgANamespace + "/issuer-987/cred987",
			seed: func(t *testing.T, app *tests.TestApp) *core.Record {
				issuer := createTestIssuer(
					t,
					app,
					testOrgAID,
					"Issuer 987",
					"https://issuer.example",
				)
				credential := createTestCredential(t, app, testOrgAID, issuer.Id, "cred987")
				credential.Set("yaml", "")
				return credential
			},
			wantOutput: map[string]any{
				"credential_offer": "openid-credential-offer://?credential_offer=%7B%22credential_configuration_ids%22%3A%5B%22cred987%22%5D%2C%22credential_issuer%22%3A%22https%3A%2F%2Fissuer.example%22%7D",
				"dynamic":          false,
			},
		},
		{
			name:       "invalid secrets yaml",
			identifier: testOrgANamespace + "/issuer-555/cred555",
			seed: func(t *testing.T, app *tests.TestApp) *core.Record {
				issuer := createTestIssuer(
					t,
					app,
					testOrgAID,
					"Issuer 555",
					"https://issuer.example",
				)
				credential := createTestCredential(t, app, testOrgAID, issuer.Id, "cred555")
				credential.Set("yaml", "code")
				credential.Set("secrets", "token: [unterminated")
				return credential
			},
			wantErr: errorcodes.DecodeFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			if tc.seed != nil {
				require.NoError(t, app.Save(tc.seed(t, app)))
			}

			result, err := executeActivity(
				t,
				NewGetCredentialOfferActivity(app),
				GetCredentialOfferInput{
					CredentialIdentifier: tc.identifier,
				},
			)
			if tc.wantErr != "" {
				requireActivityError(t, err, tc.wantErr, true)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantOutput, result.Output)
			require.Equal(t, tc.wantSecrets, result.Secrets)
		})
	}
}

func TestGetCredentialOfferActivityRequiresIdentifier(t *testing.T) {
	app := newCredimiTestApp(t)
	_, err := executeActivity(t, NewGetCredentialOfferActivity(app), GetCredentialOfferInput{})
	requireActivityError(t, err, errorcodes.MissingOrInvalidPayload, false)
}

func TestGetUseCaseVerificationDeeplinkActivity(t *testing.T) {
	seedUseCase := func(t *testing.T, app *tests.TestApp, name, secrets string) {
		t.Helper()
		verifierColl, err := app.FindCollectionByNameOrId("verifiers")
		require.NoError(t, err)
		verifier := core.NewRecord(verifierColl)
		verifier.Set("owner", testOrgAID)
		verifier.Set("name", "Verifier 123")
		verifier.Set("url", "https://verifier.example")
		verifier.Set("standard_and_version", "testsuite/draft-01")
		verifier.Set("format", []string{"SD-JWT"})
		verifier.Set("signing_algorithms", []string{"ES256"})
		verifier.Set("cryptographic_binding_methods", []string{"jwk"})
		verifier.Set("description", "example description")
		require.NoError(t, app.Save(verifier))

		coll, err := app.FindCollectionByNameOrId("use_cases_verifications")
		require.NoError(t, err)
		record := core.NewRecord(coll)
		record.Set("name", name)
		record.Set("owner", testOrgAID)
		record.Set("verifier", verifier.Id)
		record.Set("yaml", "example code")
		record.Set("secrets", secrets)
		require.NoError(t, app.Save(record))
	}

	cases := []struct {
		name        string
		secrets     string
		identifier  string
		wantErr     string
		wantSecrets map[string]any
	}{
		{
			name:        "valid use case verification identifier",
			secrets:     "pin: '1234'\n",
			identifier:  testOrgANamespace + "/verifier-123/usecase123",
			wantSecrets: map[string]any{"pin": "1234"},
		},
		{
			name:       "nonexistent use case identifier",
			identifier: "nonexistent",
			wantErr:    errorcodes.RecordNotFound,
		},
		{
			name:       "invalid secrets yaml",
			secrets:    "pin: [unterminated",
			identifier: testOrgANamespace + "/verifier-123/usecase123",
			wantErr:    errorcodes.DecodeFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := newCredimiTestApp(t)
			seedUseCase(t, app, "usecase123", tc.secrets)

			result, err := executeActivity(
				t,
				NewGetUseCaseVerificationDeeplinkActivity(app),
				GetUseCaseVerificationDeeplinkInput{UseCaseIdentifier: tc.identifier},
			)
			if tc.wantErr != "" {
				requireActivityError(t, err, tc.wantErr, true)
				return
			}
			require.NoError(t, err)
			require.Equal(t, map[string]any{"code": "example code"}, result.Output)
			require.Equal(t, tc.wantSecrets, result.Secrets)
		})
	}
}
