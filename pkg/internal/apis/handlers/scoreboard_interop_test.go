// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/router"
	"github.com/stretchr/testify/require"
)

func ptrTo[T any](v T) *T { return &v }

func setupScoreboardInteropApp(t testing.TB) *tests.TestApp {
	t.Helper()
	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	canonify.RegisterCanonifyHooks(app)
	return app
}

func findInteropCell(resp InteropMatrixResponse, rowID, colID string) (InteropMatrixCell, bool) {
	for _, c := range resp.Cells {
		if c.RowID == rowID && c.ColumnID == colID {
			return c, true
		}
	}
	return InteropMatrixCell{}, false
}

func TestBuildInteropMatrix_CartesianAndSums(t *testing.T) {
	t.Parallel()

	const (
		w1 = "wallet1"
		w2 = "wallet2"
		i1 = "issuer1"
		p1 = "pipeline_one"
		p2 = "pipeline_two"
	)
	rowEntities := map[string]InteropMatrixEntity{
		w1: {ID: w1, Name: "Wallet B", Path: "/w/b"},
		w2: {ID: w2, Name: "Wallet A", Path: "/w/a"},
	}
	colEntities := map[string]InteropMatrixEntity{
		i1: {ID: i1, Name: "Issuer One", Path: "/issuers/one"},
	}
	inputs := []interopCacheInput{
		{PipelineID: p1, TotalRuns: 92, TotalSuccesses: 78, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
		{PipelineID: p2, TotalRuns: 92, TotalSuccesses: 78, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: 60, TotalSuccesses: 53, RowIDs: []string{w2}, ColumnIDs: []string{i1}},
	}

	got := buildInteropMatrix(inputs, rowEntities, colEntities)

	require.Equal(t, interopModeWalletsIssuers, got.Mode)
	require.Equal(t, "wallet", got.RowAxis)
	require.Equal(t, "issuer", got.ColumnAxis)
	require.Len(t, got.Cells, 2)

	require.Len(t, got.Rows, 2)
	require.Equal(t, w2, got.Rows[0].ID)
	require.Equal(t, w1, got.Rows[1].ID)

	require.Len(t, got.Columns, 1)
	require.Equal(t, i1, got.Columns[0].ID)

	cellW1I1, ok := findInteropCell(got, w1, i1)
	require.True(t, ok)
	require.Equal(t, 2, cellW1I1.PipelineCount)
	require.Equal(t, 184, cellW1I1.TotalRuns)
	require.Equal(t, 156, cellW1I1.TotalSuccesses)
	require.Equal(t, interopStatusFlaky, cellW1I1.Status)
	expectedRate := 156.0 / 184.0 * 100
	require.InDelta(t, expectedRate, cellW1I1.SuccessRate, 1e-9)

	cellW2I1, ok := findInteropCell(got, w2, i1)
	require.True(t, ok)
	require.Equal(t, 1, cellW2I1.PipelineCount)
	require.Equal(t, 60, cellW2I1.TotalRuns)
	require.Equal(t, 53, cellW2I1.TotalSuccesses)
	require.Equal(t, interopStatusFlaky, cellW2I1.Status)
	require.InDelta(t, 53.0/60.0*100, cellW2I1.SuccessRate, 1e-9)
}

func TestBuildInteropMatrix_SkipsEmptySides(t *testing.T) {
	t.Parallel()

	const (
		w1 = "wallet1"
		i1 = "issuer1"
		p1 = "pipeline_one"
	)
	rowEntities := map[string]InteropMatrixEntity{w1: {ID: w1, Name: "Wal", Path: "/w"}}
	colEntities := map[string]InteropMatrixEntity{i1: {ID: i1, Name: "Iss", Path: "/i"}}

	skippedRuns := math.MaxInt
	inputs := []interopCacheInput{
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: nil, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: []string{}, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: []string{w1}, ColumnIDs: nil},
		{PipelineID: p1, TotalRuns: skippedRuns, TotalSuccesses: 1, RowIDs: []string{w1}, ColumnIDs: []string{}},
		{PipelineID: p1, TotalRuns: 0, TotalSuccesses: 0, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
		{PipelineID: p1, TotalRuns: 10, TotalSuccesses: 9, RowIDs: []string{w1}, ColumnIDs: []string{i1}},
	}

	got := buildInteropMatrix(inputs, rowEntities, colEntities)

	require.Len(t, got.Cells, 1)
	c, ok := findInteropCell(got, w1, i1)
	require.True(t, ok)
	require.Equal(t, 1, c.PipelineCount)
	require.Equal(t, 10, c.TotalRuns)
	require.Equal(t, 9, c.TotalSuccesses)
}

func TestInteropStatusFromRate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rate float64
		want interopStatus
	}{
		{rate: 90, want: interopStatusStable},
		{rate: 89.9, want: interopStatusFlaky},
		{rate: 70, want: interopStatusFlaky},
		{rate: 69.9, want: interopStatusFailing},
		{rate: 50, want: interopStatusFailing},
		{rate: 49.9, want: interopStatusBroken},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%g", tt.rate), func(t *testing.T) {
			t.Parallel()
			got := interopStatusFromRate(tt.rate)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestInteropModeValidation(t *testing.T) {
	t.Parallel()

	require.True(t, isSupportedInteropMode(interopModeWalletsIssuers))
	require.True(t, isSupportedInteropMode(interopModeWalletsCredentials))
	require.True(t, isSupportedInteropMode(interopModeWalletsVerifiers))
	require.True(t, isSupportedInteropMode("wallets_verifiers"))
	require.True(t, isSupportedInteropMode(interopModeWalletsUseCaseVerifications))
	require.True(t, isSupportedInteropMode("wallets_use_case_verifications"))
	require.False(t, isSupportedInteropMode(interopMode("")))
	require.False(t, isSupportedInteropMode(interopMode("bad_mode")))
}

func TestInteropModeConfigRelations(t *testing.T) {
	t.Parallel()

	walletIssuer, ok := getInteropModeConfig(interopModeWalletsIssuers)
	require.True(t, ok)
	require.Equal(t, "wallets", walletIssuer.RowRelationField)
	require.Equal(t, "issuers", walletIssuer.ColumnRelationField)
	require.Equal(t, "wallet", walletIssuer.RowAxis)
	require.Equal(t, "issuer", walletIssuer.ColumnAxis)
	require.Equal(t, "wallets", walletIssuer.RowCollection)
	require.Equal(t, "credential_issuers", walletIssuer.ColumnCollection)

	walletCredential, ok := getInteropModeConfig(interopModeWalletsCredentials)
	require.True(t, ok)
	require.Equal(t, "wallets", walletCredential.RowRelationField)
	require.Equal(t, "credentials", walletCredential.ColumnRelationField)
	require.Equal(t, "wallet", walletCredential.RowAxis)
	require.Equal(t, "credential", walletCredential.ColumnAxis)
	require.Equal(t, "wallets", walletCredential.RowCollection)
	require.Equal(t, "credentials", walletCredential.ColumnCollection)

	cfg, ok := getInteropModeConfig(interopModeWalletsVerifiers)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "verifiers", cfg.ColumnRelationField)
	require.Equal(t, "wallet", cfg.RowAxis)
	require.Equal(t, "wallets", cfg.RowCollection)
	require.Equal(t, "verifier", cfg.ColumnAxis)
	require.Equal(t, "verifiers", cfg.ColumnCollection)

	cfg, ok = getInteropModeConfig(interopModeWalletsUseCaseVerifications)
	require.True(t, ok)
	require.Equal(t, "wallets", cfg.RowRelationField)
	require.Equal(t, "use_case_verifications", cfg.ColumnRelationField)
	require.Equal(t, "wallet", cfg.RowAxis)
	require.Equal(t, "wallets", cfg.RowCollection)
	require.Equal(t, "use_case_verification", cfg.ColumnAxis)
	require.Equal(t, "use_cases_verifications", cfg.ColumnCollection)

	_, ok = getInteropModeConfig(interopMode("bad_mode"))
	require.False(t, ok)
}

