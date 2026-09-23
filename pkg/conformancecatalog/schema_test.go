// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaColumnListsDriveSelectAndDDL(t *testing.T) {
	t.Parallel()

	require.Equal(t, columnNames(checkColumns), catalogSelectColumns)
	require.Equal(t, columnNames(suiteColumns), suiteSelectColumns)

	checkDDL := createTableSQL(CollectionName, checkColumns)
	require.Contains(t, checkDDL, "CREATE TABLE conformance_checks")
	require.Contains(t, checkDDL, "path TEXT NOT NULL")
	require.NotContains(t, checkDDL, "suite_name")

	suiteDDL := createTableSQL(SuitesCollectionName, suiteColumns)
	require.Contains(t, suiteDDL, "CREATE TABLE conformance_suites")
	require.Contains(t, suiteDDL, "suite_name TEXT NOT NULL DEFAULT ''")
	require.Contains(t, suiteDDL, "path_prefix TEXT NOT NULL")

	checkInsert := insertSQL(CollectionName, checkColumns)
	require.Contains(t, checkInsert, "INSERT INTO conformance_checks")
	require.Contains(t, checkInsert, "{:norm_version}")
	require.Contains(t, checkInsert, "{:updated}")

	suiteInsert := insertSQL(SuitesCollectionName, suiteColumns)
	require.Contains(t, suiteInsert, "{:check_paths}")
	require.Contains(t, suiteInsert, "{:fs_standard}")
}

func TestCatalogRowDBTagsMatchCheckColumns(t *testing.T) {
	t.Parallel()
	require.Equal(t, columnNames(checkColumns), dbTagNames(reflect.TypeOf(catalogRow{})))
}

func TestSuiteRowDBTagsMatchSuiteColumns(t *testing.T) {
	t.Parallel()
	require.Equal(t, columnNames(suiteColumns), dbTagNames(reflect.TypeOf(suiteRow{})))
}

func TestClientColumnNamesOmitTimestamps(t *testing.T) {
	t.Parallel()
	checks := CheckClientColumnNames()
	suites := SuiteClientColumnNames()
	require.NotContains(t, checks, "created")
	require.NotContains(t, checks, "updated")
	require.NotContains(t, suites, "created")
	require.NotContains(t, suites, "updated")
	require.Contains(t, checks, "path")
	require.Contains(t, suites, "path_prefix")
}

func dbTagNames(t reflect.Type) []string {
	var out []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		out = append(out, tag)
	}
	return out
}
