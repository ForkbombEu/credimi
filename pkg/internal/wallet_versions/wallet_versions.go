// SPDX-FileCopyrightText: 2025 Your Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package walletversions

import (
	"github.com/pocketbase/pocketbase/core"
)

func WalletVersionHooks(app core.App) {
	app.OnRecordEnrich("wallet_versions").BindFunc(HandleWalletVersionEnrich)
}

func HandleWalletVersionEnrich(e *core.RecordEnrichEvent) error {
	if !e.Record.GetBool("downloadable") {
		e.Record.Hide("android_installer")
	}
	return e.Next()
}