func TestInteropMatrixEntityJSONShape(t *testing.T) {
	t.Parallel()

	subtitle := "subtitle"
	avatarURL := "https://example.com/avatar.png"
	entity := InteropMatrixEntity{
		ID:        "rec1",
		Name:      "Entity",
		Subtitle:  &subtitle,
		AvatarURL: &avatarURL,
		Path:      "org/entities/entity",
	}

	raw, err := json.Marshal(entity)
	require.NoError(t, err)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, "rec1", payload["id"])
	require.Equal(t, "Entity", payload["name"])
	require.Equal(t, "org/entities/entity", payload["path"])
	require.Equal(t, "subtitle", payload["subtitle"])
	require.Equal(t, "https://example.com/avatar.png", payload["avatar_url"])

	entity.Subtitle = nil
	entity.AvatarURL = nil
	raw, err = json.Marshal(entity)
	require.NoError(t, err)

	var nilPayload map[string]any
	require.NoError(t, json.Unmarshal(raw, &nilPayload))
	_, hasSubtitle := nilPayload["subtitle"]
	require.False(t, hasSubtitle)
	_, hasAvatarURL := nilPayload["avatar_url"]
	require.False(t, hasAvatarURL)
}

func TestResolveCredentialEntityMetadata_AvatarFallbackOrder(t *testing.T) {
	t.Parallel()

	credentialAvatar := "https://cdn/credential.png"
	issuerAvatar := "https://cdn/issuer.png"
	issuerName := "Issuer A"

	entity := buildEnrichedEntityMetadata(
		"cred1",
		"Credential A",
		"org/credentials/credential-a",
		&credentialAvatar,
		&issuerName,
		&issuerAvatar,
	)
	require.Equal(t, "Credential A", entity.Name)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, issuerName, *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, credentialAvatar, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"cred2",
		"Credential B",
		"org/credentials/credential-b",
		nil,
		&issuerName,
		&issuerAvatar,
	)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, issuerAvatar, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"cred3",
		"Credential C",
		"org/credentials/credential-c",
		nil,
		nil,
		nil,
	)
	require.Nil(t, entity.AvatarURL)
	require.Nil(t, entity.Subtitle)
}

