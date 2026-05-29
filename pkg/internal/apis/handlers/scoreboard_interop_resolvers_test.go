// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/stretchr/testify/require"
)

func TestGetInteropEntityResolver_UnknownCollection(t *testing.T) {
	t.Parallel()

	_, err := getInteropEntityResolver("unknown_collection")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no interop entity resolver")
}

func TestInteropEntityResolverRegistry_Collections(t *testing.T) {
	t.Parallel()

	cases := []struct {
		collection            string
		relatedCollections    []string
		supportsVersionLabels bool
	}{
		{
			collection:            "wallets",
			supportsVersionLabels: true,
		},
		{
			collection:         "credentials",
			relatedCollections: []string{"credential_issuers"},
		},
		{
			collection:         "use_cases_verifications",
			relatedCollections: []string{"verifiers"},
		},
		{
			collection: "credential_issuers",
		},
		{
			collection: "verifiers",
		},
	}

	for _, tc := range cases {
		t.Run(tc.collection, func(t *testing.T) {
			t.Parallel()

			resolver, err := getInteropEntityResolver(tc.collection)
			require.NoError(t, err)
			require.Equal(t, tc.collection, resolver.Collection())
			require.Equal(t, tc.supportsVersionLabels, resolver.SupportsVersionLabels())

			specs := resolver.RelatedCollections()
			require.Len(t, specs, len(tc.relatedCollections))
			for i, relatedCollection := range tc.relatedCollections {
				require.Equal(t, relatedCollection, specs[i].Collection)
				require.NotEmpty(t, specs[i].Field)
			}
		})
	}
}

func TestCredentialsInteropEntityResolver_Entity(t *testing.T) {
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

	resolver, err := getInteropEntityResolver("credentials")
	require.NoError(t, err)

	related := interopRelatedRecords{
		byCollection: map[string]map[string]*core.Record{
			"credential_issuers": {issuer.Id: issuer},
		},
	}

	entity, err := resolver.Entity(app, credential, related)
	require.NoError(t, err)
	require.Equal(t, credential.Id, entity.ID)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, "Resolver Issuer", *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, "https://cdn.example.com/issuer.png", *entity.AvatarURL)
}

func TestUseCasesVerificationsInteropEntityResolver_Entity(t *testing.T) {
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

	// Logo is a file field; set in-memory only for resolver reads.
	verifier.Set("logo", "https://cdn.example.com/verifier.png")

	resolver, err := getInteropEntityResolver("use_cases_verifications")
	require.NoError(t, err)

	related := interopRelatedRecords{
		byCollection: map[string]map[string]*core.Record{
			"verifiers": {verifier.Id: verifier},
		},
	}

	entity, err := resolver.Entity(app, useCase, related)
	require.NoError(t, err)
	require.Equal(t, useCase.Id, entity.ID)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, "Resolver Verifier", *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, "https://cdn.example.com/verifier.png", *entity.AvatarURL)
}

func TestWalletsInteropEntityResolver_SupportsVersionLabels(t *testing.T) {
	t.Parallel()

	resolver, err := getInteropEntityResolver("wallets")
	require.NoError(t, err)
	require.True(t, resolver.SupportsVersionLabels())
}
