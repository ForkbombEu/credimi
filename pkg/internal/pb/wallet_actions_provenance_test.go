// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalletActionProvenanceCannotBeForgedAndIsClearedAfterCodeEdit(t *testing.T) {
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	defer app.Cleanup()
	canonify.RegisterCanonifyHooks(app)
	RegisterWalletActionHooks(app)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "Provenance test wallet")
	require.NoError(t, app.Save(wallet))
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	if actions.Fields.GetByName("authoring_provenance") == nil {
		actions.Fields.Add(
			&core.JSONField{Name: "authoring_provenance", Hidden: true, MaxSize: 65536},
		)
		require.NoError(t, app.Save(actions))
	}
	code := "appId: eu.europa.ec.euidi\n---\n- assertVisible: Home\n"
	digest := sha256.Sum256([]byte(code))
	provenance := map[string]any{
		"workflow_id": "reviewed-workflow",
		"yaml_sha256": hex.EncodeToString(digest[:]),
	}
	record := core.NewRecord(actions)
	record.Set("owner", org.Id)
	record.Set("wallet", wallet.Id)
	record.Set("name", "Reviewed action")
	record.Set("code", code)
	record.Set("category", "onboarding")
	record.Set("authoring_provenance", provenance)
	require.NoError(t, app.Save(record))
	user, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	token, err := user.NewAuthToken()
	require.NoError(t, err)
	router, err := apis.NewRouter(app)
	require.NoError(t, err)
	mux, err := router.BuildMux()
	require.NoError(t, err)
	patch := func(body map[string]any) *httptest.ResponseRecorder {
		encoded, marshalErr := json.Marshal(body)
		require.NoError(t, marshalErr)
		request := httptest.NewRequest(
			http.MethodPatch,
			"/api/collections/wallet_actions/records/"+record.Id,
			bytes.NewReader(encoded),
		)
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		return response
	}
	response := patch(
		map[string]any{"authoring_provenance": map[string]any{"workflow_id": "forged"}},
	)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	stored, err := app.FindRecordById("wallet_actions", record.Id)
	require.NoError(t, err)
	var storedProvenance map[string]any
	require.NoError(
		t,
		json.Unmarshal([]byte(stored.GetString("authoring_provenance")), &storedProvenance),
	)
	assert.Equal(t, provenance, storedProvenance)
	assert.NotContains(t, response.Body.String(), "authoring_provenance")
	response = patch(map[string]any{"code": code + "# owner edited this action\n"})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	stored, err = app.FindRecordById("wallet_actions", record.Id)
	require.NoError(t, err)
	assert.Equal(t, code+"# owner edited this action\n", stored.GetString("code"))
	assert.Contains(t, []string{"", "null"}, stored.GetString("authoring_provenance"))
}
