// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package pbutils

import (
	"database/sql"
	"errors"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type OrganizationPublicationCollection struct {
	Collection  string
	OwnerField  string
	PublicField string
}

var organizationPublicationCollections = []OrganizationPublicationCollection{
	{Collection: "wallets", OwnerField: "owner", PublicField: "published"},
	{Collection: "credential_issuers", OwnerField: "owner", PublicField: "published"},
	{Collection: "verifiers", OwnerField: "owner", PublicField: "published"},
	{Collection: "custom_checks", OwnerField: "owner", PublicField: "published"},
	{Collection: "pipelines", OwnerField: "owner", PublicField: "published"},
}

// OrganizationPublicationCollections returns the entity collections that can
// make an organization publicly visible on the hub.
func OrganizationPublicationCollections() []OrganizationPublicationCollection {
	out := make([]OrganizationPublicationCollection, len(organizationPublicationCollections))
	copy(out, organizationPublicationCollections)
	return out
}

// OrganizationHasPublicEntities reports whether the organization owns at least one
// published record in any of the hub-visible entity collections.
func OrganizationHasPublicEntities(app core.App, ownerID string) (bool, error) {
	for _, collection := range organizationPublicationCollections {
		_, err := app.FindFirstRecordByFilter(
			collection.Collection,
			collection.OwnerField+"={:owner} && "+collection.PublicField+"=true",
			dbx.Params{"owner": ownerID},
		)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}

		return false, err
	}

	return false, nil
}

// OrganizationPublicationCollectionByName resolves a PocketBase collection to the
// publication metadata used by organization sync hooks.
func OrganizationPublicationCollectionByName(
	collection *core.Collection,
) *OrganizationPublicationCollection {
	if collection == nil {
		return nil
	}

	for i := range organizationPublicationCollections {
		if organizationPublicationCollections[i].Collection == collection.Name {
			return &organizationPublicationCollections[i]
		}
	}

	return nil
}
