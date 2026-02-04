// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package handlers

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

func GetUserOrganization(app core.App, userID string) (*core.Record, error) {
	orgID, err := GetUserOrganizationID(app, userID)
	if err != nil {
		return nil, err
	}
	orgRecord, err := app.FindFirstRecordByFilter(
		"organizations",
		"id={:id}",
		dbx.Params{"id": orgID},
	)
	if err != nil {
		return nil, err
	}
	return orgRecord, nil
}

func GetUserOrganizationID(app core.App, userID string) (string, error) {
	orgAuthCollection, err := app.FindCollectionByNameOrId("orgAuthorizations")
	if err != nil {
		return "", err
	}

	authOrgRecords, err := app.FindFirstRecordByFilter(
		orgAuthCollection.Id,
		"user={:user}",
		dbx.Params{"user": userID},
	)
	if err != nil {
		return "", err
	}
	return authOrgRecords.GetString("organization"), nil
}

func GetUserOrganizationCanonifiedName(app core.App, userID string) (string, error) {
	orgID, err := GetUserOrganizationID(app, userID)
	if err != nil {
		return "", err
	}
	orgRecord, err := app.FindFirstRecordByFilter(
		"organizations",
		"id={:id}",
		dbx.Params{"id": orgID},
	)
	if err != nil {
		return "", err
	}
	return orgRecord.GetString("canonified_name"), nil
}

func GetOrganizationCanonifiedName(app core.App, orgID string) (string, error) {
	orgRecord, err := app.FindFirstRecordByFilter(
		"organizations",
		"id={:id}",
		dbx.Params{"id": orgID},
	)
	if err != nil {
		return "", err
	}
	return orgRecord.GetString("canonified_name"), nil
}
