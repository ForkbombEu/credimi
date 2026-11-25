// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package actionauthor binds local authoring sessions to authorized Credimi builds.
package actionauthor

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/forkbombeu/credimi/pkg/fcaf/walletroles"
	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// ErrForbidden denies unauthorized actors or organization namespaces.
// ErrInvalid rejects malformed authoring input.
// ErrConflict rejects stale approvals and unconfirmed replacements.
var (
	ErrForbidden = errors.New("authoring access denied")
	ErrInvalid   = errors.New("invalid authoring request")
	ErrConflict  = errors.New("authoring approval is stale or conflicts with an existing action")
)

// Actor identifies the authenticated Credimi user that started the workflow.
type Actor struct {
	ActorID string `json:"actor_id"`
}

// Supported authoring platforms; each selects the wallet version installer field.
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
)

// installerFields maps each platform to its wallet_versions installer field and
// the message returned when the selected version lacks that installer.
var installerFields = map[string]struct{ field, missing string }{
	PlatformAndroid: {field: "android_installer", missing: "selected version has no Android APK"},
	PlatformIOS:     {field: "ios_installer", missing: "selected version has no iOS app"},
}

// Build sources: an installer uploaded to a wallet version, or the wallet's
// Play Store app installed at session start (Android only: the iOS simulator
// has no App Store).
const (
	SourceInstaller = "installer"
	SourceStore     = "store"
	// StoreVersionTag labels store builds in the build options.
	StoreVersionTag = "store install"
)

