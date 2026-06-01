// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestInteropAxisRegistry_BuildEntity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		hubCollection string
		expectNil     bool
	}{
		{hubCollection: "wallets"},
		{hubCollection: "credentials"},
		{hubCollection: "use_cases_verifications"},
		{hubCollection: "credential_issuers"},
		{hubCollection: "verifiers"},
		{hubCollection: "conformance-checks", expectNil: true},
	}

	for _, tc := range cases {
		t.Run(tc.hubCollection, func(t *testing.T) {
			t.Parallel()

			axis, ok := getInteropAxis(tc.hubCollection)
			require.True(t, ok)
			if tc.expectNil {
				require.Nil(t, axis.buildEntity)
				return
			}
			require.NotNil(t, axis.buildEntity)
		})
	}
}

func TestCredentialBuildEntity(t *testing.T) {
	t.Parallel()

	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	issuersCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuer := core.NewRecord(issuersCollection)
	issuer.Set("url", "https://issuer.example.com")
	issuer.Set("name", "Resolver Issuer")
	issuer.Set("owner", orgID)
	issuer.Set("imported", true)
	issuer.Set("logo_url", "https://cdn.example.com/issuer.png")
	require.NoError(t, app.Save(issuer))

	credentialsCollection, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	credential := core.NewRecord(credentialsCollection)
	credential.Set("credential_issuer", issuer.Id)
	credential.Set("name", "resolver-credential")
	credential.Set("display_name", "Resolver Credential")
	credential.Set("json", `{}`)
	credential.Set("owner", orgID)
	require.NoError(t, app.Save(credential))

	axis, ok := getInteropAxis("credentials")
	require.True(t, ok)

	entity, err := axis.buildEntity(app, credential, nil)
	require.NoError(t, err)
	require.Equal(t, credential.Id, entity.ID)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, "Resolver Issuer", *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, "https://cdn.example.com/issuer.png", *entity.AvatarURL)
}

func TestUseCaseVerificationBuildEntity(t *testing.T) {
	t.Parallel()

	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	verifiersCollection, err := app.FindCollectionByNameOrId("verifiers")
	require.NoError(t, err)

	verifier := core.NewRecord(verifiersCollection)
	verifier.Set("url", "https://verifier.example.com")
	verifier.Set("name", "Resolver Verifier")
	verifier.Set("owner", orgID)
	verifier.Set("standard_and_version", "testsuite/draft-01")
	verifier.Set("format", []string{"SD-JWT"})
	verifier.Set("signing_algorithms", []string{"ES256"})
	verifier.Set("cryptographic_binding_methods", []string{"jwk"})
	verifier.Set("description", "example description")
	require.NoError(t, app.Save(verifier))

	useCasesCollection, err := app.FindCollectionByNameOrId("use_cases_verifications")
	require.NoError(t, err)

	useCase := core.NewRecord(useCasesCollection)
	useCase.Set("name", "resolver-usecase")
	useCase.Set("deeplink", "https://example.com/usecase")
	useCase.Set("yaml", "type: verification")
	useCase.Set("verifier", verifier.Id)
	useCase.Set("owner", orgID)
	require.NoError(t, app.Save(useCase))

	axis, ok := getInteropAxis("use_cases_verifications")
	require.True(t, ok)

	entity, err := axis.buildEntity(app, useCase, nil)
	require.NoError(t, err)
	require.Equal(t, useCase.Id, entity.ID)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, "Resolver Verifier", *entity.Subtitle)
}

func TestWalletBuildEntity_VersionLabelFromCacheRecord(t *testing.T) {
	t.Parallel()

	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-wallet")
	require.NoError(t, app.Save(wallet))

	versionsCollection, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)

	version := core.NewRecord(versionsCollection)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "2.0.0")
	version.Set("owner", orgID)
	require.NoError(t, app.Save(version))

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	cacheRecord := core.NewRecord(cacheCollection)
	cacheRecord.Set("wallets", []string{wallet.Id})
	cacheRecord.Set("wallet_versions", []string{version.Id})

	axis, ok := getInteropAxis("wallets")
	require.True(t, ok)

	entity, err := axis.buildEntity(app, wallet, cacheRecord)
	require.NoError(t, err)
	require.NotNil(t, entity.VersionLabel)
	require.Equal(t, "v2.0.0", *entity.VersionLabel)
}

func TestWalletBuildEntity_NoVersionLabelWithoutCacheRecord(t *testing.T) {
	t.Parallel()

	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-wallet")
	require.NoError(t, app.Save(wallet))

	axis, ok := getInteropAxis("wallets")
	require.True(t, ok)

	entity, err := axis.buildEntity(app, wallet, nil)
	require.NoError(t, err)
	require.Nil(t, entity.VersionLabel)
}
