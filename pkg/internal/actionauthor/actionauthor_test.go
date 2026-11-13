// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
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
		Actor:                   Actor{ActorID: actor.Id},
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

// newMember creates a user whose only organization is orgID, with the member role.
func newMember(t *testing.T, app *tests.TestApp, orgID string) *core.Record {
	t.Helper()
	users, err := app.FindCollectionByNameOrId("users")
	require.NoError(t, err)
	user := core.NewRecord(users)
	user.SetEmail("authoring-member@example.org")
	user.SetPassword("authoring-member-password")
	require.NoError(t, app.Save(user))
	member, err := app.FindFirstRecordByData("orgRoles", "name", "member")
	require.NoError(t, err)
	authorizations, err := app.FindCollectionByNameOrId("orgAuthorizations")
	require.NoError(t, err)
	membership := core.NewRecord(authorizations)
	membership.Set("user", user.Id)
	membership.Set("organization", orgID)
	membership.Set("role", member.Id)
	require.NoError(t, app.Save(membership))
	return user
}

func TestListBuildOptionsRequiresMembershipAndInstaller(t *testing.T) {
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

	// A member whose organization it is sees the same builds.
	member := newMember(t, app, selection.Organization.Id)
	memberOptions, err := ListBuildOptions(app, Actor{ActorID: member.Id})
	require.NoError(t, err)
	assert.Equal(t, options, memberOptions)

	// userB's organization is its own: userA's builds are not listed.
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	options, err = ListBuildOptions(app, Actor{ActorID: other.Id})
	require.NoError(t, err)
	assert.Empty(t, options)

	// A user without an organization gets an empty list.
	authorization, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user}",
		dbx.Params{"user": member.Id},
	)
	require.NoError(t, err)
	require.NoError(t, app.Delete(authorization))
	options, err = ListBuildOptions(app, Actor{ActorID: member.Id})
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

	// Superusers are not users: they cannot author.
	superuser, err := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, "admin@example.org")
	require.NoError(t, err)
	superuserRequest := request
	superuserRequest.ActorID = superuser.Id
	_, err = Resolve(app, namespace, superuserRequest)
	require.ErrorIs(t, err, ErrForbidden)

	org, err := app.FindFirstRecordByData("organizations", "canonified_name", namespace)
	require.NoError(t, err)
	memberRequest := request
	memberRequest.ActorID = newMember(t, app, org.Id).Id
	_, err = Resolve(app, namespace, memberRequest)
	require.NoError(t, err)

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
		ReplayCount:     1,
		Spec:            Spec{Slot: "onboarding", Outcome: OutcomeSuccess},
	}
}

// testApproval is the verified approval Save records.
var testApproval = Approval{
	AuthorWorkflowID: "workflow",
	ApproverID:       "approver",
	ApprovedAt:       1_800_000_000,
}

func TestAuthoringSaveWithoutReplacesIsUnpublishedIdempotentAndCreates(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.InstallerSHA256)
	receipt, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.True(t, receipt.Created)
	assert.False(t, receipt.Published)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, in.Code, record.GetString("code"))
	assert.Equal(t, "onboarding", record.GetString("category"))
	assert.Equal(t, build.VersionID, receipt.Provenance.WalletVersionID)
	retried, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.Equal(t, receipt.RecordID, retried.RecordID)
	// Another run, or a changed candidate, never replaces the existing action
	// and does not reveal its id or code.
	_, err = Save(app, namespace, "another-workflow", "another-run", in, testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.NotContains(t, err.Error(), receipt.RecordID)
	assert.NotContains(t, err.Error(), in.CandidateSHA256)
	changed := in
	changed.Code += "# reviewed replacement\n"
	changed.CandidateSHA256 = CodeSHA256(changed.Code)
	_, err = Save(app, namespace, "workflow", "run", changed, testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "an action with this name already exists")
	record, err = app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, in.Code, record.GetString("code"))

	in.Name = "Another onboarding"
	in.InstallerSHA256 = strings.Repeat("0", 64)
	_, err = Save(app, namespace, "new-workflow", "new-run", in, testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "selected installer changed")
}

