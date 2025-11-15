// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package canonify

import (
	"database/sql"
	"errors"
	"log"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func MakeExistsFunc(
	app core.App,
	collectionName string,
	rec *core.Record,
	excludeID string,
) func(candidateName string) bool {
	return func(candidateName string) bool {
		tpl, ok := CanonifyPaths[collectionName]
		if !ok {
			return true
		}
		path, err := BuildPath(app, rec, tpl, candidateName)
		if err != nil {
			log.Printf("failed to build path: %s", err)
			// if we cannot build path, assume it exists to prevent collision
			return true
		}

		existingRec, err := Resolve(app, path)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				log.Printf("failed to resolve path: %s", err)
			}
			return !errors.Is(err, sql.ErrNoRows)
		}
		if excludeID != "" && existingRec.Id == excludeID {
			return false
		}
		return true
	}
}

// RegisterCanonifyHooks registers hooks for canonifying names in specified collections.
//
// For each collection, it registers two hooks: one for after creating a record, and one for after updating a record.
// Both hooks canonify the name of the record using the provided function,
// and save the updated record to persist the canonified field.
// The canonified field is named "canonified_name" or "canonified_title" depending on the source field name.
// The existsFunc parameter is used to check if a canonized name already exists in the collection.
// If the canonized name already exists, the function returns an error.
// If the source field is empty, the hooks skip the record and do not perform any action.
//
// The function takes a PocketBase application as a parameter, and registers the hooks for the specified collections.
func RegisterCanonifyHooks(app core.App) {

	app.OnRecordEnrich("marketplace_items").BindFunc(func(e *core.RecordEnrichEvent) error {
		colType := e.Record.GetString("type")
		colType = strings.Trim(colType, `"`)
		tpl := CanonifyPaths[colType]
		col, err := e.App.FindCollectionByNameOrId(colType)
		if err != nil {
			return err
		}
		rec, err := e.App.FindRecordById(col.Name, e.Record.Id)
		if err != nil {
			return err
		}
		path, err := BuildPath(e.App, rec, tpl, "")
		if err != nil {
			return err
		}
		e.Record.WithCustomData(true)
		e.Record.Set("__canonified_path__", path)
		return e.Next()
	})
	for col, tpl := range CanonifyPaths {
		app.OnRecordCreateRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			e.Record.Set(tpl.CanonifiedField, "")
			return e.Next()
		})
		app.OnRecordUpdateRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			e.Record.Set(tpl.CanonifiedField, "")
			return e.Next()
		})
		app.OnRecordCreate(col).BindFunc(func(e *core.RecordEvent) error {
			name := e.Record.GetString(tpl.Field)
			existsFunc := MakeExistsFunc(e.App, col, e.Record, "")
			canonName, err := Canonify(name, existsFunc)
			if err != nil {
				return err
			}
			e.Record.Set(tpl.CanonifiedField, canonName)
			return e.Next()
		})

		app.OnRecordUpdate(col).BindFunc(func(e *core.RecordEvent) error {
			name := e.Record.GetString(tpl.Field)
			if name == "" {
				return e.Next()
			}

			existsFunc := MakeExistsFunc(e.App, col, e.Record, e.Record.Id)
			opts := DefaultOptions
			opts.Fallback = col
			canonName, err := CanonifyWithOptions(name, existsFunc, opts)
			if err != nil {
				return err
			}
			e.Record.Set(tpl.CanonifiedField, canonName)
			return e.Next()
		})

		app.OnRecordEnrich(col).BindFunc(func(e *core.RecordEnrichEvent) error {
			path, err := BuildPath(e.App, e.Record, tpl, "")
			if err != nil {
				return err
			}
			e.Record.WithCustomData(true)
			e.Record.Set("__canonified_path__", path)
			return e.Next()
		})
	}
}
