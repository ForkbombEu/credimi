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
	"sort"
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"gopkg.in/yaml.v3"
)

// ErrForbidden denies unauthorized actors or organization namespaces.
// ErrInvalid rejects malformed authoring input.
// ErrConflict rejects stale approvals and unconfirmed replacements.
var (
	ErrForbidden = errors.New("authoring access denied")
	ErrInvalid   = errors.New("invalid authoring request")
	ErrConflict  = errors.New("authoring approval is stale or conflicts with an existing action")
)

// Actor identifies the authenticated Credimi principal that started the workflow.
type Actor struct {
	ActorID         string `json:"actor_id"`
	ActorCollection string `json:"actor_collection"`
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

// BuildRequest selects an existing wallet version and platform on behalf of its authenticated actor.
type BuildRequest struct {
	Actor
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	Platform                string `json:"platform"`
}

// Build binds an authorized wallet version to its platform installer digest.
type Build struct {
	WalletID          string `json:"wallet_id"`
	VersionID         string `json:"version_id"`
	VersionIdentifier string `json:"version_identifier"`
	OwnerNamespace    string `json:"owner_namespace"`
	Platform          string `json:"platform"`
	InstallerSHA256   string `json:"installer_sha256"`
}

// Selection holds the records reauthorized for a build operation.
type Selection struct {
	Version      *core.Record
	Wallet       *core.Record
	Organization *core.Record
	Actor        Actor
	Identifier   string
	Platform     string
}

// Installer returns the stored filename of the selected platform's installer.
func (s Selection) Installer() string {
	return s.Version.GetString(installerFields[s.Platform].field)
}

// BuildOption describes one build the actor can select for authoring and its installer platforms.
type BuildOption struct {
	Identifier       string   `json:"identifier"`
	WalletName       string   `json:"wallet_name"`
	OrganizationName string   `json:"organization_name"`
	VersionTag       string   `json:"version_tag"`
	Platforms        []string `json:"platforms"`
}

// OwnerFilter restricts a query on a collection with an `owner` organization
// to the organizations where the actor holds the owner role, the role Select
// requires. Superusers are not restricted.
func OwnerFilter(app core.App, in Actor) (dbx.Expression, error) {
	actor, err := findActor(app, in)
	if err != nil {
		return nil, err
	}
	if actor.IsSuperuser() {
		return dbx.NewExp("1=1"), nil
	}
	// Resolve owned organizations first: PocketBase's `=` on a back-relation
	// matches only when every authorization of the organization matches.
	authorizations, err := app.FindRecordsByFilter(
		"orgAuthorizations",
		"user = {:user} && role.name = 'owner'",
		"",
		0,
		0,
		dbx.Params{"user": actor.Id},
	)
	if err != nil {
		return nil, fmt.Errorf("list owned organizations: %w", err)
	}
	owned := make([]any, 0, len(authorizations))
	for _, authorization := range authorizations {
		owned = append(owned, authorization.GetString("organization"))
	}
	return dbx.In("owner", owned...), nil
}

// ListBuildOptions returns only builds authorized by the same owner role as Select.
func ListBuildOptions(app core.App, in Actor) ([]BuildOption, error) {
	ownerFilter, err := OwnerFilter(app, in)
	if err != nil {
		return nil, err
	}
	filter := []dbx.Expression{
		dbx.Or(dbx.NewExp("[[android_installer]] != ''"), dbx.NewExp("[[ios_installer]] != ''")),
		ownerFilter,
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
		})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Identifier < options[j].Identifier
	})
	return options, nil
}

// findActor accepts only user and superuser principals that still exist.
func findActor(app core.App, in Actor) (*core.Record, error) {
	if in.ActorCollection != "users" && in.ActorCollection != core.CollectionNameSuperusers {
		return nil, ErrForbidden
	}
	actor, err := app.FindRecordById(in.ActorCollection, in.ActorID)
	if err != nil {
		return nil, ErrForbidden
	}
	return actor, nil
}