func TestAuthoringSaveReplacesInPlace(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	const oldCode = "appId: eu.europa.ec.euidi\n---\n- launchApp\n"
	target, path := seedWalletAction(t, app, selection.Organization.Id, selection.Wallet,
		"Existing onboarding", "onboarding", oldCode, true)
	canonified := target.GetString("canonified_name")
	seedWalletAction(t, app, selection.Organization.Id, selection.Wallet,
		"Other onboarding", "onboarding", oldCode, false)
	in := acceptedRequest(request, build.InstallerSHA256)
	in.Name = "Existing onboarding"
	replacing := func(codeSHA256 string) SaveRequest {
		request := in
		request.Spec.Replaces = &ReplaceSpec{Action: path, CodeSHA256: codeSHA256}
		return request
	}

	_, err = Save(app, namespace, "workflow", "run", replacing(""), testApproval)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "replaces: the plan has no code_sha256; describe the action again")

	_, err = Save(app, namespace, "workflow", "run", replacing(CodeSHA256("older code")),
		testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "replaces: "+path+
		" changed since the plan was checked; describe the action again")

	renamed := replacing(CodeSHA256(oldCode))
	renamed.Name = "Other onboarding"
	_, err = Save(app, namespace, "workflow", "run", renamed, testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "an action with this name already exists")

	receipt, err := Save(app, namespace, "workflow", "run", replacing(CodeSHA256(oldCode)),
		testApproval)
	require.NoError(t, err)
	assert.False(t, receipt.Created)
	assert.True(t, receipt.Replaced)
	assert.True(t, receipt.Published)
	assert.Equal(t, target.Id, receipt.RecordID)
	assert.Equal(t, CodeSHA256(oldCode), receipt.Provenance.ReplacedCodeSHA256)
	record, err := app.FindRecordById("wallet_actions", target.Id)
	require.NoError(t, err)
	assert.Equal(t, in.Code, record.GetString("code"))
	assert.Equal(t, "onboarding", record.GetString("category"))
	assert.Equal(t, canonified, record.GetString("canonified_name"))
	assert.True(t, record.GetBool("published"))
	var provenance Provenance
	require.NoError(
		t,
		json.Unmarshal([]byte(record.GetString("authoring_provenance")), &provenance),
	)
	assert.Equal(t, receipt.Provenance, provenance)

	// The activity retry finds its own write; the plan check still accepts
	// the old hash for it.
	retried, err := Save(app, namespace, "workflow", "run", replacing(CodeSHA256(oldCode)),
		testApproval)
	require.NoError(t, err)
	assert.Equal(t, receipt, retried)
	spec := replacing(CodeSHA256(oldCode)).Spec
	_, err = CheckReplaceTarget(app, selection, &spec)
	require.NoError(t, err)

	// Another run with the old hash finds the code changed.
	_, err = Save(app, namespace, "another-workflow", "another-run",
		replacing(CodeSHA256(oldCode)), testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "changed since the plan was checked")
}

func TestAuthoringSaveRecordsSpecAndApproval(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.InstallerSHA256)
	in.Name = "Reviewed verification"
	in.Code = "appId: eu.europa.ec.euidi\n---\n- launchApp\n- openLink: ${deeplink}\n" +
		"- tapOn: Share\n- assertVisible: Shared\n- takeScreenshot: shared\n"
	in.CandidateSHA256 = CodeSHA256(in.Code)
	in.Spec = Spec{
		Slot:          "verify-credential",
		Outcome:       OutcomeRejection,
		Prerequisites: []PrerequisiteSpec{{Action: "org/wallet/onboarding"}},
		Parameters: []ParameterSpec{{
			Name:   "deeplink",
			Source: SourceSpec{Kind: SourceLiteral, Value: "secret test value"},
		}},
	}

	receipt, err := Save(app, namespace, "workflow", "run", in, testApproval)

	require.NoError(t, err)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, "verify-credential", record.GetString("category"))
	assert.NotContains(t, record.GetString("authoring_provenance"), "secret test value")
	assert.Equal(t, "verify-credential", receipt.Provenance.Slot)
	assert.Equal(t, OutcomeRejection, receipt.Provenance.Outcome)
	assert.Equal(t, []string{"org/wallet/onboarding"}, receipt.Provenance.Prerequisites)
	assert.Equal(t, []ProvenanceParameter{{Name: "deeplink", Kind: SourceLiteral}},
		receipt.Provenance.Parameters)
	assert.Equal(t, "approver", receipt.Provenance.ApproverID)
	assert.Equal(t, "2027-01-15T08:00:00Z", receipt.Provenance.ApprovedAt)

	// The action targets only the wallet package; the browser opens through openLink.
	in.Name = "Chrome verification"
	in.Code = strings.Replace(in.Code, "eu.europa.ec.euidi", "com.android.chrome", 1)
	in.CandidateSHA256 = CodeSHA256(in.Code)
	_, err = Save(app, namespace, "workflow-2", "run", in, testApproval)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "header appId")
}

