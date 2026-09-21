// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const pathIndexName = "idx_conformance_checks_path"

// catalogFields is the single source of truth for conformance_checks fields.
// The pb_migrations stub only creates the collection shell; EnsureCollection
// creates missing fields and the path unique index on boot / rebuild.
func catalogFields() []core.Field {
	return []core.Field{
		&core.TextField{Name: "path", Required: true},
		&core.TextField{Name: "title", Required: true},
		&core.TextField{Name: "standard", Required: true},
		&core.TextField{Name: "version", Required: true},
		&core.TextField{Name: "suite", Required: true},
		&core.TextField{Name: "file", Required: true},
		&core.SelectField{
			Name:      "visible_in",
			Values:    []string{SurfaceManual, SurfacePipeline},
			MaxSelect: 2,
		},
		&core.TextField{Name: "protocol"},
		&core.TextField{Name: "sut"},
		&core.TextField{Name: "role"},
		&core.TextField{Name: "provider"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	}
}

// EnsureCollection creates the conformance_checks collection when missing and
// adds any fields / indexes that are not yet present.
// list/view rules are open (empty string) to match public hub catalog listing.
// create/update/delete rules are null so non-superuser writes are rejected; hooks
// also reject writes outside Rebuild projection.
func EnsureCollection(app core.App) (*core.Collection, error) {
	collection, err := app.FindCollectionByNameOrId(CollectionName)
	created := false
	if err != nil || collection == nil {
		collection = core.NewBaseCollection(CollectionName)
		collection.Id = CollectionID
		collection.ListRule = types.Pointer("")
		collection.ViewRule = types.Pointer("")
		collection.CreateRule = nil
		collection.UpdateRule = nil
		collection.DeleteRule = nil
		created = true
	}

	dirty := created
	for _, field := range catalogFields() {
		if collection.Fields.GetByName(field.GetName()) == nil {
			collection.Fields.Add(field)
			dirty = true
		}
	}
	if collection.GetIndex(pathIndexName) == "" {
		collection.AddIndex(pathIndexName, true, "path", "")
		dirty = true
	}

	if dirty {
		if err := app.Save(collection); err != nil {
			return nil, fmt.Errorf("save conformance_checks collection: %w", err)
		}
	}
	return collection, nil
}
