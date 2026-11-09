// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package temporalclient

import (
	"sync"

	"github.com/forkbombeu/credimi/pkg/internal/observability"
	"github.com/forkbombeu/credimi/pkg/internal/temporalauth"
	"github.com/forkbombeu/credimi/pkg/utils"
	"go.temporal.io/sdk/client"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	tlog "go.temporal.io/sdk/log"
)

var (
	loggerMu sync.RWMutex
	logger   tlog.Logger
)

// ConfigureLogger sets the logger used by Temporal clients created after the call.
func ConfigureLogger(l tlog.Logger) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	logger = l
}

// baseOptions returns the connection options shared by every Temporal client.
// Every call carries Credimi's system token (temporalauth.SystemToken).
// A nil logger is left unset so the SDK keeps its default. With OpenTelemetry
// export on, clients and the workers built on them propagate trace context
// through workflow and activity headers and report SDK metrics.
func baseOptions(namespace string) client.Options {
	options := client.Options{
		HostPort:  utils.GetEnvironmentVariable("TEMPORAL_ADDRESS", client.DefaultHostPort),
		Namespace: namespace,
		// Credimi reaches Temporal on its private network (temporal:7233 in
		// compose, localhost in dev). API-key credentials would otherwise
		// switch TLS on.
		Credentials:       client.NewAPIKeyDynamicCredentials(temporalauth.SystemToken),
		ConnectionOptions: client.ConnectionOptions{TLSDisabled: true},
	}
	if observability.Enabled() {
		// Update spans are off: the SDK starts update handler spans during
		// replay too, so every query on an uncached semaphore re-emits a span
		// per update in its history (thousands a minute). Enqueue and cancel
		// stay visible as their HTTP request spans.
		if tracing, err := temporalotel.NewTracingInterceptor(
			temporalotel.TracerOptions{DisableUpdateTracing: true},
		); err == nil {
			options.Interceptors = []interceptor.ClientInterceptor{tracing}
		}
		options.MetricsHandler = temporalotel.NewMetricsHandler(
			temporalotel.MetricsHandlerOptions{},
		)
	}
	loggerMu.RLock()
	defer loggerMu.RUnlock()
	if logger != nil {
		options.Logger = logger
	}
	return options
}