func TestAuthoringSaveWaitAction(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.InstallerSHA256)
	in.Name = "Wait past expiry"
	in.Spec = Spec{Slot: "other", Outcome: OutcomeWait, WaitSeconds: 240}

	// The onboarding candidate asserts an outcome instead of waiting.
	_, err = Save(app, namespace, "workflow", "run", in, testApproval)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "must wait 240 s: extendedWaitUntil with visible: <text>, "+
		"optional: true and timeout: 240000")

	in.Code = "appId: eu.europa.ec.euidi\n---\n- launchApp\n- extendedWaitUntil:\n" +
		"    visible: credimi-never-shown\n    optional: true\n    timeout: 240000\n" +
		"- takeScreenshot: waited\n"
	in.CandidateSHA256 = CodeSHA256(in.Code)
	receipt, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.Equal(t, OutcomeWait, receipt.Provenance.Outcome)
	assert.Equal(t, 240, receipt.Provenance.WaitSeconds)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, "other", record.GetString("category"))
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(record.GetString("authoring_provenance")), &stored))
	assert.InDelta(t, 240, stored["wait_seconds"], 0)
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
	_, err = Save(app, namespace, "workflow", "run", in, testApproval)
	require.ErrorIs(t, err, ErrConflict)

	in = acceptedRequest(request, build.InstallerSHA256)
	receipt, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.Equal(t, PlatformIOS, receipt.Provenance.Platform)
	assert.Equal(t, CodeSHA256("selected iOS app zip"), receipt.Provenance.InstallerSHA256)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(record.GetString("authoring_provenance")), &stored))
	assert.Equal(t, "ios", stored["platform"])
	assert.Equal(t, build.InstallerSHA256, stored["installer_sha256"])

	// A retry of the same run is idempotent only for the same platform.
	retried, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.Equal(t, receipt.RecordID, retried.RecordID)
	android := acceptedRequest(request, CodeSHA256("other APK bytes"))
	android.Platform = PlatformAndroid
	_, err = Save(app, namespace, "workflow", "run", android, testApproval)
	require.ErrorIs(t, err, ErrConflict)
}

func TestAuthoringSaveRejectsUnreplayedOrChangedCandidate(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	for _, mutate := range []func(*SaveRequest){
		func(in *SaveRequest) { in.ReplayCount = 0 },
		func(in *SaveRequest) { in.Code += "# not approved\n" },
		func(in *SaveRequest) { in.PackageID = "org.wrong.wallet" },
		func(in *SaveRequest) { in.Name = " " },
		func(in *SaveRequest) { in.Spec.Slot = "install-app" },
		func(in *SaveRequest) {
			in.Code = "appId: eu.europa.ec.euidi\n---\n- launchApp\n- runScript: x.js\n" +
				"- assertVisible: Home\n- takeScreenshot: done\n"
			in.CandidateSHA256 = CodeSHA256(in.Code)
		},
	} {
		in := acceptedRequest(request, build.InstallerSHA256)
		mutate(&in)
		_, err := Save(app, namespace, "workflow", "run", in, testApproval)
		require.ErrorIs(t, err, ErrInvalid)
	}
	_, err = Save(
		app,
		"another-organization",
		"workflow",
		"run",
		acceptedRequest(request, build.InstallerSHA256),
		testApproval,
	)
	require.ErrorIs(t, err, ErrForbidden)
}

const noIOSStore = "the iOS simulator has no App Store: upload a simulator .app as a wallet version"

