// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"

	validation "github.com/pocketbase/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	walletActionsCollectionName                        = "wallet_actions"
	validationWalletActionMarketLinkRequiresInstallApp = "validation_wallet_action_market_link_requires_install_app"
)

var playStoreDetailsLinkPattern = regexp.MustCompile(
	`(?m)^\s*-\s*openLink:\s*["']market://details\?id=[^"']+["']\s*$`,
)

func RegisterWalletActionHooks(app core.App) {
	app.OnRecordCreate(walletActionsCollectionName).BindFunc(validateWalletActionCategory)
	app.OnRecordUpdate(walletActionsCollectionName).BindFunc(validateWalletActionCategory)
	app.OnRecordUpdate(walletActionsCollectionName).BindFunc(clearStaleAuthoringProvenance)
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

func clearStaleAuthoringProvenance(e *core.RecordEvent) error {
	raw := e.Record.GetString("authoring_provenance")
	if raw != "" {
		var provenance struct {
			YAMLSHA256 string `json:"yaml_sha256"`
		}
		digest := sha256.Sum256([]byte(e.Record.GetString("code")))
		if json.Unmarshal([]byte(raw), &provenance) != nil ||
			provenance.YAMLSHA256 != hex.EncodeToString(digest[:]) {
			e.Record.Set("authoring_provenance", nil)
		}
	}
	return e.Next()
}
