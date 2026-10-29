// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func authoringFixture(t *testing.T) (*tests.TestApp, BuildRequest, string) {
	t.Helper()
	app, err := tests.NewTestApp("../../../test_pb_data")
	require.NoError(t, err)
	t.Cleanup(app.Cleanup)
	canonify.RegisterCanonifyHooks(app)
	actor, err := app.FindAuthRecordByEmail("users", "userA@example.org")
	require.NoError(t, err)
	org, err := app.FindFirstRecordByData("organizations", "name", "userA's organization")
	require.NoError(t, err)
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", org.Id)
	wallet.Set("name", "Authoring test wallet")
	require.NoError(t, app.Save(wallet))
	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	version := core.NewRecord(versions)
	version.Set("owner", org.Id)
	version.Set("wallet", wallet.Id)
	version.Set("tag", "build-1")
	apk, err := filesystem.NewFileFromBytes([]byte("selected APK bytes"), "selected.apk")
	require.NoError(t, err)
	version.Set("android_installer", []*filesystem.File{apk})
	require.NoError(t, app.Save(version))
	actions, err := app.FindCollectionByNameOrId("wallet_actions")
	require.NoError(t, err)
	if actions.Fields.GetByName("authoring_provenance") == nil {
		actions.Fields.Add(
			&core.JSONField{Name: "authoring_provenance", Hidden: true, MaxSize: 65536},
		)
		require.NoError(t, app.Save(actions))
	}
	identifier := strings.Join(
		[]string{
			org.GetString("canonified_name"),
			wallet.GetString("canonified_name"),
			version.GetString("canonified_tag"),
		},
		"/",
	)
	return app, BuildRequest{
		Actor:                   Actor{ActorID: actor.Id, ActorCollection: "users"},
		WalletVersionIdentifier: identifier,
	}, org.GetString("canonified_name")
}

func TestListBuildOptionsRequiresOwnershipAndAndroidAPK(t *testing.T) {
	app, request, _ := authoringFixture(t)
	selection, err := Select(app, "", request)
	require.NoError(t, err)

	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	withoutAPK := core.NewRecord(versions)
	withoutAPK.Set("owner", selection.Organization.Id)
	withoutAPK.Set("wallet", selection.Wallet.Id)
	withoutAPK.Set("tag", "without-apk")
	require.NoError(t, app.Save(withoutAPK))

	// Another member of the organization neither hides the owner's builds nor sees them.
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	member, err := app.FindFirstRecordByData("orgRoles", "name", "member")
	require.NoError(t, err)
	authorizations, err := app.FindCollectionByNameOrId("orgAuthorizations")
	require.NoError(t, err)
	membership := core.NewRecord(authorizations)
	membership.Set("user", other.Id)
	membership.Set("organization", selection.Organization.Id)
	membership.Set("role", member.Id)
	require.NoError(t, app.Save(membership))

	options, err := ListBuildOptions(app, request.Actor)
	require.NoError(t, err)
	require.Len(t, options, 1)
	assert.Equal(t, request.WalletVersionIdentifier, options[0].Identifier)
	assert.Equal(t, "Authoring test wallet", options[0].WalletName)
	assert.Equal(t, "userA's organization", options[0].OrganizationName)
	assert.Equal(t, "build-1", options[0].VersionTag)

	options, err = ListBuildOptions(app, Actor{ActorID: other.Id, ActorCollection: "users"})
	require.NoError(t, err)
	assert.Empty(t, options)

	authorization, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user} && organization = {:organization}",
		dbx.Params{"user": request.ActorID, "organization": selection.Organization.Id},
	)
	require.NoError(t, err)
	authorization.Set("role", member.Id)
	require.NoError(t, app.Save(authorization))
	options, err = ListBuildOptions(app, request.Actor)
	require.NoError(t, err)
	assert.Empty(t, options)
}