// Select authorizes the same owner role required by wallet_actions creation.
// An empty namespace is used only by the authenticated start endpoint; activities
// must pass their actual Temporal namespace, not a value from their payload.
func Select(app core.App, namespace string, in BuildRequest) (Selection, error) {
	var selection Selection
	actor, err := findActor(app, in.Actor)
	if err != nil {
		return selection, err
	}
	segments := strings.Split(
		strings.Trim(canonify.NormalizePath(in.WalletVersionIdentifier), "/"),
		"/",
	)
	if len(segments) != 3 || segments[0] == "" || segments[1] == "" || segments[2] == "" {
		return selection, fmt.Errorf("%w: expected organization/wallet/version", ErrInvalid)
	}
	version, err := app.FindFirstRecordByFilter(
		"wallet_versions",
		"canonified_tag = {:tag} && wallet.canonified_name = {:wallet} && owner.canonified_name = {:org}",
		dbx.Params{"tag": segments[2], "wallet": segments[1], "org": segments[0]},
	)
	if err != nil {
		return selection, fmt.Errorf("%w: wallet version not found", ErrInvalid)
	}
	wallet, err := app.FindRecordById("wallets", version.GetString("wallet"))
	if err != nil || wallet.GetString("owner") != version.GetString("owner") {
		return selection, fmt.Errorf("%w: invalid wallet ownership", ErrForbidden)
	}
	organization, err := app.FindRecordById("organizations", version.GetString("owner"))
	if err != nil {
		return selection, ErrForbidden
	}
	ownerNamespace := organization.GetString("canonified_name")
	if ownerNamespace == "" || (namespace != "" && namespace != ownerNamespace) {
		return selection, ErrForbidden
	}
	if !actor.IsSuperuser() {
		_, err = app.FindFirstRecordByFilter("orgAuthorizations",
			"user = {:user} && organization = {:org} && role.name = 'owner'",
			dbx.Params{"user": actor.Id, "org": organization.Id})
		if err != nil {
			return selection, ErrForbidden
		}
	}
	installer, ok := installerFields[in.Platform]
	if !ok {
		return selection, fmt.Errorf("%w: platform must be android or ios", ErrInvalid)
	}
	if version.GetString(installer.field) == "" {
		return selection, fmt.Errorf("%w: %s", ErrInvalid, installer.missing)
	}
	selection = Selection{
		Version:      version,
		Wallet:       wallet,
		Organization: organization,
		Actor:        in.Actor,
		Identifier:   strings.Join(segments, "/"),
		Platform:     in.Platform,
	}
	return selection, nil
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
func Resolve(app core.App, namespace string, in BuildRequest) (Build, error) {
	selection, err := Select(app, namespace, in)
	if err != nil {
		return Build{}, err
	}
	digest, err := installerDigest(app, selection)
	if err != nil {
		return Build{}, err
	}
	return Build{
		WalletID:          selection.Wallet.Id,
		VersionID:         selection.Version.Id,
		VersionIdentifier: selection.Identifier,
		OwnerNamespace:    selection.Organization.GetString("canonified_name"),
		Platform:          selection.Platform,
		InstallerSHA256:   digest,
	}, nil
}

// SaveRequest carries the trusted workflow's accepted candidate and replacement precondition.
type SaveRequest struct {
	BuildRequest
	Name                   string `json:"name"`
	Code                   string `json:"code"`
	CandidateSHA256        string `json:"candidate_sha256"`
	InstallerSHA256        string `json:"installer_sha256"`
	PackageID              string `json:"package_id"`
	Slot                   string `json:"slot"`
	ReplaceActionID        string `json:"replace_action_id"`
	ExpectedExistingSHA256 string `json:"expected_existing_sha256"`
	ReplayCount            int    `json:"replay_count"`
}

// Provenance identifies the exact build and workflow that validated an action.
type Provenance struct {
	WalletVersionID         string `json:"wallet_version_id"`
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
	Platform                string `json:"platform"`
	InstallerSHA256         string `json:"installer_sha256"`
	PackageID               string `json:"package_id"`
	YAMLSHA256              string `json:"yaml_sha256"`
	WorkflowID              string `json:"workflow_id"`
	RunID                   string `json:"run_id"`
	ReplayCount             int    `json:"replay_count"`
}

// Receipt describes the saved record and its validation provenance.
type Receipt struct {
	RecordID   string     `json:"record_id"`
	Name       string     `json:"name"`
	Created    bool       `json:"created"`
	Published  bool       `json:"published"`
	Provenance Provenance `json:"provenance"`
}

// CodeSHA256 hashes the exact YAML bytes reviewed by the human.
func CodeSHA256(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

func validateAcceptedAction(in *SaveRequest) error {
	if strings.TrimSpace(in.Name) == "" || in.Slot != "onboarding" || in.ReplayCount != 2 ||
		in.CandidateSHA256 != CodeSHA256(in.Code) || in.PackageID == "" ||
		(in.ReplaceActionID == "") != (in.ExpectedExistingSHA256 == "") {
		return fmt.Errorf("%w: missing accepted candidate or replacement confirmation", ErrInvalid)
	}
	decoder := yaml.NewDecoder(strings.NewReader(in.Code))
	var header struct {
		AppID string `yaml:"appId"`
	}
	if err := decoder.Decode(&header); err != nil || header.AppID != in.PackageID {
		return fmt.Errorf("%w: accepted installer/package mismatch", ErrInvalid)
	}
	return nil
}

// Save performs the compare-and-swap and provenance write in one transaction.
// The caller is the trusted workflow's explicit human-confirmation handler.
func Save(app core.App, namespace, workflowID, runID string, in SaveRequest) (Receipt, error) {
	var receipt Receipt
	if err := validateAcceptedAction(&in); err != nil {
		return receipt, err
	}
	selection, err := Select(app, namespace, in.BuildRequest)
	if err != nil {
		return receipt, err
	}
	digest, err := installerDigest(app, selection)
	if err != nil {
		return receipt, err
	}
	if digest != in.InstallerSHA256 {
		return receipt, fmt.Errorf("%w: selected installer changed", ErrConflict)
	}
	provenance := Provenance{
		WalletVersionID:         selection.Version.Id,
		WalletVersionIdentifier: selection.Identifier,
		Platform:                selection.Platform,
		InstallerSHA256:         digest,
		PackageID:               in.PackageID,
		YAMLSHA256:              in.CandidateSHA256,
		WorkflowID:              workflowID,
		RunID:                   runID,
		ReplayCount:             in.ReplayCount,
	}
	err = app.RunInTransaction(func(tx core.App) error {
		// Recheck authorization and record binding inside the write transaction.
		current, err := Select(tx, namespace, in.BuildRequest)
		if err != nil {
			return err
		}
		installer := selection.Installer()
		if current.Version.Id != selection.Version.Id || current.Installer() != installer {
			return fmt.Errorf("%w: selected version changed", ErrConflict)
		}
		var record *core.Record
		if in.ReplaceActionID != "" {
			record, err = tx.FindRecordById("wallet_actions", in.ReplaceActionID)
		} else {
			record, err = tx.FindFirstRecordByFilter(
				"wallet_actions",
				"owner = {:owner} && wallet = {:wallet} && name = {:name}",
				dbx.Params{
					"owner":  current.Organization.Id,
					"wallet": current.Wallet.Id,
					"name":   in.Name,
				},
			)
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("find wallet action: %w", err)
		}
		created := record == nil
		if created {
			if in.ReplaceActionID != "" {
				return fmt.Errorf("%w: replacement action no longer exists", ErrConflict)
			}
			collection, err := tx.FindCollectionByNameOrId("wallet_actions")
			if err != nil {
				return fmt.Errorf("find wallet actions collection: %w", err)
			}
			record = core.NewRecord(collection)
			record.Set("owner", current.Organization.Id)
			record.Set("wallet", current.Wallet.Id)
			record.Set("published", false)
		} else {
			if record.GetString("owner") != current.Organization.Id ||
				record.GetString("wallet") != current.Wallet.Id {
				return ErrForbidden
			}
			existingSHA256 := CodeSHA256(record.GetString("code"))
			var previous Provenance
			// A retried save of this workflow run finds its own write and returns it.
			if json.Unmarshal([]byte(record.GetString("authoring_provenance")), &previous) == nil &&
				previous.WorkflowID == workflowID && previous.RunID == runID &&
				previous.YAMLSHA256 == in.CandidateSHA256 && previous.Platform == in.Platform &&
				previous.InstallerSHA256 == in.InstallerSHA256 &&
				record.GetString("name") == in.Name && existingSHA256 == in.CandidateSHA256 {
				receipt = Receipt{
					RecordID:   record.Id,
					Name:       record.GetString("name"),
					Published:  record.GetBool("published"),
					Provenance: previous,
				}
				return nil
			}
			if in.ReplaceActionID == "" || existingSHA256 != in.ExpectedExistingSHA256 {
				return fmt.Errorf(
					"%w: action %s has code SHA256 %s; explicit replacement confirmation required",
					ErrConflict,
					record.Id,
					existingSHA256,
				)
			}
		}
		record.Set("name", in.Name)
		record.Set("code", in.Code)
		record.Set("category", "onboarding")
		record.Set("authoring_provenance", provenance)
		if err := tx.Save(record); err != nil {
			return fmt.Errorf("save authored action: %w", err)
		}
		receipt = Receipt{
			RecordID:   record.Id,
			Name:       record.GetString("name"),
			Created:    created,
			Published:  record.GetBool("published"),
			Provenance: provenance,
		}
		return nil
	})
	return receipt, err
}