func TestLoadInteropMatrixFromCache_UnsupportedModeError(t *testing.T) {
	t.Parallel()

	app, err := tests.NewTestApp(testDataDir)
	require.NoError(t, err)
	defer app.Cleanup()

	_, err = loadInteropMatrixFromCache(app, interopMode("bad_mode"))
	require.Error(t, err)

	unsupported := unsupportedInteropModeError{}
	require.ErrorAs(t, err, &unsupported)
	require.Equal(t, interopMode("bad_mode"), unsupported.mode)
}

func TestHandleInteropMatrix_WalletsCredentialsHappyPath(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipeline := createPipelineRecord(t, app, orgID, "interop-wallets-credentials")

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-wallet")
	require.NoError(t, app.Save(wallet))

	issuersCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuer := core.NewRecord(issuersCollection)
	issuer.Set("url", "https://interop-issuer.example.com")
	issuer.Set("name", "interop-issuer")
	issuer.Set("owner", orgID)
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	credentialsCollection, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	credential := core.NewRecord(credentialsCollection)
	credential.Set("credential_issuer", issuer.Id)
	credential.Set("name", "interop-credential")
	credential.Set("display_name", "Interop Credential")
	credential.Set("json", `{}`)
	credential.Set("owner", orgID)
	require.NoError(t, app.Save(credential))

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	cacheRecord := core.NewRecord(cacheCollection)
	cacheRecord.Set("pipeline", pipeline.Id)
	cacheRecord.Set("total_runs", 10)
	cacheRecord.Set("total_successes", 8)
	cacheRecord.Set("success_rate", 80.0)
	cacheRecord.Set("manually_executed_runs", 6)
	cacheRecord.Set("scheduled_runs", 4)
	cacheRecord.Set("CI_runs", 0)
	cacheRecord.Set("minimum_running_time", "1m10s")
	cacheRecord.Set("first_execution", "2026-05-01T10:00:00Z")
	cacheRecord.Set("last_execution_date", "2026-05-01T11:00:00Z")
	cacheRecord.Set("wallets", []string{wallet.Id})
	cacheRecord.Set("credentials", []string{credential.Id})
	require.NoError(t, app.Save(cacheRecord))

	req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode=wallets_credentials", nil)
	rec := httptest.NewRecorder()

	err = HandleInteropMatrix()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp InteropMatrixResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	require.Equal(t, interopModeWalletsCredentials, resp.Mode)
	require.Equal(t, "wallet", resp.RowAxis)
	require.Equal(t, "credential", resp.ColumnAxis)
	require.NotEmpty(t, resp.Cells)

	cell, ok := findInteropCell(resp, wallet.Id, credential.Id)
	require.True(t, ok, "expected wallet x credential relation cell")
	require.Equal(t, 10, cell.TotalRuns)
	require.Equal(t, 8, cell.TotalSuccesses)
}

