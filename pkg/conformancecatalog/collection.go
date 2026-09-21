// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// EnsureCollection creates the conformance_checks collection when missing.
// list/view rules are open (empty string) to match public /api/template/blueprints.
// create/update/delete rules are null so non-superuser writes are rejected; hooks
// also reject writes outside Rebuild projection.
func EnsureCollection(app core.App) (*core.Collection, error) {
	existing, err := app.FindCollectionByNameOrId(CollectionName)
	if err == nil && existing != nil {
		return existing, nil
	}

	collection := core.NewBaseCollection(CollectionName)
	collection.Id = CollectionID
	collection.ListRule = types.Pointer("")
	collection.ViewRule = types.Pointer("")
	collection.CreateRule = nil
	collection.UpdateRule = nil
	collection.DeleteRule = nil

	collection.Fields.Add(
		&core.TextField{
			Name:     "path",
			Required: true,
		},
		&core.TextField{
			Name:     "title",
			Required: true,
		},
		&core.TextField{
			Name:     "standard",
			Required: true,
		},
		&core.TextField{
			Name:     "version",
			Required: true,
		},
		&core.TextField{
			Name:     "suite",
			Required: true,
		},
		&core.TextField{
			Name:     "file",
			Required: true,
		},
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
	)

	collection.AddIndex("idx_conformance_checks_path", true, "path", "")

	if err := app.Save(collection); err != nil {
		return nil, fmt.Errorf("save conformance_checks collection: %w", err)
	}
	return collection, nil
}
