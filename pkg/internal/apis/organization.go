// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package apis

import (
	"strings"

	"github.com/forkbombeu/credimi/pkg/internal/canonify"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func HookAtUserCreation(app *pocketbase.PocketBase) {
	app.OnRecordAfterCreateSuccess("users").BindFunc(func(e *core.RecordEvent) error {
		err := createNewOrganizationForUser(e.App, e.Record)
		if err != nil {
			return err
		}
		return e.Next()
	})
}

func HookAtUserLogin(app *pocketbase.PocketBase) {
	app.OnRecordAuthRequest().BindFunc(func(e *core.RecordAuthRequestEvent) error {
		orgAuthCollection, err := e.App.FindCollectionByNameOrId("orgAuthorizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
		}
		user := e.Record
		if isSuperUser(e.App, user) {
			return e.Next()
		}
		_, orgNotFound := e.App.FindFirstRecordByFilter(
			orgAuthCollection.Id,
			"user = {:user}",
			dbx.Params{"user": user.Id},
		)
		if orgNotFound == nil {
			return e.Next()
		}
		err = createNewOrganizationForUser(e.App, user)
		if err != nil {
			return apis.NewInternalServerError("failed to create new organization for user", err)
		}
		return e.Next()
	})
}

func createNewOrganizationForUser(app core.App, user *core.Record) error {
	err := app.RunInTransaction(func(txApp core.App) error {
		orgCollection, err := txApp.FindCollectionByNameOrId("organizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find organizations collection", err)
		}

		newOrg := core.NewRecord(orgCollection)
		emailParts := strings.SplitN(user.Email(), "@", 2)
		if len(emailParts) != 2 {
			return apis.NewInternalServerError("invalid email format", nil)
		}

		orgName := emailParts[0] + "'s organization"
		existsFunc := canonify.MakeExistsFunc(app, "organizations", "canonified_name", "")
		canonName, err := canonify.Canonify(orgName, existsFunc)
		if err != nil {
			return err
		}

		newOrg.Set("name", orgName)
		newOrg.Set("canonified_name", canonName)
		txApp.Save(newOrg)

		ownerRoleRecord, err := txApp.FindFirstRecordByFilter("orgRoles", "name='owner'")
		if err != nil {
			return apis.NewInternalServerError("failed to find owner role", err)
		}

		orgAuthCollection, err := txApp.FindCollectionByNameOrId("orgAuthorizations")
		if err != nil {
			return apis.NewInternalServerError("failed to find orgAuthorizations collection", err)
		}
		newOrgAuth := core.NewRecord(orgAuthCollection)
		newOrgAuth.Set("user", user.Id)
		newOrgAuth.Set("organization", newOrg.Id)
		newOrgAuth.Set("role", ownerRoleRecord.Id)
		txApp.Save(newOrgAuth)

		return nil
	})
	return err
}