func TestHandleInteropMatrix_WalletsIssuersHappyPath(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipeline := createPipelineRecord(t, app, orgID, "interop-wallets-issuers")

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-wallet")
	require.NoError(t, app.Save(wallet))

	issuersCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuer := core.NewRecord(issuersCollection)
	issuer.Set("url", "https://interop-issuer.example.com")
	issuer.Set("name", "interop-issuer")
	issuer.Set("owner", orgID)
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	cacheRecord := core.NewRecord(cacheCollection)
	cacheRecord.Set("pipeline", pipeline.Id)
	cacheRecord.Set("total_runs", 12)
	cacheRecord.Set("total_successes", 9)
	cacheRecord.Set("success_rate", 75.0)
	cacheRecord.Set("manually_executed_runs", 7)
	cacheRecord.Set("scheduled_runs", 5)
	cacheRecord.Set("CI_runs", 0)
	cacheRecord.Set("minimum_running_time", "1m15s")
	cacheRecord.Set("first_execution", "2026-05-01T10:00:00Z")
	cacheRecord.Set("last_execution_date", "2026-05-01T11:00:00Z")
	cacheRecord.Set("wallets", []string{wallet.Id})
	cacheRecord.Set("issuers", []string{issuer.Id})
	require.NoError(t, app.Save(cacheRecord))

	req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode=wallets_issuers", nil)
	rec := httptest.NewRecorder()

	err = HandleInteropMatrix()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp InteropMatrixResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	require.Equal(t, interopModeWalletsIssuers, resp.Mode)
	require.Equal(t, "wallet", resp.RowAxis)
	require.Equal(t, "issuer", resp.ColumnAxis)
	require.NotEmpty(t, resp.Cells)

	cell, ok := findInteropCell(resp, wallet.Id, issuer.Id)
	require.True(t, ok, "expected wallet x issuer relation cell")
	require.Equal(t, 12, cell.TotalRuns)
	require.Equal(t, 9, cell.TotalSuccesses)
}

