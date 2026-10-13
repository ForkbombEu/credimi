// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"path/filepath"
	"strings"

	"github.com/pocketbase/dbx"
)

// SuitePathPrefix builds the durable suite path_prefix (fs_standard/fs_version/suite).
func SuitePathPrefix(fsStandard, fsVersion, suite string) string {
	return suitePathPrefix(fsStandard, fsVersion, suite)
}

// SuitePathPrefixFromCheckID derives path_prefix from a check path
// (fs_standard/fs_version/suite/…). Returns "" when the id has fewer than 3 segments.
func SuitePathPrefixFromCheckID(checkID string) string {
	parts := strings.Split(filepath.ToSlash(strings.Trim(checkID, "/")), "/")
	if len(parts) < 3 {
		return ""
	}
	return suitePathPrefix(parts[0], parts[1], parts[2])
}

// SuiteHasQR reports whether the suite grain at pathPrefix has authored has_qr.
// Absent suite, catalog unavailable, or false ⇒ false.
func SuiteHasQR(pathPrefix string) bool {
	pathPrefix = strings.TrimSpace(pathPrefix)
	if pathPrefix == "" {
		return false
	}
	db, err := catalogDB()
	if err != nil {
		return false
	}
	var hasQR bool
	err = db.Select("has_qr").
		From(SuitesCollectionName).
		AndWhere(dbx.HashExp{"path_prefix": pathPrefix}).
		Row(&hasQR)
	if err != nil {
		return false
	}
	return hasQR
}
