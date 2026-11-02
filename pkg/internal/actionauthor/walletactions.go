// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package actionauthor

import (
	"fmt"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// WalletAction is one action a session can run as a prerequisite.
type WalletAction struct {
	Path       string   `json:"path"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Published  bool     `json:"published"`
	Parameters []string `json:"parameters"`
}

// ListWalletActions returns the selected wallet's actions that its
// organization owns or that are published, sorted by category then name. A
// non-empty category keeps only that category.
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
		actions = append(actions, WalletAction{
			Path:       path,
			Name:       record.GetString("name"),
			Category:   record.GetString("category"),
			Published:  record.GetBool("published"),
			Parameters: ActionParameters(record.GetString("code")),
		})
	}
	return actions, nil
}