func TestHandleInteropMatrix_WalletsCredentialsColumnMetadataFallbackOrder(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipeline := createPipelineRecord(t, app, orgID, "interop-wallets-credentials-column-metadata")

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-wallet")
	require.NoError(t, app.Save(wallet))

	issuersCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuerWithAvatar := core.NewRecord(issuersCollection)
	issuerWithAvatar.Set("url", "https://interop-issuer-a.example.com")
	issuerWithAvatar.Set("name", "Issuer With Avatar")
	issuerWithAvatar.Set("logo_url", "https://cdn.example.com/issuer.png")
	issuerWithAvatar.Set("owner", orgID)
	issuerWithAvatar.Set("imported", true)
	require.NoError(t, app.Save(issuerWithAvatar))

	issuerWithoutAvatar := core.NewRecord(issuersCollection)
	issuerWithoutAvatar.Set("url", "https://interop-issuer-b.example.com")
	issuerWithoutAvatar.Set("name", "Issuer Without Avatar")
	issuerWithoutAvatar.Set("owner", orgID)
	issuerWithoutAvatar.Set("imported", true)
	require.NoError(t, app.Save(issuerWithoutAvatar))

	credentialsCollection, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	credentialWithOwnAvatar := core.NewRecord(credentialsCollection)
	credentialWithOwnAvatar.Set("credential_issuer", issuerWithAvatar.Id)
	credentialWithOwnAvatar.Set("name", "credential-with-own-avatar")
	credentialWithOwnAvatar.Set("display_name", "Credential With Own Avatar")
	credentialWithOwnAvatar.Set("logo_url", "https://cdn.example.com/credential.png")
	credentialWithOwnAvatar.Set("json", `{}`)
	credentialWithOwnAvatar.Set("owner", orgID)
	require.NoError(t, app.Save(credentialWithOwnAvatar))

	credentialWithIssuerFallback := core.NewRecord(credentialsCollection)
	credentialWithIssuerFallback.Set("credential_issuer", issuerWithAvatar.Id)
	credentialWithIssuerFallback.Set("name", "credential-with-issuer-fallback")
	credentialWithIssuerFallback.Set("display_name", "Credential With Issuer Fallback")
	credentialWithIssuerFallback.Set("json", `{}`)
	credentialWithIssuerFallback.Set("owner", orgID)
	require.NoError(t, app.Save(credentialWithIssuerFallback))

	credentialWithNoAvatars := core.NewRecord(credentialsCollection)
	credentialWithNoAvatars.Set("credential_issuer", issuerWithoutAvatar.Id)
	credentialWithNoAvatars.Set("name", "credential-with-no-avatars")
	credentialWithNoAvatars.Set("display_name", "Credential With No Avatars")
	credentialWithNoAvatars.Set("json", `{}`)
	credentialWithNoAvatars.Set("owner", orgID)
	require.NoError(t, app.Save(credentialWithNoAvatars))

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	cacheRecord := core.NewRecord(cacheCollection)
	cacheRecord.Set("pipeline", pipeline.Id)
	cacheRecord.Set("total_runs", 12)
	cacheRecord.Set("total_successes", 9)
	cacheRecord.Set("success_rate", 75.0)
	cacheRecord.Set("manually_executed_runs", 7)
	cacheRecord.Set("scheduled_runs", 5)
	cacheRecord.Set("CI_runs", 0)
	cacheRecord.Set("minimum_running_time", "1m20s")
	cacheRecord.Set("first_execution", "2026-05-02T10:00:00Z")
	cacheRecord.Set("last_execution_date", "2026-05-02T11:00:00Z")
	cacheRecord.Set("wallets", []string{wallet.Id})
	cacheRecord.Set("credentials", []string{
		credentialWithOwnAvatar.Id,
		credentialWithIssuerFallback.Id,
		credentialWithNoAvatars.Id,
	})
	require.NoError(t, app.Save(cacheRecord))

	req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode=wallets_credentials", nil)
	rec := httptest.NewRecorder()

	err = HandleInteropMatrix()(&core.RequestEvent{
		App: app,
		Event: router.Event{
			Request:  req,
			Response: rec,
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp InteropMatrixResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	columnsByID := make(map[string]InteropMatrixEntity, len(resp.Columns))
	for _, column := range resp.Columns {
		columnsByID[column.ID] = column
	}

	colWithOwnAvatar, ok := columnsByID[credentialWithOwnAvatar.Id]
	require.True(t, ok, "expected credential-with-own-avatar column")
	require.NotNil(t, colWithOwnAvatar.Subtitle)
	require.Equal(t, "Issuer With Avatar", *colWithOwnAvatar.Subtitle)
	require.NotNil(t, colWithOwnAvatar.AvatarURL)
	require.Equal(t, "https://cdn.example.com/credential.png", *colWithOwnAvatar.AvatarURL)

	colWithIssuerFallback, ok := columnsByID[credentialWithIssuerFallback.Id]
	require.True(t, ok, "expected credential-with-issuer-fallback column")
	require.NotNil(t, colWithIssuerFallback.Subtitle)
	require.Equal(t, "Issuer With Avatar", *colWithIssuerFallback.Subtitle)
	require.NotNil(t, colWithIssuerFallback.AvatarURL)
	require.Equal(t, "https://cdn.example.com/issuer.png", *colWithIssuerFallback.AvatarURL)

	colWithNoAvatars, ok := columnsByID[credentialWithNoAvatars.Id]
	require.True(t, ok, "expected credential-with-no-avatars column")
	require.NotNil(t, colWithNoAvatars.Subtitle)
	require.Equal(t, "Issuer Without Avatar", *colWithNoAvatars.Subtitle)
	if colWithNoAvatars.AvatarURL != nil {
		require.Empty(t, *colWithNoAvatars.AvatarURL)
	}
}

func TestHandleInteropMatrix_ModeValidationReturnsBadRequest(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
	}{
		{
			name: "invalid mode",
			url:  "/api/scoreboard/interop?mode=bad_mode",
		},
		{
			name: "missing mode",
			url:  "/api/scoreboard/interop",
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app, err := tests.NewTestApp(testDataDir)
			require.NoError(t, err)
			defer app.Cleanup()

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()

			err = HandleInteropMatrix()(&core.RequestEvent{
				App: app,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusBadRequest, rec.Code)

			var apiErr apierror.APIError
			require.NoError(t, json.NewDecoder(rec.Body).Decode(&apiErr))
			require.Equal(t, http.StatusBadRequest, apiErr.Code)
			require.Equal(t, "mode", apiErr.Domain)
			require.Equal(t, "unsupported or missing mode", apiErr.Reason)
			require.Equal(t, "use mode=wallets_credentials, wallets_issuers, wallets_verifiers, or wallets_use_case_verifications", apiErr.Message)
		})
	}
}

func TestBuildEnrichedEntityMetadata_UseCaseVerification(t *testing.T) {
	t.Parallel()

	useCaseLogo := "https://cdn/usecase-logo.png"
	verifierLogo := "https://cdn/verifier-logo.png"
	verifierName := "Verifier A"

	entity := buildEnrichedEntityMetadata(
		"uc1", "PID Verification", "org/v/p",
		ptrTo(useCaseLogo), ptrTo(verifierName), ptrTo(verifierLogo),
	)
	require.Equal(t, "PID Verification", entity.Name)
	require.NotNil(t, entity.Subtitle)
	require.Equal(t, verifierName, *entity.Subtitle)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, useCaseLogo, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"uc2", "PID Verification", "org/v/p",
		nil, ptrTo(verifierName), ptrTo(verifierLogo),
	)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, verifierLogo, *entity.AvatarURL)

	entity = buildEnrichedEntityMetadata(
		"uc3", "PID Verification", "org/v/p",
		nil, nil, nil,
	)
	require.Nil(t, entity.AvatarURL)
	require.Nil(t, entity.Subtitle)
}

