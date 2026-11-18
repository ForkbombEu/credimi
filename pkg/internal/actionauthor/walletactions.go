// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"encoding/json"
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// WalletAction is one action a session can run as a prerequisite. StartsFrom
// lists the actions it was saved to start from, in order, and
// ParameterSources the record sources its parameters were saved with. Owned
// actions belong to the caller's organization: only they can be replaced.
type WalletAction struct {
	Path             string            `json:"path"`
	Name             string            `json:"name"`
	Category         string            `json:"category"`
	Published        bool              `json:"published"`
	Owned            bool              `json:"owned"`
	Parameters       []string          `json:"parameters"`
	StartsFrom       []string          `json:"starts_from"`
	ParameterSources []ParameterSource `json:"parameter_sources"`
}

// ParameterSource is the credential offer or use-case verification record a
// wallet action's parameter was saved with.
type ParameterSource struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
}

// savedProvenance decodes a wallet action's authoring provenance. It is the
// zero Provenance for an action without one or whose provenance can't be decoded.
func savedProvenance(record *core.Record) Provenance {
	var provenance Provenance
	raw := record.GetString("authoring_provenance")
	if raw == "" || json.Unmarshal([]byte(raw), &provenance) != nil {
		return Provenance{}
	}
	return provenance
}

// startsFrom returns the action paths a wallet action was saved to start
// from, in order. It is empty, never nil.
func startsFrom(provenance Provenance) []string {
	paths := make([]string, 0, len(provenance.Prerequisites))
	for _, path := range provenance.Prerequisites {
		paths = append(paths, canonify.NormalizePath(path))
	}
	return paths
}

// parameterSources returns the provenance parameters that have a source, in
// order. It is empty, never nil.
func parameterSources(provenance Provenance) []ParameterSource {
	sources := make([]ParameterSource, 0, len(provenance.Parameters))
	for _, parameter := range provenance.Parameters {
		if parameter.Source != "" {
			sources = append(sources, ParameterSource(parameter))
		}
	}
	return sources
}

// ListWalletActions returns the selected wallet's actions that the caller's
// organization (selection.Organization, not the wallet's owner) owns or that
// are published, sorted by category then name. A non-empty category keeps
// only that category.
func ListWalletActions(app core.App, selection Selection, category string) ([]WalletAction, error) {
	filter := "wallet = {:wallet} && (owner = {:org} || published = true)"
	params := dbx.Params{"wallet": selection.Wallet.Id, "org": selection.Organization.Id}
	if category != "" {
		filter += " && category = {:category}"
		params["category"] = category
	}
	records, err := app.FindRecordsByFilter("wallet_actions", filter, "category,name", 0, 0, params)
	if err != nil {
		return nil, fmt.Errorf("list wallet actions: %w", err)
	}
	actions := make([]WalletAction, 0, len(records))
	for _, record := range records {
		path, err := canonify.BuildPath(
			app,
			record,
			canonify.CanonifyPaths["wallet_actions"],
			record.GetString("canonified_name"),
		)
		if err != nil {
			return nil, fmt.Errorf("build wallet action path: %w", err)
		}
		provenance := savedProvenance(record)
		actions = append(actions, WalletAction{
			Path:             path,
			Name:             record.GetString("name"),
			Category:         record.GetString("category"),
			Published:        record.GetBool("published"),
			Owned:            record.GetString("owner") == selection.Organization.Id,
			Parameters:       ActionParameters(record.GetString("code")),
			StartsFrom:       startsFrom(provenance),
			ParameterSources: parameterSources(provenance),
		})
	}
	return actions, nil
}
