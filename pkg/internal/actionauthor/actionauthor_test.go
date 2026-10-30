// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
	"strings"
	"testing"

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
		Platform:                PlatformAndroid,
	}, org.GetString("canonified_name")
}

// seedVersion adds a build of the fixture wallet with the given installers and returns its identifier.
func seedVersion(
	t *testing.T,
	app *tests.TestApp,
	selection Selection,
	tag string,
	android, ios []byte,
) string {
	t.Helper()
	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	version := core.NewRecord(versions)
	version.Set("owner", selection.Organization.Id)
	version.Set("wallet", selection.Wallet.Id)
	version.Set("tag", tag)
	if android != nil {
		apk, err := filesystem.NewFileFromBytes(android, "selected.apk")
		require.NoError(t, err)
		version.Set("android_installer", []*filesystem.File{apk})
	}
	if ios != nil {
		zipped, err := filesystem.NewFileFromBytes(ios, "Selected.zip")
		require.NoError(t, err)
		version.Set("ios_installer", []*filesystem.File{zipped})
	}
	require.NoError(t, app.Save(version))
	return strings.Join([]string{
		selection.Organization.GetString("canonified_name"),
		selection.Wallet.GetString("canonified_name"),
		version.GetString("canonified_tag"),
	}, "/")
}

func TestListBuildOptionsRequiresOwnershipAndInstaller(t *testing.T) {
	app, request, _ := authoringFixture(t)
	selection, err := Select(app, "", request)
	require.NoError(t, err)

	versions, err := app.FindCollectionByNameOrId("wallet_versions")
	require.NoError(t, err)
	withoutInstaller := core.NewRecord(versions)
	withoutInstaller.Set("owner", selection.Organization.Id)
	withoutInstaller.Set("wallet", selection.Wallet.Id)
	withoutInstaller.Set("tag", "without-installer")
	require.NoError(t, app.Save(withoutInstaller))
	iosOnly := seedVersion(t, app, selection, "ios-only", nil, []byte("ios app zip"))
	both := seedVersion(t, app, selection, "both", []byte("apk"), []byte("ios app zip"))

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
	platforms := map[string][]string{}
	for _, option := range options {
		platforms[option.Identifier] = option.Platforms
	}
	assert.Equal(t, map[string][]string{
		request.WalletVersionIdentifier: {PlatformAndroid},
		iosOnly:                         {PlatformIOS},
		both:                            {PlatformAndroid, PlatformIOS},
	}, platforms)
	for _, option := range options {
		if option.Identifier == request.WalletVersionIdentifier {
			assert.Equal(t, "Authoring test wallet", option.WalletName)
			assert.Equal(t, "userA's organization", option.OrganizationName)
			assert.Equal(t, "build-1", option.VersionTag)
		}
	}

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

func TestSelectRequiresPlatformInstaller(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	iosOnly := seedVersion(t, app, selection, "ios-only", nil, []byte("ios app zip"))
	for _, tc := range []struct {
		name       string
		identifier string
		platform   string
		message    string
	}{
		{"missing platform", request.WalletVersionIdentifier, "", "platform must be android or ios"},
		{"invalid platform", request.WalletVersionIdentifier, "windows", "platform must be android or ios"},
		{"android build without iOS app", request.WalletVersionIdentifier, PlatformIOS, "selected version has no iOS app"},
		{"iOS build without Android APK", iosOnly, PlatformAndroid, "selected version has no Android APK"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := request
			in.WalletVersionIdentifier = tc.identifier
			in.Platform = tc.platform
			_, err := Select(app, namespace, in)
			require.ErrorIs(t, err, ErrInvalid)
			assert.ErrorContains(t, err, tc.message)
		})
	}

	// Authorization is still checked before the platform installer.
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	unauthorized := request
	unauthorized.ActorID = other.Id
	unauthorized.Platform = PlatformIOS
	_, err = Select(app, namespace, unauthorized)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestAuthoringBuildResolvesPlatformInstaller(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	iosRequest := request
	iosRequest.WalletVersionIdentifier = seedVersion(
		t, app, selection, "ios-build", []byte("other APK bytes"), []byte("selected iOS app zip"),
	)
	iosRequest.Platform = PlatformIOS
	for _, tc := range []struct {
		name     string
		request  BuildRequest
		content  string
		platform string
	}{
		{"android", request, "selected APK bytes", PlatformAndroid},
		{"ios", iosRequest, "selected iOS app zip", PlatformIOS},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := Resolve(app, namespace, tc.request)
			require.NoError(t, err)
			assert.Equal(t, tc.platform, build.Platform)
			assert.Equal(t, CodeSHA256(tc.content), build.InstallerSHA256)
		})
	}
}

