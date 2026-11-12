// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"encoding/json"
	"regexp"

	"github.com/forkbombeu/credimi/pkg/internal/actionauthor"
	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	walletActionsCollectionName                        = "wallet_actions"
	validationWalletActionMarketLinkRequiresInstallApp = "validation_wallet_action_market_link_requires_install_app"
	// agentAuthoredField tells clients the action author agent validated the stored code,
	// without exposing the hidden provenance.
	agentAuthoredField = "__agent_authored__"
)

var playStoreDetailsLinkPattern = regexp.MustCompile(
	`(?m)^\s*-\s*openLink:\s*["']market://details\?id=[^"']+["']\s*$`,
)

// RegisterWalletActionHooks validates wallet action categories, clears authoring
// provenance that no longer matches the action's code, and marks agent-authored actions.
func RegisterWalletActionHooks(app core.App) {
	app.OnRecordCreate(walletActionsCollectionName).BindFunc(validateWalletActionCategory)
	app.OnRecordUpdate(walletActionsCollectionName).BindFunc(validateWalletActionCategory)
	app.OnRecordUpdate(walletActionsCollectionName).BindFunc(clearStaleAuthoringProvenance)
	app.OnRecordEnrich(walletActionsCollectionName).BindFunc(markAgentAuthored)
}

// markAgentAuthored sets agentAuthoredField when the record keeps provenance of an
// action author session. Edits that change the code clear that provenance, so the
// mark disappears once a human rewrites the action.
func markAgentAuthored(e *core.RecordEnrichEvent) error {
	var provenance actionauthor.Provenance
	authored := json.Unmarshal(
		[]byte(e.Record.GetString("authoring_provenance")),
		&provenance,
	) == nil && provenance.WorkflowID != ""
	e.Record.WithCustomData(true)
	e.Record.Set(agentAuthoredField, authored)
	return e.Next()
}

func validateWalletActionCategory(e *core.RecordEvent) error {
	if !playStoreDetailsLinkPattern.MatchString(e.Record.GetString("code")) ||
		e.Record.GetString("category") == "install-app" {
		return e.Next()
	}

	return apis.NewBadRequestError(
		validationWalletActionMarketLinkRequiresInstallApp,
		validation.Errors{
			"category": validation.NewError(
				validationWalletActionMarketLinkRequiresInstallApp,
				validationWalletActionMarketLinkRequiresInstallApp,
			),
		},
	)
}

// clearStaleAuthoringProvenance drops provenance that no longer describes the stored code.
func clearStaleAuthoringProvenance(e *core.RecordEvent) error {
	if raw := e.Record.GetString("authoring_provenance"); raw != "" {
		var provenance actionauthor.Provenance
		if json.Unmarshal([]byte(raw), &provenance) != nil ||
			provenance.YAMLSHA256 != actionauthor.CodeSHA256(e.Record.GetString("code")) {
			e.Record.Set("authoring_provenance", nil)
		}
	}
	return e.Next()
}
