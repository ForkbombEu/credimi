// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"log"

	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	tlog "go.temporal.io/sdk/log"
)

// TemporalClientSetupHook validates the Temporal secrets encryption key and
// routes Temporal SDK logs through the PocketBase logger when the server starts.
// It must be bound before any other hook that creates Temporal clients on serve.
func TemporalClientSetupHook(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := temporalcrypto.ValidateEnv(); err != nil {
			log.Fatalf("invalid Temporal secrets encryption key: %v", err)
		}
		temporalclient.ConfigureLogger(
			tlog.NewStructuredLogger(se.App.Logger().With("component", "temporal")),
		)
		return se.Next()
	})
}