func TestInteropEntityFromRecord_CredentialIssuerLogoURLFallback(t *testing.T) {
	t.Parallel()
	app := setupScoreboardInteropApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	credentialIssuersColl, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	rec := core.NewRecord(credentialIssuersColl)
	rec.Set("owner", orgID)
	rec.Set("url", "https://example.com")
	rec.Set("name", "Test Issuer")
	rec.Set("canonified_name", "test-issuer")
	rec.Set("imported", true)
	rec.Set("logo_url", "https://cdn.example.com/logo.png")
	require.NoError(t, app.Save(rec))

	entity, err := interopEntityFromRecord(app, rec, "credential_issuers")
	require.NoError(t, err)
	require.Equal(t, "Test Issuer", entity.Name)
	require.NotNil(t, entity.AvatarURL)
	require.Equal(t, "https://cdn.example.com/logo.png", *entity.AvatarURL)
	require.Nil(t, entity.Subtitle)
}

func TestHandleInteropMatrix_AllSupportedModes(t *testing.T) {
	app := setupPipelineApp(t)
	defer app.Cleanup()

	orgID, err := getOrgIDfromName("userA's organization")
	require.NoError(t, err)

	pipeline := createPipelineRecord(t, app, orgID, "interop-all-modes")

	walletsCollection, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)

	wallet := core.NewRecord(walletsCollection)
	wallet.Set("owner", orgID)
	wallet.Set("name", "interop-mode-wallet")
	require.NoError(t, app.Save(wallet))

	issuersCollection, err := app.FindCollectionByNameOrId("credential_issuers")
	require.NoError(t, err)

	issuer := core.NewRecord(issuersCollection)
	issuer.Set("url", "https://interop-all-modes.example.com")
	issuer.Set("name", "interop-mode-issuer")
	issuer.Set("owner", orgID)
	issuer.Set("imported", true)
	require.NoError(t, app.Save(issuer))

	credentialsCollection, err := app.FindCollectionByNameOrId("credentials")
	require.NoError(t, err)

	credential := core.NewRecord(credentialsCollection)
	credential.Set("credential_issuer", issuer.Id)
	credential.Set("name", "interop-mode-credential")
	credential.Set("display_name", "Interop Mode Credential")
	credential.Set("json", `{}`)
	credential.Set("owner", orgID)
	require.NoError(t, app.Save(credential))

	verifiersCollection, err := app.FindCollectionByNameOrId("verifiers")
	require.NoError(t, err)

	verifier := core.NewRecord(verifiersCollection)
	verifier.Set("url", "https://interop-mode-verifier.example.com")
	verifier.Set("name", "interop-mode-verifier")
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
	useCase.Set("name", "interop-mode-usecase")
	useCase.Set("deeplink", "https://example.com/usecase")
	useCase.Set("yaml", "type: verification")
	useCase.Set("verifier", verifier.Id)
	useCase.Set("owner", orgID)
	require.NoError(t, app.Save(useCase))

	cacheCollection, err := app.FindCollectionByNameOrId("pipeline_scoreboard_cache")
	require.NoError(t, err)

	cacheRecord := core.NewRecord(cacheCollection)
	cacheRecord.Set("pipeline", pipeline.Id)
	cacheRecord.Set("total_runs", 10)
	cacheRecord.Set("total_successes", 8)
	cacheRecord.Set("success_rate", 80.0)
	cacheRecord.Set("manually_executed_runs", 6)
	cacheRecord.Set("scheduled_runs", 4)
	cacheRecord.Set("CI_runs", 0)
	cacheRecord.Set("minimum_running_time", "1m10s")
	cacheRecord.Set("first_execution", "2026-05-01T10:00:00Z")
	cacheRecord.Set("last_execution_date", "2026-05-01T11:00:00Z")
	cacheRecord.Set("wallets", []string{wallet.Id})
	cacheRecord.Set("issuers", []string{issuer.Id})
	cacheRecord.Set("credentials", []string{credential.Id})
	cacheRecord.Set("verifiers", []string{verifier.Id})
	cacheRecord.Set("use_case_verifications", []string{useCase.Id})
	require.NoError(t, app.Save(cacheRecord))

	for _, mode := range []string{
		"wallets_credentials",
		"wallets_issuers",
		"wallets_verifiers",
		"wallets_use_case_verifications",
	} {
		t.Run(mode, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/scoreboard/interop?mode="+mode, nil)
			rec := httptest.NewRecorder()

			err := HandleInteropMatrix()(&core.RequestEvent{
				App: app,
				Event: router.Event{
					Request:  req,
					Response: rec,
				},
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
		})
	}
}


