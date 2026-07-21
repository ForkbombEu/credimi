// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pb

import (
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
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