// seedStoreWallet adds a wallet of the fixture organization with the given
// store fields and returns its `organization/wallet` identifier.
func seedStoreWallet(
	t *testing.T,
	app *tests.TestApp,
	selection Selection,
	name string,
	fields map[string]any,
) string {
	t.Helper()
	wallets, err := app.FindCollectionByNameOrId("wallets")
	require.NoError(t, err)
	wallet := core.NewRecord(wallets)
	wallet.Set("owner", selection.Organization.Id)
	wallet.Set("name", name)
	for field, value := range fields {
		wallet.Set(field, value)
	}
	require.NoError(t, app.Save(wallet))
	return selection.Organization.GetString("canonified_name") + "/" +
		wallet.GetString("canonified_name")
}

func TestSelectStoreWallet(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	assert.Equal(t, SourceInstaller, selection.Source)
	assert.Empty(t, selection.PackageID)

	googleApp := seedStoreWallet(t, app, selection, "Play wallet", map[string]any{
		"google_app_id": "com.credenco.wallet",
	})
	playURL := seedStoreWallet(t, app, selection, "Play URL wallet", map[string]any{
		"playstore_url": "https://play.google.com/store/apps/details?id=com.sphereon.ssi.wallet&hl=en",
	})
	appleOnly := seedStoreWallet(t, app, selection, "Apple wallet", map[string]any{
		"apple_app_id": "co.talao.wallet",
		"appstore_url": "https://apps.apple.com/app/talao/id1582183266",
	})
	foreignURL := seedStoreWallet(t, app, selection, "Foreign URL wallet", map[string]any{
		"playstore_url": "https://example.org/store/apps/details?id=com.example.wallet",
	})
	otherHost := seedStoreWallet(t, app, selection, "Other host wallet", map[string]any{
		"google_app_id": "com.example.otherhost",
		"playstore_url": "https://example.org/store/apps/details?id=com.example.otherhost",
	})
	idMismatch := seedStoreWallet(t, app, selection, "Id mismatch wallet", map[string]any{
		"google_app_id": "com.example.mismatch",
		"playstore_url": "https://play.google.com/store/apps/details?id=com.example.another",
	})
	matching := seedStoreWallet(t, app, selection, "Matching URL wallet", map[string]any{
		"google_app_id": "com.example.matching",
		"playstore_url": "https://play.google.com/store/apps/details?id=com.example.matching&hl=it",
	})

	for _, tc := range []struct {
		name, identifier, platform, pkg, storeURL string
		storeLinks                                []string
	}{
		{
			"android by google_app_id", googleApp, PlatformAndroid, "com.credenco.wallet",
			"https://play.google.com/store/apps/details?id=com.credenco.wallet",
			[]string{
				"market://details?id=com.credenco.wallet",
				"https://play.google.com/store/apps/details?id=com.credenco.wallet",
			},
		},
		{
			"android by playstore_url", playURL, PlatformAndroid, "com.sphereon.ssi.wallet",
			"https://play.google.com/store/apps/details?id=com.sphereon.ssi.wallet&hl=en",
			[]string{
				"market://details?id=com.sphereon.ssi.wallet",
				"https://play.google.com/store/apps/details?id=com.sphereon.ssi.wallet",
				"https://play.google.com/store/apps/details?id=com.sphereon.ssi.wallet&hl=en",
			},
		},
		{
			"android playstore_url on another host", otherHost, PlatformAndroid, "com.example.otherhost",
			"https://play.google.com/store/apps/details?id=com.example.otherhost",
			[]string{
				"market://details?id=com.example.otherhost",
				"https://play.google.com/store/apps/details?id=com.example.otherhost",
			},
		},
		{
			"android playstore_url of another app", idMismatch, PlatformAndroid, "com.example.mismatch",
			"https://play.google.com/store/apps/details?id=com.example.mismatch",
			[]string{
				"market://details?id=com.example.mismatch",
				"https://play.google.com/store/apps/details?id=com.example.mismatch",
			},
		},
		{
			"android matching playstore_url", matching, PlatformAndroid, "com.example.matching",
			"https://play.google.com/store/apps/details?id=com.example.matching&hl=it",
			[]string{
				"market://details?id=com.example.matching",
				"https://play.google.com/store/apps/details?id=com.example.matching",
				"https://play.google.com/store/apps/details?id=com.example.matching&hl=it",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := request
			in.WalletVersionIdentifier = tc.identifier
			in.Platform = tc.platform
			selected, err := Select(app, namespace, in)
			require.NoError(t, err)
			assert.Nil(t, selected.Version)
			assert.Empty(t, selected.Installer())
			assert.Equal(t, SourceStore, selected.Source)
			assert.Equal(t, tc.pkg, selected.PackageID)
			assert.Equal(t, tc.storeURL, selected.StoreURL)
			assert.Equal(t, tc.identifier, selected.Identifier)
			build, err := Resolve(app, namespace, in)
			require.NoError(t, err)
			assert.Equal(t, Build{
				WalletID:          selected.Wallet.Id,
				VersionIdentifier: tc.identifier,
				OwnerNamespace:    namespace,
				Platform:          tc.platform,
				Source:            SourceStore,
				PackageID:         tc.pkg,
				StoreURL:          tc.storeURL,
			}, build)
			assert.Equal(
				t,
				tc.storeLinks,
				StoreLinks(SlotInstallApp, tc.platform, build.PackageID, build.StoreURL),
			)
		})
	}

	for _, tc := range []struct {
		name, identifier, platform, message string
	}{
		{"android without app", appleOnly, PlatformAndroid, "wallet has no Play Store app (google_app_id)"},
		{"android foreign store url", foreignURL, PlatformAndroid, "wallet has no Play Store app (google_app_id)"},
		{"ios", appleOnly, PlatformIOS, noIOSStore},
		{"ios with a Play Store app", googleApp, PlatformIOS, noIOSStore},
		{"missing platform", googleApp, "", "platform must be android or ios"},
		{"unknown wallet", namespace + "/missing-wallet", PlatformAndroid, "wallet not found"},
		{"one segment", namespace, PlatformAndroid, "expected organization/wallet/version or organization/wallet"},
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

	// The store path checks the namespace and the owner role like the version path.
	in := request
	in.WalletVersionIdentifier = googleApp
	_, err = Select(app, "another-organization", in)
	require.ErrorIs(t, err, ErrForbidden)
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	in.ActorID = other.Id
	_, err = Select(app, namespace, in)
	require.ErrorIs(t, err, ErrForbidden)
}

func TestListBuildOptionsIncludesPlayStoreWallets(t *testing.T) {
	app, request, _ := authoringFixture(t)
	selection, err := Select(app, "", request)
	require.NoError(t, err)
	both := seedStoreWallet(t, app, selection, "Both stores wallet", map[string]any{
		"google_app_id": "com.example.both",
		"apple_app_id":  "com.example.both",
		"appstore_url":  "https://apps.apple.com/app/both/id1",
	})
	seedStoreWallet(t, app, selection, "No store wallet", nil)

	options, err := ListBuildOptions(app, request.Actor)
	require.NoError(t, err)
	byIdentifier := map[string]BuildOption{}
	identifiers := make([]string, 0, len(options))
	for _, option := range options {
		byIdentifier[option.Identifier] = option
		identifiers = append(identifiers, option.Identifier)
	}
	assert.IsIncreasing(t, identifiers)
	assert.Len(t, options, 2)
	assert.Equal(t, BuildOption{
		Identifier:       both,
		WalletName:       "Both stores wallet",
		OrganizationName: "userA's organization",
		VersionTag:       StoreVersionTag,
		Platforms:        []string{PlatformAndroid},
		Source:           SourceStore,
	}, byIdentifier[both])
	assert.Equal(t, SourceInstaller, byIdentifier[request.WalletVersionIdentifier].Source)
}

func TestAuthoringSaveStoreBuild(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	request.WalletVersionIdentifier = seedStoreWallet(
		t,
		app,
		selection,
		"Store wallet",
		map[string]any{
			"google_app_id": "eu.europa.ec.euidi",
		},
	)
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)

	in := acceptedRequest(request, build.InstallerSHA256)
	in.InstalledVersion = "2026.10.1"
	other := in
	other.Code = strings.ReplaceAll(in.Code, "eu.europa.ec.euidi", "org.other.wallet")
	other.CandidateSHA256 = CodeSHA256(other.Code)
	other.PackageID = "org.other.wallet"
	_, err = Save(app, namespace, "workflow", "run", other, testApproval)
	require.ErrorIs(t, err, ErrConflict)
	assert.ErrorContains(t, err, "wallet app changed")
	digest := in
	digest.InstallerSHA256 = strings.Repeat("0", 64)
	_, err = Save(app, namespace, "workflow", "run", digest, testApproval)
	require.ErrorIs(t, err, ErrInvalid)

	receipt, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.Equal(t, SourceStore, receipt.Provenance.Source)
	assert.Empty(t, receipt.Provenance.WalletVersionID)
	assert.Empty(t, receipt.Provenance.InstallerSHA256)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, build.WalletID, record.GetString("wallet"))
	var stored map[string]any
	require.NoError(t, json.Unmarshal([]byte(record.GetString("authoring_provenance")), &stored))
	assert.Equal(t, "store", stored["source"])
	assert.Equal(t, "2026.10.1", stored["installed_version"])
	assert.Equal(t, "", stored["installer_sha256"])
	assert.Equal(t, request.WalletVersionIdentifier, stored["wallet_version_identifier"])
	retried, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	assert.Equal(t, receipt.RecordID, retried.RecordID)
}