func TestAuthoringBuildPermissionAndCapability(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	file, err := OpenInstaller(app, build.DownloadGrant)
	require.NoError(t, err)
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	require.NoError(t, file.Close())
	assert.Equal(t, []byte("selected APK bytes"), content)
	assert.Equal(t, CodeSHA256(string(content)), build.APKSHA256)

	_, err = Resolve(app, "another-organization", request)
	require.ErrorIs(t, err, ErrForbidden)
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	otherRequest := request
	otherRequest.ActorID = other.Id
	_, err = Resolve(app, namespace, otherRequest)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = OpenInstaller(app, "invalid."+build.DownloadGrant)
	require.ErrorIs(t, err, ErrForbidden)

	// A correctly signed but expired capability must fail, not just malformed ones.
	encoded, _, _ := strings.Cut(build.DownloadGrant, ".")
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var claims installerGrant
	require.NoError(t, json.Unmarshal(payload, &claims))
	claims.Expires = time.Now().Add(-time.Minute).Unix()
	payload, err = json.Marshal(claims)
	require.NoError(t, err)
	encoded = base64.RawURLEncoding.EncodeToString(payload)
	key, err := grantKey(app)
	require.NoError(t, err)
	mac := hmac.New(sha256.New, key)
	_, err = io.WriteString(mac, encoded)
	require.NoError(t, err)
	_, err = OpenInstaller(app, encoded+"."+base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))
	require.ErrorIs(t, err, ErrForbidden)

	authorization, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user} && organization.canonified_name = {:org}",
		map[string]any{"user": request.ActorID, "org": namespace},
	)
	require.NoError(t, err)
	require.NoError(t, app.Delete(authorization))
	_, err = OpenInstaller(app, build.DownloadGrant)
	require.ErrorIs(t, err, ErrForbidden)
}

func acceptedRequest(request BuildRequest, digest string) SaveRequest {
	code := "appId: eu.europa.ec.euidi\n---\n- launchApp\n- inputText: \"001234\"\n- assertVisible: Home\n- takeScreenshot: done\n"
	return SaveRequest{
		BuildRequest:    request,
		Name:            "Reviewed onboarding",
		Code:            code,
		CandidateSHA256: CodeSHA256(code),
		APKSHA256:       digest,
		PackageID:       "eu.europa.ec.euidi",
		Slot:            "onboarding",
		ReplayCount:     2,
	}
}

func TestAuthoringSaveIsUnpublishedIdempotentAndCompareAndSwap(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.APKSHA256)
	receipt, err := Save(app, namespace, "workflow", "run", in)
	require.NoError(t, err)
	assert.True(t, receipt.Created)
	assert.False(t, receipt.Published)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, in.Code, record.GetString("code"))
	assert.Equal(t, build.VersionID, receipt.Provenance.WalletVersionID)
	retried, err := Save(app, namespace, "workflow", "run", in)
	require.NoError(t, err)
	assert.Equal(t, receipt.RecordID, retried.RecordID)
	_, err = Save(app, namespace, "another-workflow", "another-run", in)
	require.ErrorIs(t, err, ErrConflict)

	in.ReplaceActionID = receipt.RecordID
	in.ExpectedExistingSHA256 = in.CandidateSHA256
	in.Code += "# reviewed replacement\n"
	in.CandidateSHA256 = CodeSHA256(in.Code)
	replacement, err := Save(app, namespace, "replacement-workflow", "replacement-run", in)
	require.NoError(t, err)
	assert.Equal(t, receipt.RecordID, replacement.RecordID)
	assert.False(t, replacement.Published)
	_, err = Save(app, namespace, "stale-workflow", "stale-run", in)
	require.ErrorIs(t, err, ErrConflict)
	in.ExpectedExistingSHA256 = in.CandidateSHA256
	in.APKSHA256 = strings.Repeat("0", 64)
	_, err = Save(app, namespace, "new-workflow", "new-run", in)
	require.ErrorIs(t, err, ErrConflict)
}

func TestAuthoringSaveRejectsUnreplayedOrChangedCandidate(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	for _, mutate := range []func(*SaveRequest){
		func(in *SaveRequest) { in.ReplayCount = 1 },
		func(in *SaveRequest) { in.Code += "# not approved\n" },
		func(in *SaveRequest) { in.PackageID = "org.wrong.wallet" },
		func(in *SaveRequest) { in.ReplaceActionID = "unconfirmed" },
	} {
		in := acceptedRequest(request, build.APKSHA256)
		mutate(&in)
		_, err := Save(app, namespace, "workflow", "run", in)
		require.ErrorIs(t, err, ErrInvalid)
	}
	_, err = Save(
		app,
		"another-organization",
		"workflow",
		"run",
		acceptedRequest(request, build.APKSHA256),
	)
	require.ErrorIs(t, err, ErrForbidden)
}
