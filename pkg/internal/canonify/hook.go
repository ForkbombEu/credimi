// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later
package canonify

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Collections to automatically canonify
var canonifyCollections = map[string]string{
	"users":                   "name",
	"organizations":           "name",
	"credential_issuers":      "name",
	"credentials":             "name",
	"custom_checks":           "name",
	"use_cases_verifications": "name",
	"verifiers":               "name",
	"wallet_actions":          "name",
	"wallets":                 "name",
	"news":                    "title",
}

func MakeExistsFunc(app core.App, collectionName string, canonifiedField string, excludeID string) ExistsFunc {
	return func(value string) bool {
		exp := dbx.HashExp{canonifiedField: value}

		records, _ := app.FindAllRecords(collectionName, exp)
		for _, r := range records {
			if r.Id != excludeID {
				return true
			}
		}
		return false
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

	for col, field := range canonifyCollections {

		canonifiedField := "canonified_name"
		if field == "title" {
			canonifiedField = "canonified_title"
		}

		app.OnRecordCreateRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			e.Record.Set(canonifiedField, "")
			return e.Next()
		})
		app.OnRecordUpdateRequest(col).BindFunc(func(e *core.RecordRequestEvent) error {
			e.Record.Set(canonifiedField, "")
			return e.Next()
		})
		app.OnRecordCreate(col).BindFunc(func(e *core.RecordEvent) error {
			name := e.Record.GetString(field)
			existsFunc := MakeExistsFunc(e.App, col, canonifiedField, "")
			canonName, err := Canonify(name, existsFunc)
			if err != nil {
				return err
			}
			e.Record.Set(canonifiedField, canonName)
			return e.Next()
		})

		app.OnRecordUpdate(col).BindFunc(func(e *core.RecordEvent) error {
			name := e.Record.GetString(field)
			if name == "" {
				return nil
			}

			existsFunc := MakeExistsFunc(e.App, col, canonifiedField, e.Record.Id)
			opts := DefaultOptions
			opts.Fallback = col
			canonName, err := CanonifyWithOptions(name, existsFunc, opts)
			if err != nil {
				return err
			}
			e.Record.Set(canonifiedField, canonName)
			return e.Next()
		})
	}
}
