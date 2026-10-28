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

// BuildRequest selects an existing wallet version on behalf of its authenticated actor.
type BuildRequest struct {
	Actor
	WalletVersionIdentifier string `json:"wallet_version_identifier"`
}

// Build binds an authorized wallet version to its APK digest and download capability.
type Build struct {
	WalletID          string `json:"wallet_id"`
	VersionID         string `json:"version_id"`
	VersionIdentifier string `json:"version_identifier"`
	OwnerNamespace    string `json:"owner_namespace"`
	InstallerName     string `json:"installer_name"`
	APKSHA256         string `json:"apk_sha256"`
	DownloadGrant     string `json:"download_grant"`
}

// Selection holds the records reauthorized for a build operation.
type Selection struct {
	Version      *core.Record
	Wallet       *core.Record
	Organization *core.Record
	Actor        Actor
	Identifier   string
}

// BuildOption describes one Android build the actor can select for authoring.
type BuildOption struct {
	Identifier       string `json:"identifier"`
	WalletName       string `json:"wallet_name"`
	OrganizationName string `json:"organization_name"`
	VersionTag       string `json:"version_tag"`
}

// ListBuildOptions returns only builds authorized by the same owner role as Select.
func ListBuildOptions(app core.App, in Actor) ([]BuildOption, error) {
	if in.ActorCollection != "users" && in.ActorCollection != core.CollectionNameSuperusers {
		return nil, ErrForbidden
	}
	actor, err := app.FindRecordById(in.ActorCollection, in.ActorID)
	if err != nil {
		return nil, ErrForbidden
	}
	filter := "android_installer != ''"
	params := dbx.Params{}
	if !actor.IsSuperuser() {
		filter += " && owner.orgAuthorizations_via_organization.user = {:user}" +
			" && owner.orgAuthorizations_via_organization.role.name = 'owner'"
		params["user"] = actor.Id
	}
	versions, err := app.FindRecordsByFilter("wallet_versions", filter, "created", 0, 0, params)
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
		options = append(options, BuildOption{
			Identifier: strings.Join([]string{
				organization.GetString("canonified_name"),
				wallet.GetString("canonified_name"),
				version.GetString("canonified_tag"),
			}, "/"),
			WalletName:       wallet.GetString("name"),
			OrganizationName: organization.GetString("name"),
			VersionTag:       version.GetString("tag"),
		})
	}
	sort.Slice(options, func(i, j int) bool {
		return options[i].Identifier < options[j].Identifier
	})
	return options, nil
}