func TestInstallAppSlotNeedsStoreWallet(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	install := Spec{Slot: "install-app", Outcome: OutcomeSuccess}
	code := "appId: eu.europa.ec.euidi\n---\n" +
		"- openLink:\n    link: \"market://details?id=eu.europa.ec.euidi\"\n    autoVerify: false\n" +
		"- runFlow:\n    when:\n      visible: \"Install\"\n    commands:\n      - tapOn: \"Install\"\n" +
		"- extendedWaitUntil:\n    visible: \"Open\"\n    timeout: 300000\n" +
		"- takeScreenshot: store-installed\n"

	// A wallet version's installer build has no install-app action.
	_, err = CheckSpecRecords(app, selection, &install)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "the install-app slot needs a wallet selected from the store")
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.InstallerSHA256)
	in.Name = "Install"
	in.Code = code
	in.CandidateSHA256 = CodeSHA256(code)
	in.Spec = install
	_, err = Save(app, namespace, "workflow", "run", in, testApproval)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "the install-app slot needs a wallet selected from the store")

	// A store wallet saves it, opening its store page; the page is refused for other slots.
	request.WalletVersionIdentifier = seedStoreWallet(
		t,
		app,
		selection,
		"Store wallet",
		map[string]any{
			"google_app_id": "eu.europa.ec.euidi",
		},
	)
	store, err := Select(app, namespace, request)
	require.NoError(t, err)
	_, err = CheckSpecRecords(app, store, &install)
	require.NoError(t, err)
	in = acceptedRequest(request, "")
	in.Name = "Install"
	in.Code = code
	in.CandidateSHA256 = CodeSHA256(code)
	in.Spec = install
	receipt, err := Save(app, namespace, "workflow", "run", in, testApproval)
	require.NoError(t, err)
	record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
	require.NoError(t, err)
	assert.Equal(t, "install-app", record.GetString("category"))
	in.Name = "Onboarding with store page"
	in.Spec = Spec{Slot: "onboarding", Outcome: OutcomeSuccess}
	_, err = Save(app, namespace, "workflow", "run", in, testApproval)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "openLink is not allowed: this session has no parameters")
}

