// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase/core"
)

// Register wires boot rebuild (ephemeral catalog) and drops any durable
// conformance_checks collection shell left from earlier projections.
//
// List/get are served by Credimi routes that mimic the PocketBase collection
// URL (/api/collections/conformance_checks/records) — see RecordsListHTTP.
//
// Boot: OnBootstrap drops the shell and rebuilds from TemplatesDir().
// A missing templates directory is skipped (test apps / empty checkouts); if the
// directory exists, rebuild failures fail bootstrap. Refresh without restart:
// Rebuild(app, "") or POST /api/conformance-catalog/rebuild.
func Register(app core.App) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if err := dropCollectionShell(e.App); err != nil {
			e.App.Logger().Error("conformance catalog: drop collection shell failed", "error", err)
			return err
		}
		if err := bootRebuild(e.App, TemplatesDir()); err != nil {
			e.App.Logger().Error("conformance catalog: boot failed", "error", err)
			return err
		}
		return nil
	})
}

// bootRebuild rebuilds from templatesDir into the ephemeral store.
// Missing templates dirs are non-fatal; any other rebuild error fails boot.
func bootRebuild(app core.App, templatesDir string) error {
	if _, err := os.Stat(templatesDir); err != nil {
		if os.IsNotExist(err) {
			app.Logger().Warn(
				"conformance catalog: templates dir missing; skipping boot rebuild",
				"dir", templatesDir,
			)
			return nil
		}
		return fmt.Errorf("stat templates dir %q: %w", templatesDir, err)
	}

	if err := Rebuild(app, templatesDir); err != nil {
		return fmt.Errorf("boot rebuild from %q: %w", templatesDir, err)
	}
	return nil
}

// RebuildHTTP returns a handler that rebuilds the catalog (internal admin key).
func RebuildHTTP() func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if err := Rebuild(e.App, ""); err != nil {
			return e.InternalServerError("conformance catalog rebuild failed", err)
		}
		n := len(Default().Snapshot())
		return e.JSON(http.StatusOK, map[string]any{
			"ok":     true,
			"count":  n,
			"source": TemplatesDir(),
		})
	}
}
