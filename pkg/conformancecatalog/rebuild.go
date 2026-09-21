// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Rebuild walks templatesDir (or TemplatesDir() when empty), replaces the
// in-memory snapshot, and fully replaces conformance_checks collection rows.
//
// Refresh path for local template edits: call Rebuild, or POST
// /api/conformance-catalog/rebuild with X-Api-Key = CREDIMI_INTERNAL_ADMIN_KEY,
// or restart the process (Register hooks rebuild on bootstrap).
func Rebuild(app core.App, templatesDir string) error {
	if templatesDir == "" {
		templatesDir = TemplatesDir()
	}

	checks, blueprints, err := loadFromDir(templatesDir)
	if err != nil {
		return err
	}

	if _, err := EnsureCollection(app); err != nil {
		return err
	}

	collection, err := app.FindCollectionByNameOrId(CollectionName)
	if err != nil {
		return fmt.Errorf("find conformance_checks: %w", err)
	}

	projecting.Store(true)
	defer projecting.Store(false)

	err = app.RunInTransaction(func(txApp core.App) error {
		if _, err := txApp.NonconcurrentDB().Delete(collection.Name, dbx.NewExp("1 = 1")).Execute(); err != nil {
			return fmt.Errorf("clear conformance_checks: %w", err)
		}

		for _, ch := range checks {
			record := core.NewRecord(collection)
			record.Set("id", ch.ID)
			record.Set("path", ch.Path)
			record.Set("title", ch.Title)
			record.Set("standard", ch.Standard)
			record.Set("version", ch.Version)
			record.Set("suite", ch.Suite)
			record.Set("file", ch.File)
			record.Set("visible_in", ch.VisibleIn)
			record.Set("protocol", ch.Protocol)
			record.Set("sut", ch.SUT)
			record.Set("role", ch.Role)
			record.Set("provider", ch.Provider)
			if err := txApp.SaveNoValidate(record); err != nil {
				return fmt.Errorf("save check %s: %w", ch.Path, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	Default().replaceSnapshot(checks, blueprints, templatesDir)
	return nil
}
