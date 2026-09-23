// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// Rebuild walks templatesDir (or TemplatesDir() when empty) and fully replaces
// the process-private :memory: query cache used by the fake PocketBase
// collection URL. That cache is the sole post-rebuild projection.
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

	loaded, err := LoadFromDir(templatesDir)
	if err != nil {
		return err
	}

	if err := replaceEphemeralRows(loaded); err != nil {
		return fmt.Errorf("project ephemeral catalog: %w", err)
	}
	return nil
}
