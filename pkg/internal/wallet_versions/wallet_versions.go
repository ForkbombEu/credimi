// SPDX-FileCopyrightText: 2025 Your Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package walletversions

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func WalletVersionHooks(app core.App) {
	app.OnRecordEnrich("wallet_versions").BindFunc(HandleWalletVersionEnrich)
}

// HandleWalletVersionEnrich hides the installers of non-downloadable versions
// from everyone except members of the owning organization.
func HandleWalletVersionEnrich(e *core.RecordEnrichEvent) error {
	if !e.Record.GetBool("downloadable") && !isOwnerOrganizationMember(e) {
		e.Record.Hide("android_installer", "ios_installer")
	}
	return e.Next()
}

func isOwnerOrganizationMember(e *core.RecordEnrichEvent) bool {
	if e.RequestInfo == nil || e.RequestInfo.Auth == nil {
		return false
	}
	auth := e.RequestInfo.Auth
	if auth.Collection().Name != "users" {
		return false
	}
	_, err := e.App.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user} && organization = {:org}",
		dbx.Params{"user": auth.Id, "org": e.Record.GetString("owner")},
	)
	return err == nil
}
