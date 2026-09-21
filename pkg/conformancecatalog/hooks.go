// SPDX-FileCopyrightText: 2026 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package conformancecatalog

import (
	"net/http"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const writeRejectMessage = "conformance_checks is a read-only catalog projection of config_templates"

// Register wires boot rebuild and write-rejection hooks.
//
// Boot: OnBootstrap ensures the collection exists and rebuilds from TemplatesDir().
// Refresh without restart: Rebuild(app, "") or the internal rebuild HTTP route.
func Register(app core.App) {
	app.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		if _, err := EnsureCollection(e.App); err != nil {
			e.App.Logger().Error("conformance catalog: ensure collection failed", "error", err)
			return nil
		}
		if err := Rebuild(e.App, ""); err != nil {
			// Missing templates dir in some test apps is non-fatal; log and continue.
			e.App.Logger().Warn("conformance catalog: boot rebuild failed", "error", err)
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
