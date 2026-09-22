// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// Rebuild walks templatesDir (or TemplatesDir() when empty), replaces the
// in-memory snapshot, and fully replaces the process-private :memory: query
// cache used by the fake PocketBase collection URL.
//
// Refresh path for local template edits: call Rebuild, or POST
// /api/conformance-catalog/rebuild with X-Api-Key = CREDIMI_INTERNAL_ADMIN_KEY,
// or restart the process (Register hooks rebuild on bootstrap).
//
// The app argument is retained for call-site compatibility; rebuild does not
// write to PocketBase data.db.
func Rebuild(app core.App, templatesDir string) error {
	_ = app
	if templatesDir == "" {
		templatesDir = TemplatesDir()
	}

	checks, err := LoadFromDir(templatesDir)
	if err != nil {
		return err
	}

	if err := replaceEphemeralRows(checks); err != nil {
		return fmt.Errorf("project ephemeral catalog: %w", err)
	}

	Default().replaceSnapshot(checks)
	return nil
}
