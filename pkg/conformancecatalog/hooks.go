// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"fmt"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const writeRejectMessage = "conformance_checks is a read-only catalog projection of config_templates"

// Register wires boot rebuild and write-rejection hooks.
//
// Boot: OnBootstrap ensures the collection exists and rebuilds from TemplatesDir().
// A missing templates directory is skipped (test apps / empty checkouts); if the
// directory exists, ensure/rebuild failures fail bootstrap so the process does
// not serve stale durable rows. Refresh without restart: Rebuild(app, "") or the
// internal rebuild HTTP route.
func Register(app core.App) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if err := bootRebuild(e.App, TemplatesDir()); err != nil {
			e.App.Logger().Error("conformance catalog: boot failed", "error", err)
			return err
		}
		return nil
	})

	reject := func(e *core.RecordEvent) error {
		if e.Record == nil || e.Record.Collection().Name != CollectionName {
			return e.Next()
		}
		if IsProjecting() {
			return e.Next()
		}
		return apis.NewBadRequestError(writeRejectMessage, nil)
	}

	app.OnRecordCreateRequest(CollectionName).BindFunc(func(e *core.RecordRequestEvent) error {
		if IsProjecting() {
			return e.Next()
		}
		return e.BadRequestError(writeRejectMessage, nil)
	})
	app.OnRecordUpdateRequest(CollectionName).BindFunc(func(e *core.RecordRequestEvent) error {
		if IsProjecting() {
			return e.Next()
		}
		return e.BadRequestError(writeRejectMessage, nil)
	})
	app.OnRecordDeleteRequest(CollectionName).BindFunc(func(e *core.RecordRequestEvent) error {
		if IsProjecting() {
			return e.Next()
		}
		return e.BadRequestError(writeRejectMessage, nil)
	})

	app.OnRecordCreate(CollectionName).BindFunc(reject)
	app.OnRecordUpdate(CollectionName).BindFunc(reject)
	app.OnRecordDelete(CollectionName).BindFunc(reject)
}

// bootRebuild ensures the collection and rebuilds from templatesDir.
// Missing templates dirs are non-fatal; any other ensure/rebuild error fails boot.
func bootRebuild(app core.App, templatesDir string) error {
	if _, err := EnsureCollection(app); err != nil {
		return fmt.Errorf("ensure collection: %w", err)
	}

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