func TestAuthoringBuildPermission(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	_, err := Resolve(app, namespace, request)
	require.NoError(t, err)

	_, err = Resolve(app, "another-organization", request)
	require.ErrorIs(t, err, ErrForbidden)
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	otherRequest := request
	otherRequest.ActorID = other.Id
	_, err = Resolve(app, namespace, otherRequest)
	require.ErrorIs(t, err, ErrForbidden)

	authorization, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user} && organization.canonified_name = {:org}",
		map[string]any{"user": request.ActorID, "org": namespace},
	)
	require.NoError(t, err)
	require.NoError(t, app.Delete(authorization))
	_, err = Resolve(app, namespace, request)
	require.ErrorIs(t, err, ErrForbidden)
}

func acceptedRequest(request BuildRequest, digest string) SaveRequest {
	code := "appId: eu.europa.ec.euidi\n---\n- launchApp\n- inputText: \"001234\"\n- assertVisible: Home\n- takeScreenshot: done\n"
	return SaveRequest{
		BuildRequest:    request,
		Name:            "Reviewed onboarding",
		Code:            code,
		CandidateSHA256: CodeSHA256(code),
		InstallerSHA256: digest,
		PackageID:       "eu.europa.ec.euidi",
		Slot:            "onboarding",
		ReplayCount:     2,
	}
}

func TestAuthoringSaveIsUnpublishedIdempotentAndCompareAndSwap(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.InstallerSHA256)
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
	in.InstallerSHA256 = strings.Repeat("0", 64)
	_, err = Save(app, namespace, "new-workflow", "new-run", in)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "selected installer changed")
}

func TestAuthoringSaveRecordsIOSProvenance(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	request.WalletVersionIdentifier = seedVersion(
		t, app, selection, "ios-build", []byte("other APK bytes"), []byte("selected iOS app zip"),
	)
	request.Platform = PlatformIOS
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)

	// The Android digest of the same version does not authorize the iOS save.
	in := acceptedRequest(request, CodeSHA256("other APK bytes"))
	_, err = Save(app, namespace, "workflow", "run", in)
	require.ErrorIs(t, err, ErrConflict)

	in = acceptedRequest(request, build.InstallerSHA256)
	receipt, err := Save(app, namespace, "workflow", "run", in)
	require.NoError(t, err)
	assert.Equal(t, PlatformIOS, receipt.Provenance.Platform)
	assert.Equal(t, CodeSHA256("selected iOS app zip"), receipt.Provenance.InstallerSHA256)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(record.GetString("authoring_provenance")), &stored))
	assert.Equal(t, "ios", stored["platform"])
	assert.Equal(t, build.InstallerSHA256, stored["installer_sha256"])
	assert.NotContains(t, stored, "apk_sha256")

	// A retry of the same run is idempotent only for the same platform.
	retried, err := Save(app, namespace, "workflow", "run", in)
	require.NoError(t, err)
	assert.Equal(t, receipt.RecordID, retried.RecordID)
	android := acceptedRequest(request, CodeSHA256("other APK bytes"))
	android.Platform = PlatformAndroid
	_, err = Save(app, namespace, "workflow", "run", android)
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
		in := acceptedRequest(request, build.InstallerSHA256)
		mutate(&in)
		_, err := Save(app, namespace, "workflow", "run", in)
		require.ErrorIs(t, err, ErrInvalid)
	}
	_, err = Save(
		app,
		"another-organization",
		"workflow",
		"run",
		acceptedRequest(request, build.InstallerSHA256),
	)
	require.ErrorIs(t, err, ErrForbidden)
}