func TestInstallAppSlotRefusedOnIOS(t *testing.T) {
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	request.WalletVersionIdentifier = seedVersion(
		t, app, selection, "ios-build", nil, []byte("selected iOS app zip"),
	)
	request.Platform = PlatformIOS
	ios, err := Select(app, namespace, request)
	require.NoError(t, err)
	install := Spec{Slot: SlotInstallApp, Outcome: OutcomeSuccess}
	code := "appId: eu.europa.ec.euidi\n---\n- launchApp\n- takeScreenshot: installed\n"

	// iOS builds are simulator installers, never store wallets: the store-only
	// rule refuses install-app in the plan check and on save.
	_, err = CheckSpecRecords(app, ios, &install)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "the install-app slot needs a wallet selected from the store")
	build, err := Resolve(app, namespace, request)
	require.NoError(t, err)
	in := acceptedRequest(request, build.InstallerSHA256)
	in.Name = "Install"
	in.Code = code
	in.CandidateSHA256 = CodeSHA256(code)
	in.Spec = install
	_, err = Save(app, namespace, "workflow", "run", in, testApproval)
	require.ErrorIs(t, err, ErrInvalid)
	assert.ErrorContains(t, err, "the install-app slot needs a wallet selected from the store")
}

// publicWalletFixture returns the fixture with userA's selection and userB as
// the caller, with userB's organization record.
func publicWalletFixture(t *testing.T) (*tests.TestApp, Selection, Actor, *core.Record) {
	t.Helper()
	app, request, namespace := authoringFixture(t)
	selection, err := Select(app, namespace, request)
	require.NoError(t, err)
	other, err := app.FindAuthRecordByEmail("users", "userB@example.org")
	require.NoError(t, err)
	orgID, err := pbutils.GetUserOrganizationID(app, other.Id)
	require.NoError(t, err)
	org, err := app.FindRecordById("organizations", orgID)
	require.NoError(t, err)
	require.NotEqual(t, selection.Organization.Id, org.Id)
	return app, selection, Actor{ActorID: other.Id}, org
}