// appIDPattern matches an Android package identifier.
var appIDPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*(\.[A-Za-z0-9_-]+)+$`)

// BuildRequest selects an existing wallet version and platform on behalf of its authenticated actor.
type BuildRequest struct {
	Actor
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	Platform                string `json:"platform"`
}

// Build binds an authorized wallet version to its platform installer digest,
// or an authorized wallet to its store app. VersionID and InstallerSHA256 are
// empty for store builds.
type Build struct {
	WalletID          string `json:"wallet_id"`
	VersionID         string `json:"version_id"`
	VersionIdentifier string `json:"version_identifier"`
	OwnerNamespace    string `json:"owner_namespace"`
	Platform          string `json:"platform"`
	InstallerSHA256   string `json:"installer_sha256"`
	Source            string `json:"source"`
	PackageID         string `json:"package_id,omitempty"`
	StoreURL          string `json:"store_url,omitempty"`
}

// Selection holds the records reauthorized for a build operation. Version is
// nil for store selections, whose app is PackageID on the store page StoreURL.
// Organization is the caller's organization, which owns saved actions and
// whose records the session may use; WalletOwner owns the wallet and its
// build, and differs from Organization for published wallets of other
// organizations.
type Selection struct {
	Version      *core.Record
	Wallet       *core.Record
	Organization *core.Record
	WalletOwner  *core.Record
	Actor        Actor
	Identifier   string
	Platform     string
	Source       string
	PackageID    string
	StoreURL     string
}

// Installer returns the stored filename of the selected platform's installer,
// empty for store selections.
func (s Selection) Installer() string {
	if s.Version == nil {
		return ""
	}
	return s.Version.GetString(installerFields[s.Platform].field)
}

// BuildOption describes one build the actor can select for authoring and its platforms.
type BuildOption struct {
	Identifier       string   `json:"identifier"`
	WalletName       string   `json:"wallet_name"`
	OrganizationName string   `json:"organization_name"`
	VersionTag       string   `json:"version_tag"`
	Platforms        []string `json:"platforms"`
	Source           string   `json:"source"`
}

// errNoIOSStore refuses a store build on iOS: the action author's only iOS
// devices are simulators, which have no App Store.
var errNoIOSStore = fmt.Errorf(
	"%w: the iOS simulator has no App Store: upload a simulator .app as a wallet version",
	ErrInvalid,
)

// playStoreApp returns the wallet's Play Store app and its store page.
func playStoreApp(wallet *core.Record) (string, string, error) {
	pkg := wallet.GetString("google_app_id")
	storeURL := wallet.GetString("playstore_url")
	if pkg == "" && storeURL != "" {
		if parsed, err := url.Parse(storeURL); err == nil && parsed.Host == "play.google.com" {
			pkg = parsed.Query().Get("id")
		}
	}
	if !appIDPattern.MatchString(pkg) {
		return "", "", fmt.Errorf(
			"%w: wallet has no Play Store app (google_app_id)",
			ErrInvalid,
		)
	}
	return pkg, playStorePage(storeURL, pkg), nil
}

// playStorePage returns storeURL when it is an https page on play.google.com
// of pkg (its id query parameter), the Play Store details page of pkg otherwise.
func playStorePage(storeURL, pkg string) string {
	parsed, err := url.Parse(storeURL)
	if err == nil && parsed.Scheme == "https" && parsed.Host == "play.google.com" &&
		parsed.Query().Get("id") == pkg {
		return storeURL
	}
	return "https://play.google.com/store/apps/details?id=" + pkg
}

// ListBuildOptions returns the builds Select authorizes for the actor: builds
// of the caller's organization's wallets and of published wallets of other
// organizations (store builds, and downloadable versions). A user without an
// organization gets none.
func ListBuildOptions(app core.App, in Actor) ([]BuildOption, error) {
	actor, err := findActor(app, in)
	if err != nil {
		return nil, err
	}
	orgID, err := pbutils.GetUserOrganizationID(app, actor.Id)
	if errors.Is(err, sql.ErrNoRows) {
		return []BuildOption{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find actor organization: %w", err)
	}
	filter := []dbx.Expression{
		dbx.Or(dbx.NewExp("[[android_installer]] != ''"), dbx.NewExp("[[ios_installer]] != ''")),
		dbx.Or(
			dbx.HashExp{"owner": orgID},
			dbx.NewExp(
				"[[downloadable]] = TRUE AND [[owner]] IN (SELECT w.[[owner]] FROM [[wallets]] w WHERE w.[[id]] = [[wallet_versions.wallet]] AND w.[[published]] = TRUE)",
			),
		),
	}
	versions, err := app.FindAllRecords("wallet_versions", filter...)
	if err != nil {
		return nil, fmt.Errorf("list authoring wallet versions: %w", err)
	}
	if failures := app.ExpandRecords(
		versions,
		[]string{"wallet", "owner"},
		nil,
	); len(
		failures,
	) > 0 {
		return nil, fmt.Errorf("expand authoring wallet versions: %v", failures)
	}
	options := make([]BuildOption, 0, len(versions))
	for _, version := range versions {
		wallet := version.ExpandedOne("wallet")
		organization := version.ExpandedOne("owner")
		if wallet == nil || organization == nil || wallet.GetString("owner") != organization.Id {
			continue
		}
		platforms := make([]string, 0, len(installerFields))
		for _, platform := range []string{PlatformAndroid, PlatformIOS} {
			if version.GetString(installerFields[platform].field) != "" {
				platforms = append(platforms, platform)
			}
		}
		options = append(options, BuildOption{
			Identifier: strings.Join([]string{
				organization.GetString("canonified_name"),
				wallet.GetString("canonified_name"),
				version.GetString("canonified_tag"),
			}, "/"),
			WalletName:       wallet.GetString("name"),
			OrganizationName: organization.GetString("name"),
			VersionTag:       version.GetString("tag"),
			Platforms:        platforms,
			Source:           SourceInstaller,
		})
	}
	stores, err := storeBuildOptions(app, orgID)
	if err != nil {
		return nil, err
	}
	options = append(options, stores...)
	sort.Slice(options, func(i, j int) bool {
		return options[i].Identifier < options[j].Identifier
	})
	return options, nil
}

// storeBuildOptions lists the wallets of the actor's organization and the
// published wallets of other organizations that have a Play Store app.
func storeBuildOptions(app core.App, orgID string) ([]BuildOption, error) {
	wallets, err := app.FindAllRecords(
		"wallets",
		dbx.Or(dbx.HashExp{"owner": orgID}, dbx.HashExp{"published": true}),
	)
	if err != nil {
		return nil, fmt.Errorf("list authoring wallets: %w", err)
	}
	if failures := app.ExpandRecords(wallets, []string{"owner"}, nil); len(failures) > 0 {
		return nil, fmt.Errorf("expand authoring wallets: %v", failures)
	}
	options := make([]BuildOption, 0, len(wallets))
	for _, wallet := range wallets {
		organization := wallet.ExpandedOne("owner")
		if organization == nil {
			continue
		}
		if _, _, err := playStoreApp(wallet); err != nil {
			continue
		}
		options = append(options, BuildOption{
			Identifier: organization.GetString("canonified_name") + "/" +
				wallet.GetString("canonified_name"),
			WalletName:       wallet.GetString("name"),
			OrganizationName: organization.GetString("name"),
			VersionTag:       StoreVersionTag,
			Platforms:        []string{PlatformAndroid},
			Source:           SourceStore,
		})
	}
	return options, nil
}

// findActor accepts only users that still exist.
func findActor(app core.App, in Actor) (*core.Record, error) {
	if in.ActorID == "" {
		return nil, ErrForbidden
	}
	actor, err := app.FindRecordById("users", in.ActorID)
	if err != nil {
		return nil, ErrForbidden
	}
	return actor, nil
}

// Select authorizes members of an organization, the same rule as pipeline
// runs: the actor's organization is its first one. The wallet must be its
// own, or a published wallet of another organization, whose versions are
// accepted only when their installer is downloadable.
// `organization/wallet/version` selects an uploaded installer and
// `organization/wallet` the wallet's Play Store app (Android only).
// organization is the canonified_name of the caller's organization, the one
// the session acts for. An empty organization skips that check; activities
// must pass the organization of their verified session, never a value from
// their payload.
func Select(app core.App, organization string, in BuildRequest) (Selection, error) {
	var selection Selection
	actor, err := findActor(app, in.Actor)
	if err != nil {
		return selection, err
	}
	segments := pathSegments(in.WalletVersionIdentifier)
	var version, wallet *core.Record
	switch len(segments) {
	case 3:
		version, err = app.FindFirstRecordByFilter(
			"wallet_versions",
			"canonified_tag = {:tag} && wallet.canonified_name = {:wallet} && owner.canonified_name = {:org}",
			dbx.Params{"tag": segments[2], "wallet": segments[1], "org": segments[0]},
		)
		if err != nil {
			return selection, fmt.Errorf("%w: wallet version not found", ErrInvalid)
		}
		wallet, err = app.FindRecordById("wallets", version.GetString("wallet"))
		if err != nil || wallet.GetString("owner") != version.GetString("owner") {
			return selection, fmt.Errorf("%w: invalid wallet ownership", ErrForbidden)
		}
	case 2:
		wallet, err = app.FindFirstRecordByFilter(
			"wallets",
			"canonified_name = {:wallet} && owner.canonified_name = {:org}",
			dbx.Params{"wallet": segments[1], "org": segments[0]},
		)
		if err != nil {
			return selection, fmt.Errorf("%w: wallet not found", ErrInvalid)
		}
	default:
		return selection, fmt.Errorf(
			"%w: expected organization/wallet/version or organization/wallet",
			ErrInvalid,
		)
	}
	caller, walletOwner, foreign, err := walletAccess(app, organization, actor, wallet)
	if err != nil {
		return selection, err
	}
	installer, ok := installerFields[in.Platform]
	if !ok {
		return selection, fmt.Errorf("%w: platform must be android or ios", ErrInvalid)
	}
	selection = Selection{
		Version:      version,
		Wallet:       wallet,
		Organization: caller,
		WalletOwner:  walletOwner,
		Actor:        in.Actor,
		Identifier:   strings.Join(segments, "/"),
		Platform:     in.Platform,
		Source:       SourceInstaller,
	}
	if version == nil {
		if in.Platform != PlatformAndroid {
			return Selection{}, errNoIOSStore
		}
		pkg, storeURL, err := playStoreApp(wallet)
		if err != nil {
			return Selection{}, err
		}
		selection.Source = SourceStore
		selection.PackageID = pkg
		selection.StoreURL = storeURL
		return selection, nil
	}
	if foreign && !version.GetBool("downloadable") {
		return Selection{}, fmt.Errorf(
			"%w: the owner of this wallet has not made this version's installer downloadable",
			ErrForbidden,
		)
	}
	if version.GetString(installer.field) == "" {
		return Selection{}, fmt.Errorf("%w: %s", ErrInvalid, installer.missing)
	}
	return selection, nil
}

// SelectWallet authorizes the caller on the wallet walletPath
// (organization/wallet, canonified) as Select does, without a build: the
// selection has no version, platform or package. The FCAF coverage endpoint
// uses it. organization must not be empty. A wallet that is not found is
// ErrForbidden, as one the caller cannot author.
func SelectWallet(
	app core.App,
	organization string,
	in Actor,
	walletPath string,
) (Selection, error) {
	if organization == "" {
		return Selection{}, fmt.Errorf("%w: no organization", ErrForbidden)
	}
	actor, err := findActor(app, in)
	if err != nil {
		return Selection{}, err
	}
	segments := pathSegments(walletPath)
	if len(segments) != 2 {
		return Selection{}, fmt.Errorf("%w: expected organization/wallet", ErrInvalid)
	}
	wallet, err := app.FindFirstRecordByFilter(
		"wallets",
		"canonified_name = {:wallet} && owner.canonified_name = {:org}",
		dbx.Params{"wallet": segments[1], "org": segments[0]},
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Selection{}, fmt.Errorf("%w: wallet not found", ErrForbidden)
	}
	if err != nil {
		return Selection{}, fmt.Errorf("find wallet: %w", err)
	}
	caller, walletOwner, _, err := walletAccess(app, organization, actor, wallet)
	if err != nil {
		return Selection{}, err
	}
	return Selection{
		Wallet:       wallet,
		Organization: caller,
		WalletOwner:  walletOwner,
		Actor:        in,
		Identifier:   strings.Join(segments, "/"),
	}, nil
}

// pathSegments splits a canonified build or wallet path; any empty segment
// gives none.
func pathSegments(path string) []string {
	segments := strings.Split(strings.Trim(canonify.NormalizePath(path), "/"), "/")
	for _, segment := range segments {
		if segment == "" {
			return nil
		}
	}
	return segments
}

// walletAccess authorizes actor, acting for organization (empty: any), on
// wallet: their organization's own wallet, or another's published one.
func walletAccess(
	app core.App,
	organization string,
	actor *core.Record,
	wallet *core.Record,
) (caller, walletOwner *core.Record, foreign bool, err error) {
	walletOwner, err = app.FindRecordById("organizations", wallet.GetString("owner"))
	if err != nil || walletOwner.GetString("canonified_name") == "" {
		return nil, nil, false, ErrForbidden
	}
	orgID, err := pbutils.GetUserOrganizationID(app, actor.Id)
	if err != nil {
		return nil, nil, false, ErrForbidden
	}
	caller, err = app.FindRecordById("organizations", orgID)
	if err != nil || (organization != "" && organization != caller.GetString("canonified_name")) {
		return nil, nil, false, ErrForbidden
	}
	foreign = walletOwner.Id != caller.Id
	if foreign && !wallet.GetBool("published") {
		return nil, nil, false, ErrForbidden
	}
	return caller, walletOwner, foreign, nil
}

// installerDigest streams the selected platform's installer rather than retaining it in memory.
func installerDigest(app core.App, selection Selection) (string, error) {
	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", fmt.Errorf("open installer filesystem: %w", err)
	}
	defer fsys.Close()
	file, err := fsys.GetReader(selection.Version.BaseFilesPath() + "/" + selection.Installer())
	if err != nil {
		return "", fmt.Errorf("open selected installer: %w", err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("hash selected installer: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// Resolve hashes the authorized platform installer, the provenance Save checks.
// The runner installs the installer itself, through the pipeline device code.
// Store builds have no installer: the session installs their app from the store.
// organization is the caller's organization, as for Select. OwnerNamespace is
// the wallet owner's namespace, which owns the build.
func Resolve(app core.App, organization string, in BuildRequest) (Build, error) {
	selection, err := Select(app, organization, in)
	if err != nil {
		return Build{}, err
	}
	build := Build{
		WalletID:          selection.Wallet.Id,
		VersionIdentifier: selection.Identifier,
		OwnerNamespace:    selection.WalletOwner.GetString("canonified_name"),
		Platform:          selection.Platform,
		Source:            selection.Source,
		PackageID:         selection.PackageID,
		StoreURL:          selection.StoreURL,
	}
	if selection.Source == SourceStore {
		return build, nil
	}
	build.VersionID = selection.Version.Id
	build.InstallerSHA256, err = installerDigest(app, selection)
	if err != nil {
		return Build{}, err
	}
	return build, nil
}

// SaveRequest carries the trusted workflow's accepted candidate, the
// Credimi-signed approval of its human review and the plan Check action
// author plan returned, which the activity checks again for the session.
// BatchKey is the candidate's key in the session; Candidates maps the key of
// each candidate prerequisite of the plan to its saved record id.
type SaveRequest struct {
	BuildRequest
	Name             string            `json:"name"`
	Code             string            `json:"code"`
	CandidateSHA256  string            `json:"candidate_sha256"`
	InstallerSHA256  string            `json:"installer_sha256"`
	PackageID        string            `json:"package_id"`
	InstalledVersion string            `json:"installed_version,omitempty"`
	ReplayCount      int               `json:"replay_count"`
	Approval         string            `json:"approval"`
	Spec             Spec              `json:"plan"`
	BatchKey         string            `json:"batch_key"`
	Candidates       map[string]string `json:"candidates,omitempty"`
	// FCAF is the FCAF scope of the session, set by the activity.
	FCAF *FCAFScope `json:"-"`
	// FCAFProfile is the agent's notes on the wallet, kept with an FCAF action.
	FCAFProfile map[string]any `json:"fcaf_profile,omitempty"`
	// FCAFResult is the agent's verdict on an FCAF action, kept with it.
	FCAFResult *FCAFResult `json:"fcaf_result,omitempty"`
	// FCAFReplays are the action's replays on fixtures of its role's
	// authoring set, kept with it.
	FCAFReplays []FCAFReplay `json:"fcaf_replays,omitempty"`
}

// maxFCAFProfileBytes bounds the JSON encoding of an FCAF profile.
const maxFCAFProfileBytes = 8192

// ProvenanceParameter records a parameter's name and source kind, never its
// values. Source is the record path of a credential_offer or
// use_case_verification source, already public in the hub; other kinds have none.
type ProvenanceParameter struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Source string `json:"source,omitempty"`
}

// Provenance identifies the exact build, spec, workflow and human approval
// that validated an action. Store builds have no version or installer digest:
// InstalledVersion is the store app version the session reported.
type Provenance struct {
	WalletVersionID         string                `json:"wallet_version_id"`
	WalletVersionIdentifier string                `json:"wallet_version_identifier"`
	Platform                string                `json:"platform"`
	Source                  string                `json:"source"`
	InstallerSHA256         string                `json:"installer_sha256"`
	PackageID               string                `json:"package_id"`
	InstalledVersion        string                `json:"installed_version,omitempty"`
	YAMLSHA256              string                `json:"yaml_sha256"`
	WorkflowID              string                `json:"workflow_id"`
	RunID                   string                `json:"run_id"`
	ReplayCount             int                   `json:"replay_count"`
	Slot                    string                `json:"slot"`
	Outcome                 string                `json:"outcome"`
	WaitSeconds             int                   `json:"wait_seconds,omitempty"`
	Prerequisites           []string              `json:"prerequisites"`
	Parameters              []ProvenanceParameter `json:"parameters"`
	ApproverID              string                `json:"approver_id"`
	ApprovedAt              string                `json:"approved_at"`
	// ReplacedCodeSHA256 is the hash of the code a replacing save overwrote.
	ReplacedCodeSHA256 string `json:"replaced_code_sha256,omitempty"`
	// BatchKey is the candidate key the session gave the action.
	BatchKey string `json:"batch_key,omitempty"`
	// FCAFRole is the FCAF role the action implements, and FCAFRoleSHA256
	// the role's hash when the action was saved.
	FCAFRole       string `json:"fcaf_role,omitempty"`
	FCAFRoleSHA256 string `json:"fcaf_role_sha256,omitempty"`
	// FCAFProfile is the agent's notes on the wallet, given with the save.
	FCAFProfile map[string]any `json:"fcaf_profile,omitempty"`
	// FCAFResult is the agent's verdict on the FCAF action, given with the save.
	FCAFResult *FCAFResult `json:"fcaf_result,omitempty"`
	// FCAFReplays are the action's replays the save carried, and
	// FCAFFixturesSHA256 the FCAFFixturesSHA256 of the role's authoring set
	// when it was saved.
	FCAFReplays        []FCAFReplay `json:"fcaf_replays,omitempty"`
	FCAFFixturesSHA256 string       `json:"fcaf_fixtures_sha256,omitempty"`
}

// Receipt describes the saved record and its validation provenance.
type Receipt struct {
	RecordID   string     `json:"record_id"`
	Name       string     `json:"name"`
	Created    bool       `json:"created"`
	Replaced   bool       `json:"replaced"`
	Published  bool       `json:"published"`
	Provenance Provenance `json:"provenance"`
}

// CodeSHA256 hashes the exact YAML bytes reviewed by the human.
func CodeSHA256(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

// fcafProfilePIN is the shape of an FCAF profile's PIN.
var fcafProfilePIN = regexp.MustCompile(`^[0-9]{4,8}$`)

// Bounds of an FCAF profile's notes.
const (
	maxFCAFProfileNotes     = 20
	maxFCAFProfileNoteRunes = 300
)

// CheckFCAFProfile refuses a profile for an action without an FCAF role, a
// profile whose JSON encoding exceeds 8 KiB, and one off the profile schema:
// an optional pin of 4-8 digits, optional notes, at most 20 strings of
// 1-300 characters, an optional pid_issuer, one of walletroles.PIDIssuers,
// an optional request_scheme, only walletroles.OpenID4VPRequestScheme, and
// nothing else.
func CheckFCAFProfile(profile map[string]any, fcafRole string) error {
	if profile == nil {
		return nil
	}
	encoded, err := json.Marshal(profile)
	if fcafRole == "" || err != nil || len(encoded) > maxFCAFProfileBytes {
		return fmt.Errorf("%w: fcaf_profile is only for FCAF actions, at most 8 KiB", ErrInvalid)
	}
	if !validFCAFProfile(profile) {
		return fmt.Errorf(
			"%w: fcaf_profile: must be {pin: 4-8 digits, notes: at most 20 notes of up to 300 characters, "+
				"pid_issuer: key-attestation or jwt-proof, request_scheme: openid4vp://}",
			ErrInvalid,
		)
	}
	return nil
}

func validFCAFProfile(profile map[string]any) bool {
	for key, value := range profile {
		switch key {
		case "pin":
			pin, ok := value.(string)
			if !ok || !fcafProfilePIN.MatchString(pin) {
				return false
			}
		case "notes":
			if !validFCAFProfileNotes(value) {
				return false
			}
		case "pid_issuer":
			issuer, ok := value.(string)
			if !ok || !slices.Contains(walletroles.PIDIssuers, issuer) {
				return false
			}
		case "request_scheme":
			if value != walletroles.OpenID4VPRequestScheme {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func validFCAFProfileNotes(value any) bool {
	notes, ok := value.([]any)
	if !ok || len(notes) > maxFCAFProfileNotes {
		return false
	}
	for _, raw := range notes {
		note, ok := raw.(string)
		length := utf8.RuneCountInString(note)
		if !ok || length < 1 || length > maxFCAFProfileNoteRunes {
			return false
		}
	}
	return true
}

// FCAFResult is what the agent found when it wrote an FCAF action: the wallet
// did the role (passed), cannot do it (unsupported) or does not conform
// (finding), and its note.
type FCAFResult struct {
	Status string `json:"status"`
	Note   string `json:"note"`
}

// FCAF result statuses, and the bound of a result's note.
const (
	FCAFResultPassed      = "passed"
	FCAFResultUnsupported = "unsupported"
	// FCAFResultFinding: the action is right and the wallet does not conform:
	// it does what the role forbids, or misbehaves on a feature it has.
	FCAFResultFinding      = "finding"
	maxFCAFResultNoteRunes = 2000
)

// IsFCAFResultStatus reports whether status is passed, unsupported or finding.
func IsFCAFResultStatus(status string) bool {
	return status == FCAFResultPassed || status == FCAFResultUnsupported ||
		status == FCAFResultFinding
}

// CheckFCAFResult refuses a result for an action without an FCAF role, a
// status other than passed, unsupported or finding, and a note over 2000 characters.
func CheckFCAFResult(result *FCAFResult, fcafRole string) error {
	switch {
	case result == nil:
		return nil
	case fcafRole == "":
		return fmt.Errorf("%w: fcaf_result: only FCAF actions carry a result", ErrInvalid)
	case !IsFCAFResultStatus(result.Status):
		return fmt.Errorf(
			"%w: fcaf_result: status must be passed, unsupported or finding",
			ErrInvalid,
		)
	case utf8.RuneCountInString(result.Note) > maxFCAFResultNoteRunes:
		return fmt.Errorf("%w: fcaf_result: note is longer than 2000 characters", ErrInvalid)
	}
	return nil
}

// validateAcceptedAction checks the accepted candidate and its code; for an
// FCAF plan it also checks the plan and code against the role, which it
// returns, nil otherwise.
func validateAcceptedAction(in *SaveRequest, selection Selection) (*walletroles.Role, error) {
	if _, ok := Slots[in.Spec.Slot]; !ok || strings.TrimSpace(in.Name) == "" ||
		in.ReplayCount != 1 || in.CandidateSHA256 != CodeSHA256(in.Code) || in.PackageID == "" {
		return nil, fmt.Errorf("%w: missing accepted candidate", ErrInvalid)
	}
	if !candidateKey.MatchString(in.BatchKey) {
		return nil, fmt.Errorf("%w: batch_key must match a<number>", ErrInvalid)
	}
	if err := CheckFCAFProfile(in.FCAFProfile, in.Spec.FCAFRole); err != nil {
		return nil, err
	}
	if err := CheckFCAFResult(in.FCAFResult, in.Spec.FCAFRole); err != nil {
		return nil, err
	}
	scope := CodeScope{
		AppIDs:     AllowedAppIDs(in.PackageID, in.Spec.Slot, selection.Platform),
		Parameters: ParameterNames(in.Spec.Parameters),
		StoreLinks: StoreLinks(
			in.Spec.Slot,
			selection.Platform,
			in.PackageID,
			selection.StoreURL,
		),
		WaitSeconds: in.Spec.WaitSeconds,
	}
	var role *walletroles.Role
	if in.Spec.FCAFRole != "" {
		if in.FCAF == nil {
			return nil, fmt.Errorf("%w: fcaf_role needs an FCAF session", ErrInvalid)
		}
		checked, err := CheckFCAFPlan(in.FCAF, &in.Spec)
		if err != nil {
			return nil, err
		}
		if in.Name != checked.Name {
			return nil, fmt.Errorf(
				"%w: an FCAF action is named after its role: %s", ErrInvalid, checked.Name,
			)
		}
		role = &checked
		packages := map[walletroles.App]string{
			walletroles.AppWallet:  in.PackageID,
			walletroles.AppBrowser: in.FCAF.Contract.Reference.AppIDs[walletroles.AppBrowser],
		}
		scope.AppIDs = FCAFAppIDs(checked, packages)
		scope.StoreLinks = nil
		scope.FCAF = &FCAFCodeScope{Role: checked, Packages: packages}
	}
	if err := CheckFCAFReplays(in.FCAF, in.FCAFReplays, in.Spec.FCAFRole); err != nil {
		return nil, err
	}
	if problems := ValidateCode(in.Code, scope); len(problems) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrInvalid, strings.Join(problems, "; "))
	}
	return role, nil
}

// fcafProvenance checks the resolved starting actions of an FCAF action
// against its role and records the role in the provenance.
func fcafProvenance(provenance *Provenance, in *SaveRequest, role walletroles.Role) error {
	starts := in.FCAF.Contract.StartsFrom(role)
	matches := len(provenance.Prerequisites) == len(starts)
	for i := 0; matches && i < len(starts); i++ {
		matches = path.Base(provenance.Prerequisites[i]) == starts[i]
	}
	if !matches {
		return fmt.Errorf("%w: an FCAF %s action starts from %v", ErrInvalid, role.Name, starts)
	}
	provenance.FCAFRole = role.Name
	provenance.FCAFRoleSHA256 = role.SHA256()
	provenance.FCAFProfile = in.FCAFProfile
	provenance.FCAFResult = in.FCAFResult
	provenance.FCAFReplays = in.FCAFReplays
	set, err := in.FCAF.Contract.AuthoringSet(in.FCAF.Suite, role.Name)
	if err != nil {
		return err
	}
	provenance.FCAFFixturesSHA256 = FCAFFixturesSHA256(set)
	return nil
}

// setFCAFTags merges the FCAF tags of the role into the record's tags; other
// actions (nil role) keep their tags.
func setFCAFTags(record *core.Record, role *walletroles.Role) {
	if role != nil {
		record.Set("tags", mergeTags(record.GetString("tags"), fcafTags(*role)))
	}
}

// provenanceSpec records the session spec without parameter values.
func provenanceSpec(provenance *Provenance, spec Spec) {
	provenance.Slot = spec.Slot
	provenance.Outcome = spec.Outcome
	provenance.WaitSeconds = spec.WaitSeconds
	provenance.Prerequisites = make([]string, 0, len(spec.Prerequisites))
	for _, prerequisite := range spec.Prerequisites {
		provenance.Prerequisites = append(provenance.Prerequisites, prerequisite.Action)
	}
	provenance.Parameters = make([]ProvenanceParameter, 0, len(spec.Parameters))
	for _, parameter := range spec.Parameters {
		recorded := ProvenanceParameter{Name: parameter.Name, Kind: parameter.Source.Kind}
		if _, ok := sourceCollections[parameter.Source.Kind]; ok {
			recorded.Source = canonify.NormalizePath(parameter.Source.Identifier)
		}
		provenance.Parameters = append(provenance.Parameters, recorded)
	}
}

// resolveCandidates sets each candidate prerequisite's Action to the path of
// the record this session saved for it, keeping its key, then checks the
// resolved prerequisites, including their starting actions.
func resolveCandidates(
	app core.App,
	selection Selection,
	workflowID string,
	in *SaveRequest,
) error {
	resolved := false
	for i := range in.Spec.Prerequisites {
		prerequisite := &in.Spec.Prerequisites[i]
		key := prerequisite.Candidate
		if key == "" {
			continue
		}
		record, err := sessionCandidate(app, selection, workflowID, key, in.Candidates[key])
		if err != nil {
			return err
		}
		prerequisite.Action, err = walletActionPath(app, record)
		if err != nil {
			return err
		}
		resolved = true
	}
	if !resolved {
		return nil
	}
	_, err := ResolvePrerequisites(app, selection, &in.Spec)
	return err
}

// sessionCandidate returns the wallet_actions record id, which this
// session's workflow saved as the candidate key for the selected wallet.
// Another organization's record is answered as a missing one.
func sessionCandidate(
	app core.App,
	selection Selection,
	workflowID, key, id string,
) (*core.Record, error) {
	notSaved := invalidf("prerequisite %s is not saved yet: save it first", key)
	if id == "" {
		return nil, notSaved
	}
	record, err := app.FindRecordById("wallet_actions", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notSaved
	}
	if err != nil {
		return nil, fmt.Errorf("find prerequisite %s: %w", key, err)
	}
	if record.GetString("owner") != selection.Organization.Id {
		return nil, notSaved
	}
	var provenance Provenance
	if record.GetString("wallet") != selection.Wallet.Id ||
		json.Unmarshal([]byte(record.GetString("authoring_provenance")), &provenance) != nil ||
		provenance.WorkflowID != workflowID || provenance.BatchKey != key {
		return nil, fmt.Errorf(
			"%w: prerequisite %s was not saved by this session", ErrForbidden, key,
		)
	}
	return record, nil
}

// checkSessionKeyFree refuses a save under a candidate key that the workflow
// already saved as another record than except.
func checkSessionKeyFree(tx core.App, organizationID, workflowID, key, except string) error {
	record, err := tx.FindFirstRecordByFilter(
		"wallet_actions",
		"owner = {:owner} && id != {:id} && "+
			"authoring_provenance.workflow_id = {:workflow} && "+
			"authoring_provenance.batch_key = {:key}",
		dbx.Params{"owner": organizationID, "id": except, "workflow": workflowID, "key": key},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("find wallet action of %s: %w", key, err)
	}
	if record != nil {
		return fmt.Errorf("%w: action %s of this session is already saved", ErrConflict, key)
	}
	return nil
}

// Save stores the accepted action and its provenance in one transaction.
// Without Replaces it creates the action: a name already in use by the
// wallet is a conflict, except for a retry of the same workflow run that
// wrote it. With Replaces it rewrites the target's name, code, category and
// provenance in place, keeping its id and published flag, when the target
// still has the code the plan was checked against. Either way a workflow
// saves each candidate key as one record only.
// The caller is the trusted workflow's explicit human-confirmation handler,
// after verifying the approval; organization is the session's organization,
// as for Select.
func Save(
	app core.App,
	organization, workflowID, runID string,
	in SaveRequest,
	approval Approval,
) (Receipt, error) {
	var receipt Receipt
	selection, err := Select(app, organization, in.BuildRequest)
	if err != nil {
		return receipt, err
	}
	if err := checkStoreSlot(in.Spec.Slot, selection); err != nil {
		return receipt, err
	}
	role, err := validateAcceptedAction(&in, selection)
	if err != nil {
		return receipt, err
	}
	if in.Spec.Replaces != nil && in.Spec.Replaces.CodeSHA256 == "" {
		return receipt, invalidf("replaces: the plan has no code_sha256; describe the action again")
	}
	provenance := Provenance{
		WalletVersionIdentifier: selection.Identifier,
		Platform:                selection.Platform,
		Source:                  selection.Source,
		PackageID:               in.PackageID,
		YAMLSHA256:              in.CandidateSHA256,
		WorkflowID:              workflowID,
		RunID:                   runID,
		ReplayCount:             in.ReplayCount,
		ApproverID:              approval.ApproverID,
		ApprovedAt:              time.Unix(approval.ApprovedAt, 0).UTC().Format(time.RFC3339),
		BatchKey:                in.BatchKey,
	}
	if selection.Source == SourceStore {
		if in.InstallerSHA256 != "" {
			return receipt, fmt.Errorf("%w: store builds have no installer", ErrInvalid)
		}
		if in.PackageID != selection.PackageID {
			return receipt, fmt.Errorf("%w: wallet app changed", ErrConflict)
		}
		provenance.InstalledVersion = in.InstalledVersion
	} else {
		digest, err := installerDigest(app, selection)
		if err != nil {
			return receipt, err
		}
		if digest != in.InstallerSHA256 {
			return receipt, fmt.Errorf("%w: selected installer changed", ErrConflict)
		}
		provenance.WalletVersionID = selection.Version.Id
		provenance.InstallerSHA256 = digest
	}
	if err := resolveCandidates(app, selection, workflowID, &in); err != nil {
		return receipt, err
	}
	provenanceSpec(&provenance, in.Spec)
	if role != nil {
		if err := fcafProvenance(&provenance, &in, *role); err != nil {
			return receipt, err
		}
	}
	err = app.RunInTransaction(func(tx core.App) error {
		// Recheck authorization and record binding inside the write transaction.
		current, err := Select(tx, organization, in.BuildRequest)
		if err != nil {
			return err
		}
		if selection.Source == SourceStore {
			if current.Source != SourceStore || current.PackageID != selection.PackageID ||
				current.Wallet.Id != selection.Wallet.Id {
				return fmt.Errorf("%w: selected wallet changed", ErrConflict)
			}
		} else if current.Version == nil || current.Version.Id != selection.Version.Id ||
			current.Installer() != selection.Installer() {
			return fmt.Errorf("%w: selected version changed", ErrConflict)
		}
		if in.Spec.Replaces != nil {
			receipt, err = replaceAction(tx, current, workflowID, runID, &in, provenance, role)
			return err
		}
		record, err := tx.FindFirstRecordByFilter(
			"wallet_actions",
			"owner = {:owner} && wallet = {:wallet} && name = {:name}",
			dbx.Params{
				"owner":  current.Organization.Id,
				"wallet": current.Wallet.Id,
				"name":   in.Name,
			},
		)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find wallet action: %w", err)
		}
		if record != nil {
			// A retried save of this workflow run finds its own write and returns it.
			if previous, ok := savedBy(record, workflowID, runID, &in); ok {
				receipt = Receipt{
					RecordID:   record.Id,
					Name:       record.GetString("name"),
					Published:  record.GetBool("published"),
					Provenance: previous,
				}
				return nil
			}
			return fmt.Errorf("%w: an action with this name already exists", ErrConflict)
		}
		err = checkSessionKeyFree(tx, current.Organization.Id, workflowID, in.BatchKey, "")
		if err != nil {
			return err
		}
		collection, err := tx.FindCollectionByNameOrId("wallet_actions")
		if err != nil {
			return fmt.Errorf("find wallet actions collection: %w", err)
		}
		record = core.NewRecord(collection)
		record.Set("owner", current.Organization.Id)
		record.Set("wallet", current.Wallet.Id)
		record.Set("published", false)
		record.Set("name", in.Name)
		record.Set("code", in.Code)
		record.Set("category", SlotCategory(in.Spec.Slot))
		record.Set("authoring_provenance", provenance)
		setFCAFTags(record, role)
		if err := tx.Save(record); err != nil {
			return fmt.Errorf("save authored action: %w", err)
		}
		receipt = Receipt{
			RecordID:   record.Id,
			Name:       record.GetString("name"),
			Created:    true,
			Published:  record.GetBool("published"),
			Provenance: provenance,
		}
		return nil
	})
	return receipt, err
}

// savedBy decodes the record's provenance and reports whether this workflow
// run's save of the request wrote the record as it is.
func savedBy(record *core.Record, workflowID, runID string, in *SaveRequest) (Provenance, bool) {
	var previous Provenance
	ok := json.Unmarshal([]byte(record.GetString("authoring_provenance")), &previous) == nil &&
		previous.WorkflowID == workflowID && previous.RunID == runID &&
		previous.YAMLSHA256 == in.CandidateSHA256 && previous.Platform == in.Platform &&
		previous.InstallerSHA256 == in.InstallerSHA256 &&
		record.GetString("code") == in.Code && previous.BatchKey == in.BatchKey
	return previous, ok
}

// replaceAction rewrites the plan's target in place inside Save's
// transaction: its name, code, category and provenance, keeping its id and
// published flag. A retry of the same workflow run returns its own write.
func replaceAction(
	tx core.App,
	current Selection,
	workflowID, runID string,
	in *SaveRequest,
	provenance Provenance,
	role *walletroles.Role,
) (Receipt, error) {
	record, err := replaceTarget(tx, current, &in.Spec)
	if err != nil {
		return Receipt{}, err
	}
	if previous, ok := savedBy(record, workflowID, runID, in); ok {
		return Receipt{
			RecordID:   record.Id,
			Name:       record.GetString("name"),
			Replaced:   true,
			Published:  record.GetBool("published"),
			Provenance: previous,
		}, nil
	}
	replaced := CodeSHA256(record.GetString("code"))
	if replaced != in.Spec.Replaces.CodeSHA256 {
		return Receipt{}, fmt.Errorf(
			"%w: replaces: %s changed since the plan was checked; describe the action again",
			ErrConflict,
			in.Spec.Replaces.Action,
		)
	}
	other, err := tx.FindFirstRecordByFilter(
		"wallet_actions",
		"owner = {:owner} && wallet = {:wallet} && name = {:name} && id != {:id}",
		dbx.Params{
			"owner":  current.Organization.Id,
			"wallet": current.Wallet.Id,
			"name":   in.Name,
			"id":     record.Id,
		},
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Receipt{}, fmt.Errorf("find wallet action: %w", err)
	}
	if other != nil {
		return Receipt{}, fmt.Errorf("%w: an action with this name already exists", ErrConflict)
	}
	err = checkSessionKeyFree(tx, current.Organization.Id, workflowID, in.BatchKey, record.Id)
	if err != nil {
		return Receipt{}, err
	}
	provenance.ReplacedCodeSHA256 = replaced
	record.Set("name", in.Name)
	record.Set("code", in.Code)
	record.Set("category", SlotCategory(in.Spec.Slot))
	record.Set("authoring_provenance", provenance)
	setFCAFTags(record, role)
	if err := tx.Save(record); err != nil {
		return Receipt{}, fmt.Errorf("replace authored action: %w", err)
	}
	return Receipt{
		RecordID:   record.Id,
		Name:       record.GetString("name"),
		Replaced:   true,
		Published:  record.GetBool("published"),
		Provenance: provenance,
	}, nil
}
