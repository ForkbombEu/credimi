// SPDX-FileCopyrightText: 2025 Your Company
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package walletversions

import (
	"errors"
	"slices"

	"github.com/forkbombeu/credimi/pkg/internal/middlewares"
	"github.com/forkbombeu/credimi/pkg/internal/pbutils"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// installerFields are hidden and protected, so PocketBase rejects filters and
// sorts on them, drops them from responses and non-superuser writes, and
// leaves their downloads to HandleInstallerDownload.
var installerFields = []string{"android_installer", "ios_installer"}

func WalletVersionHooks(app core.App) {
	app.OnRecordEnrich("wallet_versions").BindFunc(HandleWalletVersionEnrich)
	app.OnRecordCreateRequest("wallet_versions").BindFunc(HandleWalletVersionCreate)
	app.OnRecordUpdateRequest("wallet_versions").BindFunc(HandleWalletVersionUpdate)
	app.OnFileDownloadRequest("wallet_versions").BindFunc(HandleInstallerDownload)
}

// HandleWalletVersionEnrich shows the installers of non-downloadable versions
// to superusers and members of the owning organization only.
func HandleWalletVersionEnrich(e *core.RecordEnrichEvent) error {
	var auth *core.Record
	if e.RequestInfo != nil {
		auth = e.RequestInfo.Auth
	}
	if canAccessInstallers(e.App, auth, e.Record) {
		e.Record.Unhide(installerFields...)
	} else {
		e.Record.Hide(installerFields...)
	}
	return e.Next()
}

// HandleWalletVersionCreate keeps the submitted installers of a create the
// collection rule already authorized, and requires at least one of them.
func HandleWalletVersionCreate(e *core.RecordRequestEvent) error {
	if e.HasSuperuserAuth() {
		return e.Next()
	}
	loaded, err := pbutils.LoadHiddenRequestFields(e, installerFields...)
	if err != nil {
		return e.BadRequestError("Failed to read the submitted data.", err)
	}
	if !slices.ContainsFunc(installerFields, func(name string) bool {
		_, ok := loaded[name]
		return ok
	}) {
		return e.BadRequestError(
			"Failed to create record",
			errors.New("android_installer or ios_installer is required"),
		)
	}
	return e.Next()
}

// HandleWalletVersionUpdate keeps the submitted installers of an update the
// collection rule already authorized.
func HandleWalletVersionUpdate(e *core.RecordRequestEvent) error {
	if _, err := pbutils.LoadHiddenRequestFields(e, installerFields...); err != nil {
		return e.BadRequestError("Failed to read the submitted data.", err)
	}
	return e.Next()
}

// HandleInstallerDownload serves the installers of non-downloadable versions
// only to superusers and members of the owning organization, identified by a
// file token, an auth token or a Credimi-Api-Key.
func HandleInstallerDownload(e *core.FileDownloadRequestEvent) error {
	if !slices.Contains(installerFields, e.FileField.Name) || e.Record.GetBool("downloadable") {
		return e.Next()
	}
	if canAccessInstallers(e.App, downloadPrincipal(e.RequestEvent), e.Record) {
		return e.Next()
	}
	return e.NotFoundError("", errors.New("insufficient permissions to access the installer"))
}

func downloadPrincipal(e *core.RequestEvent) *core.Record {
	if token := e.Request.URL.Query().Get("token"); token != "" {
		auth, err := e.App.FindAuthRecordByToken(token, core.TokenTypeFile)
		if err != nil {
			return nil
		}
		return auth
	}
	if e.Auth != nil {
		return e.Auth
	}
	principal, apiErr := middlewares.APIKeyPrincipal(e)
	if apiErr != nil {
		return nil
	}
	return principal
}

func canAccessInstallers(app core.App, auth *core.Record, version *core.Record) bool {
	if version.GetBool("downloadable") {
		return true
	}
	if auth == nil {
		return false
	}
	if auth.IsSuperuser() {
		return true
	}
	if auth.Collection().Name != "users" {
		return false
	}
	_, err := app.FindFirstRecordByFilter(
		"orgAuthorizations",
		"user = {:user} && organization = {:org}",
		dbx.Params{"user": auth.Id, "org": version.GetString("owner")},
	)
	return err == nil
}