// Select authorizes the same owner role required by wallet_actions creation.
// An empty namespace is used only by the authenticated start endpoint; activities
// must pass their actual Temporal namespace, not a value from their payload.
func Select(app core.App, namespace string, in BuildRequest) (Selection, error) {
	var selection Selection
	if in.ActorCollection != "users" && in.ActorCollection != core.CollectionNameSuperusers {
		return selection, ErrForbidden
	}
	actor, err := app.FindRecordById(in.ActorCollection, in.ActorID)
	if err != nil {
		return selection, ErrForbidden
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
	if version.GetString("android_installer") == "" {
		return selection, fmt.Errorf("%w: selected version has no Android APK", ErrInvalid)
	}
	selection = Selection{
		Version:      version,
		Wallet:       wallet,
		Organization: organization,
		Actor:        in.Actor,
		Identifier:   strings.Join(segments, "/"),
	}
	return selection, nil
}

// APKDigest streams the selected object rather than retaining the installer in memory.
func APKDigest(app core.App, selection Selection) (string, error) {
	fsys, err := app.NewFilesystem()
	if err != nil {
		return "", fmt.Errorf("open installer filesystem: %w", err)
	}
	defer fsys.Close()
	file, err := fsys.GetReader(
		selection.Version.BaseFilesPath() + "/" + selection.Version.GetString("android_installer"),
	)
	if err != nil {
		return "", fmt.Errorf("open selected APK: %w", err)
	}
	defer file.Close()
	digest := sha256.New()
	if _, err = io.Copy(digest, file); err != nil {
		return "", fmt.Errorf("hash selected APK: %w", err)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// Resolve hashes the authorized APK and issues a short-lived, build-bound download grant.
func Resolve(app core.App, namespace string, in BuildRequest) (Build, error) {
	selection, err := Select(app, namespace, in)
	if err != nil {
		return Build{}, err
	}
	digest, err := APKDigest(app, selection)
	if err != nil {
		return Build{}, err
	}
	grant, err := IssueGrant(app, selection, digest)
	if err != nil {
		return Build{}, err
	}
	return Build{
		WalletID:          selection.Wallet.Id,
		VersionID:         selection.Version.Id,
		VersionIdentifier: selection.Identifier,
		OwnerNamespace:    selection.Organization.GetString("canonified_name"),
		InstallerName: selection.Version.GetString(
			"android_installer",
		),
		APKSHA256:     digest,
		DownloadGrant: grant,
	}, nil
}

// SaveRequest carries the trusted workflow's accepted candidate and replacement precondition.
type SaveRequest struct {
	BuildRequest
	Name                   string `json:"name"`
	Code                   string `json:"code"`
	CandidateSHA256        string `json:"candidate_sha256"`
	APKSHA256              string `json:"apk_sha256"`
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
	APKSHA256               string `json:"apk_sha256"`
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
		return fmt.Errorf("%w: accepted APK/package mismatch", ErrInvalid)
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
	digest, err := APKDigest(app, selection)
	if err != nil {
		return receipt, err
	}
	if digest != in.APKSHA256 {
		return receipt, fmt.Errorf("%w: selected APK changed", ErrConflict)
	}
	provenance := Provenance{WalletVersionID: selection.Version.Id,
		WalletVersionIdentifier: selection.Identifier, APKSHA256: digest, PackageID: in.PackageID,
		YAMLSHA256: in.CandidateSHA256, WorkflowID: workflowID, RunID: runID, ReplayCount: 2}
	err = app.RunInTransaction(func(tx core.App) error {
		// Recheck authorization and record binding inside the write transaction.
		current, selectErr := Select(tx, namespace, in.BuildRequest)
		if selectErr != nil {
			return selectErr
		}
		if current.Version.Id != selection.Version.Id ||
			current.Version.GetString(
				"android_installer",
			) != selection.Version.GetString(
				"android_installer",
			) {
			return fmt.Errorf("%w: selected version changed", ErrConflict)
		}
		var record *core.Record
		created := false
		if in.ReplaceActionID != "" {
			record, selectErr = tx.FindRecordById("wallet_actions", in.ReplaceActionID)
		} else {
			record, selectErr = tx.FindFirstRecordByFilter(
				"wallet_actions",
				"owner = {:owner} && wallet = {:wallet} && name = {:name}",
				dbx.Params{
					"owner":  current.Organization.Id,
					"wallet": current.Wallet.Id,
					"name":   in.Name,
				},
			)
		}
		if selectErr != nil && !errors.Is(selectErr, sql.ErrNoRows) {
			return fmt.Errorf("find wallet action: %w", selectErr)
		}
		if record != nil {
			if record.GetString("owner") != current.Organization.Id ||
				record.GetString("wallet") != current.Wallet.Id {
				return ErrForbidden
			}
			var previous Provenance
			if err := json.Unmarshal(
				[]byte(record.GetString("authoring_provenance")),
				&previous,
			); err == nil &&
				previous.WorkflowID == workflowID && previous.RunID == runID &&
				previous.YAMLSHA256 == in.CandidateSHA256 && previous.APKSHA256 == in.APKSHA256 &&
				record.GetString("name") == in.Name &&
				CodeSHA256(record.GetString("code")) == in.CandidateSHA256 {
				receipt = Receipt{
					RecordID:   record.Id,
					Name:       record.GetString("name"),
					Published:  record.GetBool("published"),
					Provenance: previous,
				}
				return nil
			}
			if in.ReplaceActionID == "" ||
				CodeSHA256(record.GetString("code")) != in.ExpectedExistingSHA256 {
				return fmt.Errorf(
					"%w: action %s has code SHA256 %s; explicit replacement confirmation required",
					ErrConflict,
					record.Id,
					CodeSHA256(record.GetString("code")),
				)
			}
		} else {
			if in.ReplaceActionID != "" {
				return fmt.Errorf("%w: replacement action no longer exists", ErrConflict)
			}
			collection, collectionErr := tx.FindCollectionByNameOrId("wallet_actions")
			if collectionErr != nil {
				return collectionErr
			}
			record = core.NewRecord(collection)
			record.Set("owner", current.Organization.Id)
			record.Set("wallet", current.Wallet.Id)
			record.Set("published", false)
			created = true
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
