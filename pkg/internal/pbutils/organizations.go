// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pbutils

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// FindOrganizationByNamespace returns the organization whose canonified name is
// namespace, ignoring surrounding spaces. A missing or blank namespace returns
// an error wrapping sql.ErrNoRows.
func FindOrganizationByNamespace(app core.App, namespace string) (*core.Record, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("organization namespace is required: %w", sql.ErrNoRows)
	}
	org, err := app.FindFirstRecordByFilter(
		"organizations",
		"canonified_name = {:namespace}",
		dbx.Params{"namespace": namespace},
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("organization %s not found: %w", namespace, err)
		}
		return nil, fmt.Errorf("find organization %s: %w", namespace, err)
	}
	return org, nil
}

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
	return GetOrganizationCanonifiedName(app, orgID)
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
