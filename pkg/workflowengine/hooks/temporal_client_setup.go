// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package hooks

import (
	"log"
	"log/slog"

	"github.com/forkbombeu/credimi/pkg/internal/observability"
	"github.com/forkbombeu/credimi/pkg/internal/temporalauth"
	"github.com/forkbombeu/credimi/pkg/internal/temporalclient"
	"github.com/forkbombeu/credimi/pkg/internal/temporalcrypto"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	tlog "go.temporal.io/sdk/log"
)

// TemporalClientSetupHook validates the Temporal secrets encryption key and
// the Temporal JWT signing keys, writes the signing keys' public JWKS file
// that Temporal reads (temporalauth.JWKSDir), and routes Temporal SDK logs
// through the PocketBase logger, and to OpenTelemetry when export is on, when
// the server starts.
// It must be bound before any other hook that creates Temporal clients on serve.
func TemporalClientSetupHook(app *pocketbase.PocketBase) {
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		if err := temporalcrypto.ValidateEnv(); err != nil {
			log.Fatalf("invalid Temporal secrets encryption key: %v", err)
		}
		signer, err := temporalauth.DefaultSigner()
		if err != nil {
			log.Fatalf("invalid Temporal JWT signing keys: %v", err)
		}
		if err := temporalauth.WriteJWKS(temporalauth.JWKSDir(), signer); err != nil {
			log.Fatalf("write Temporal JWKS: %v", err)
		}
		logger := slog.New(observability.Handler(se.App.Logger().Handler()))
		temporalclient.ConfigureLogger(
			tlog.NewStructuredLogger(logger.With("component", "temporal")),
		)
		return se.Next()
	})
}
