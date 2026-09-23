// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"strings"
)

// columnSpec is one persisted catalog column. This table is the Go source of
// truth for ephemeral DDL, SELECT lists, and INSERT column order. Client-facing
// Zod / typegen stubs must stay aligned with Client=true columns (see schema_test
// and webapp/src/lib/conformance/columns.ts).
type columnSpec struct {
	Name    string
	SQLType string // SQLITE column type + constraints (without the name)
	Client  bool   // exposed on FE Zod / synthetic CollectionModel (not created/updated)
}

// checkColumns defines conformance_checks ephemeral + list/get wire fields.
var checkColumns = []columnSpec{
	{Name: "id", SQLType: "TEXT PRIMARY KEY NOT NULL", Client: true},
	{Name: "path", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "title", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "standard", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "version", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "suite", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "file", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "visible_in", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true},
	{Name: "protocol", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "sut", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "role", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "provider", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "norm_standard", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "component", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "norm_version", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "created", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false},
	{Name: "updated", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false},
}

// suiteColumns defines conformance_suites ephemeral + list/get wire fields.
var suiteColumns = []columnSpec{
	{Name: "id", SQLType: "TEXT PRIMARY KEY NOT NULL", Client: true},
	{Name: "standard", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "component", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "component_rank", SQLType: "INTEGER NOT NULL DEFAULT 9", Client: true},
	{Name: "version", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "provider", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite_name", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite_homepage", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite_repository", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite_help", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite_description", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "suite_logo", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "check_count", SQLType: "INTEGER NOT NULL DEFAULT 0", Client: true},
	{Name: "check_paths", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true},
	{Name: "check_titles", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true},
	{Name: "check_files", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true},
	{Name: "visible_in", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true},
	{Name: "fs_standard", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "fs_version", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true},
	{Name: "path_prefix", SQLType: "TEXT NOT NULL", Client: true},
	{Name: "created", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false},
	{Name: "updated", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false},
}

func columnNames(cols []columnSpec) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Name
	}
	return out
}

func clientColumnNames(cols []columnSpec) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		if c.Client {
			out = append(out, c.Name)
		}
	}
	return out
}

func createTableSQL(table string, cols []columnSpec) string {
	lines := make([]string, 0, len(cols))
	for _, c := range cols {
		lines = append(lines, fmt.Sprintf("\t%s %s", c.Name, c.SQLType))
	}
	return fmt.Sprintf("CREATE TABLE %s (\n%s\n)", table, strings.Join(lines, ",\n"))
}

func insertSQL(table string, cols []columnSpec) string {
	names := columnNames(cols)
	placeholders := make([]string, len(names))
	for i, n := range names {
		placeholders[i] = "{:" + n + "}"
	}
	return fmt.Sprintf(
		"INSERT INTO %s (\n\t%s\n) VALUES (\n\t%s\n)",
		table,
		strings.Join(names, ", "),
		strings.Join(placeholders, ", "),
	)
}

// catalogSelectColumns / suiteSelectColumns are derived once for list/get + search.
var (
	catalogSelectColumns = columnNames(checkColumns)
	suiteSelectColumns   = columnNames(suiteColumns)
	catalogSearchFields  = catalogSelectColumns
	suiteSearchFields    = suiteSelectColumns
)

// CheckClientColumnNames is the FE Zod / typegen field set for checks.
func CheckClientColumnNames() []string { return clientColumnNames(checkColumns) }

// SuiteClientColumnNames is the FE Zod / typegen field set for suites.
func SuiteClientColumnNames() []string { return clientColumnNames(suiteColumns) }

//go:generate go run gen_columns.go
