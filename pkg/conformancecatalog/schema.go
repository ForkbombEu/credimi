// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"strings"
)

// ColumnKind is the FE wire kind for a Client=true column (Zod + PB typegen).
type ColumnKind string

const (
	ColumnKindString      ColumnKind = "string"
	ColumnKindInt         ColumnKind = "int"
	ColumnKindNonNegInt   ColumnKind = "nonNegInt"
	ColumnKindStringArray ColumnKind = "stringArray"
	ColumnKindMemberArray ColumnKind = "memberArray"
)

// ClientColumn is one Client-facing catalog wire field (exported for go generate).
type ClientColumn struct {
	Name     string
	Kind     ColumnKind
	Optional bool
	// Default is a TypeScript literal pasted into z.….default(<Default>) when Optional.
	// Examples: "''", "[]", "9". Empty means no .default().
	Default string
}

// GrainColumn is one persisted catalog column exposed for Go record generate
// (ADR-0009). Includes Client=false timestamps.
type GrainColumn struct {
	Name string
	Kind ColumnKind
}

// columnSpec is one persisted catalog column. This table is the Go source of
// truth for ephemeral DDL, SELECT lists, INSERT column order, Go grain records
// (ADR-0009), and (for Client=true) FE wire emit: columns.ts (incl. pbType),
// Zod, and PB record bodies.
type columnSpec struct {
	Name     string
	SQLType  string // SQLITE column type + constraints (without the name)
	Client   bool   // exposed on FE Zod / synthetic CollectionModel (not created/updated)
	Kind     ColumnKind
	Optional bool
	Default  string
}

// checkColumns defines conformance_checks ephemeral + list/get wire fields.
var checkColumns = []columnSpec{
	{Name: "id", SQLType: "TEXT PRIMARY KEY NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "path", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "title", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "fs_standard", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "fs_version", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "suite", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "file", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "visible_in", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true, Kind: ColumnKindStringArray, Optional: true, Default: "[]"},
	{Name: "protocol", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "sut", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "role", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "provider", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "standard", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "component", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "version", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "created", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false, Kind: ColumnKindString},
	{Name: "updated", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false, Kind: ColumnKindString},
}

// suiteColumns defines conformance_suites ephemeral + list/get wire fields.
var suiteColumns = []columnSpec{
	{Name: "id", SQLType: "TEXT PRIMARY KEY NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "standard", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "component", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "component_rank", SQLType: "INTEGER NOT NULL DEFAULT 9", Client: true, Kind: ColumnKindNonNegInt, Optional: true, Default: "9"},
	{Name: "version", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "provider", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "provider_label", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_name", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_subtitle", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_homepage", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_repository", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_help", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_description", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "suite_logo", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString, Optional: true, Default: "''"},
	{Name: "check_count", SQLType: "INTEGER NOT NULL DEFAULT 0", Client: true, Kind: ColumnKindNonNegInt},
	{Name: "members", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true, Kind: ColumnKindMemberArray, Optional: true, Default: "[]"},
	{Name: "visible_in", SQLType: "TEXT NOT NULL DEFAULT '[]'", Client: true, Kind: ColumnKindStringArray, Optional: true, Default: "[]"},
	{Name: "fs_standard", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString},
	{Name: "fs_version", SQLType: "TEXT NOT NULL DEFAULT ''", Client: true, Kind: ColumnKindString},
	{Name: "path_prefix", SQLType: "TEXT NOT NULL", Client: true, Kind: ColumnKindString},
	{Name: "created", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false, Kind: ColumnKindString},
	{Name: "updated", SQLType: "TEXT NOT NULL DEFAULT ''", Client: false, Kind: ColumnKindString},
}

func columnNames(cols []columnSpec) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.Name
	}
	return out
}

func clientColumns(cols []columnSpec) []ClientColumn {
	out := make([]ClientColumn, 0, len(cols))
	for _, c := range cols {
		if !c.Client {
			continue
		}
		out = append(out, ClientColumn{
			Name:     c.Name,
			Kind:     c.Kind,
			Optional: c.Optional,
			Default:  c.Default,
		})
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

func grainColumns(cols []columnSpec) []GrainColumn {
	out := make([]GrainColumn, len(cols))
	for i, c := range cols {
		out[i] = GrainColumn{Name: c.Name, Kind: c.Kind}
	}
	return out
}

// CheckGrainColumns returns all check-grain columns for Go record generate.
func CheckGrainColumns() []GrainColumn { return grainColumns(checkColumns) }

// SuiteGrainColumns returns all suite-grain columns for Go record generate.
func SuiteGrainColumns() []GrainColumn { return grainColumns(suiteColumns) }

// CheckClientColumnNames is the FE Zod / typegen field set for checks.
func CheckClientColumnNames() []string { return clientColumnNames(checkColumns) }

// SuiteClientColumnNames is the FE Zod / typegen field set for suites.
func SuiteClientColumnNames() []string { return clientColumnNames(suiteColumns) }

// CheckClientColumns returns Client wire specs for checks (go generate).
func CheckClientColumns() []ClientColumn { return clientColumns(checkColumns) }

// SuiteClientColumns returns Client wire specs for suites (go generate).
func SuiteClientColumns() []ClientColumn { return clientColumns(suiteColumns) }

//go:generate go run gen_columns.go