func buildOptionsByIdentifier(
	t *testing.T,
	app *tests.TestApp,
	actor Actor,
) map[string]BuildOption {
	t.Helper()
	options, err := ListBuildOptions(app, actor)
	require.NoError(t, err)
	byIdentifier := map[string]BuildOption{}
	for _, option := range options {
		byIdentifier[option.Identifier] = option
	}
	return byIdentifier
}

func setRecordFields(t *testing.T, app *tests.TestApp, record *core.Record, fields map[string]any) {
	t.Helper()
	for field, value := range fields {
		record.Set(field, value)
	}
	require.NoError(t, app.Save(record))
}

func TestPublicWalletOfAnotherOrganization(t *testing.T) {
	const notDownloadable = "the owner of this wallet has not made this version's installer downloadable"
	storeRequest := func(actor Actor, identifier string) BuildRequest {
		return BuildRequest{
			Actor:                   actor,
			WalletVersionIdentifier: identifier,
			Platform:                PlatformAndroid,
		}
	}
	cases := []struct {
		name string
		run  func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, org *core.Record)
	}{
		{
			name: "unpublished wallet",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, _ *core.Record) {
				store := seedStoreWallet(t, app, owner, "Private store wallet", map[string]any{
					"google_app_id": "eu.europa.ec.euidi",
				})
				prefix := owner.Organization.GetString("canonified_name") + "/"
				for identifier := range buildOptionsByIdentifier(t, app, caller) {
					assert.False(t, strings.HasPrefix(identifier, prefix), identifier)
				}
				_, err := Select(app, "", storeRequest(caller, store))
				require.ErrorIs(t, err, ErrForbidden)
			},
		},
		{
			name: "published store wallet",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, org *core.Record) {
				store := seedStoreWallet(t, app, owner, "Public store wallet", map[string]any{
					"google_app_id": "eu.europa.ec.euidi",
					"published":     true,
				})
				option, ok := buildOptionsByIdentifier(t, app, caller)[store]
				require.True(t, ok)
				assert.Equal(t, SourceStore, option.Source)
				selection, err := Select(app, "", storeRequest(caller, store))
				require.NoError(t, err)
				assert.Equal(t, org.Id, selection.Organization.Id)
				assert.Equal(t, owner.Organization.Id, selection.WalletOwner.Id)
				build, err := Resolve(
					app,
					org.GetString("canonified_name"),
					storeRequest(caller, store),
				)
				require.NoError(t, err)
				assert.Equal(
					t,
					owner.Organization.GetString("canonified_name"),
					build.OwnerNamespace,
				)
			},
		},
		{
			name: "published wallet, version not downloadable",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, _ *core.Record) {
				setRecordFields(t, app, owner.Wallet, map[string]any{"published": true})
				_, listed := buildOptionsByIdentifier(t, app, caller)[owner.Identifier]
				assert.False(t, listed)
				_, err := Select(app, "", storeRequest(caller, owner.Identifier))
				require.ErrorIs(t, err, ErrForbidden)
				assert.ErrorContains(t, err, notDownloadable)
			},
		},
		{
			name: "published wallet, version downloadable",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, _ *core.Record) {
				setRecordFields(t, app, owner.Wallet, map[string]any{"published": true})
				setRecordFields(t, app, owner.Version, map[string]any{"downloadable": true})
				option, listed := buildOptionsByIdentifier(t, app, caller)[owner.Identifier]
				require.True(t, listed)
				assert.Equal(t, SourceInstaller, option.Source)
				_, err := Select(app, "", storeRequest(caller, owner.Identifier))
				require.NoError(t, err)
			},
		},
		{
			name: "unpublished wallet, version downloadable",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, _ *core.Record) {
				setRecordFields(t, app, owner.Version, map[string]any{"downloadable": true})
				_, listed := buildOptionsByIdentifier(t, app, caller)[owner.Identifier]
				assert.False(t, listed)
				_, err := Select(app, "", storeRequest(caller, owner.Identifier))
				require.ErrorIs(t, err, ErrForbidden)
			},
		},
		{
			name: "save after the wallet was unpublished is refused",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, org *core.Record) {
				store := seedStoreWallet(t, app, owner, "Public store wallet", map[string]any{
					"google_app_id": "eu.europa.ec.euidi",
					"published":     true,
				})
				namespace := org.GetString("canonified_name")
				request := storeRequest(caller, store)
				build, err := Resolve(app, namespace, request)
				require.NoError(t, err)
				wallet, err := app.FindFirstRecordByFilter(
					"wallets",
					"canonified_name = {:name} && owner = {:owner}",
					dbx.Params{
						"name": strings.TrimPrefix(
							store,
							owner.Organization.GetString("canonified_name")+"/",
						),
						"owner": owner.Organization.Id,
					},
				)
				require.NoError(t, err)
				setRecordFields(t, app, wallet, map[string]any{"published": false})
				_, err = Save(
					app, namespace, "workflow", "run",
					acceptedRequest(request, build.InstallerSHA256), testApproval,
				)
				require.ErrorIs(t, err, ErrForbidden)
				saved, err := app.FindAllRecords("wallet_actions", dbx.HashExp{"wallet": wallet.Id})
				require.NoError(t, err)
				assert.Empty(t, saved)
			},
		},
		{
			name: "session organization is not the caller's",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, _ *core.Record) {
				store := seedStoreWallet(t, app, owner, "Public store wallet", map[string]any{
					"google_app_id": "eu.europa.ec.euidi",
					"published":     true,
				})
				_, err := Select(
					app,
					owner.Organization.GetString("canonified_name"),
					storeRequest(caller, store),
				)
				require.ErrorIs(t, err, ErrForbidden)
			},
		},
		{
			name: "save is owned by the caller's organization",
			run: func(t *testing.T, app *tests.TestApp, owner Selection, caller Actor, org *core.Record) {
				store := seedStoreWallet(t, app, owner, "Public store wallet", map[string]any{
					"google_app_id": "eu.europa.ec.euidi",
					"published":     true,
				})
				namespace := org.GetString("canonified_name")
				request := storeRequest(caller, store)
				build, err := Resolve(app, namespace, request)
				require.NoError(t, err)
				receipt, err := Save(
					app, namespace, "workflow", "run",
					acceptedRequest(request, build.InstallerSHA256), testApproval,
				)
				require.NoError(t, err)
				record, err := app.FindRecordById("wallet_actions", receipt.RecordID)
				require.NoError(t, err)
				assert.Equal(t, org.Id, record.GetString("owner"))
				wallet, err := app.FindRecordById("wallets", record.GetString("wallet"))
				require.NoError(t, err)
				assert.Equal(t, store, owner.Organization.GetString("canonified_name")+"/"+
					wallet.GetString("canonified_name"))
				assert.False(t, record.GetBool("published"))
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, owner, caller, org := publicWalletFixture(t)
			tc.run(t, app, owner, caller, org)
		})
	}
}
