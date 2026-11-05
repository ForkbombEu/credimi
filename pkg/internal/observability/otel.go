// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package observability exports Credimi traces, metrics and logs over OTLP/HTTP
// (SigNoz). Export is on only when OTEL_EXPORTER_OTLP_ENDPOINT is set; the
// exporters read the standard OTEL_* variables themselves (endpoint, headers,
// timeout, compression), and the resource honours OTEL_SERVICE_NAME and
// OTEL_RESOURCE_ATTRIBUTES.
package observability

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/forkbombeu/credimi/pkg/internal/apierror"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

const scope = "github.com/forkbombeu/credimi"

var enabled atomic.Bool

// Enabled reports whether Setup turned OTLP export on.
func Enabled() bool { return enabled.Load() }

// Handler returns next, also sending every record to OpenTelemetry when export
// is on. Records logged with a context carry its trace and span IDs.
func Handler(next slog.Handler) slog.Handler {
	if !Enabled() {
		return next
	}
	return slog.NewMultiHandler(next, otelslog.NewHandler(scope))
}

// start installs the global tracer, meter and logger providers, sends slog and
// the standard log package to OTel as well as stderr, and starts Go runtime
// metrics. The returned function flushes and stops the exporters.
func start(ctx context.Context) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName("credimi")),
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithProcessRuntimeVersion(),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	traceExp, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}
	metricExp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("otlp metric exporter: %w", err)
	}
	logExp, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("otlp log exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExp))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)
	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetLoggerProvider(lp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if err := runtime.Start(); err != nil {
		return nil, fmt.Errorf("otel runtime metrics: %w", err)
	}

	enabled.Store(true)
	// The text handler, not the previous default: SetDefault redirects the log
	// package into slog, and the old default handler writes back to it.
	slog.SetDefault(slog.New(Handler(slog.NewTextHandler(os.Stderr, nil))))
	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx), lp.Shutdown(ctx))
	}, nil
}

// Setup turns OTLP export on when OTEL_EXPORTER_OTLP_ENDPOINT is set. The
// exporters are flushed on terminate. Every served request is traced: the
// otelhttp wrapper goes around the final mux, so it also covers the UI and
// Temporal UI proxies, and the router middleware names each span after the
// matched route pattern and records the authenticated user and handler errors.
func Setup(app core.App) {
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") == "" {
		return
	}
	shutdown, err := start(context.Background())
	if err != nil {
		log.Printf("OpenTelemetry export disabled: %v", err)
		return
	}
	app.OnTerminate().BindFunc(func(e *core.TerminateEvent) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(ctx); err != nil {
			log.Printf("OpenTelemetry shutdown: %v", err)
		}
		return e.Next()
	})
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST(BrowserTracesPath, browserTracesHandler())
		se.Router.BindFunc(func(e *core.RequestEvent) (err error) {
			span := trace.SpanFromContext(e.Request.Context())
			defer func() {
				if r := recover(); r != nil {
					// PocketBase recovers the panic into a 500; record it first,
					// while the stack still holds the panicking frames.
					if err, ok := r.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
						span.RecordError(fmt.Errorf("panic: %v", r), trace.WithStackTrace(true))
						span.SetStatus(codes.Error, "panic")
					}
					panic(r)
				}
			}()
			err = e.Next()
			if p := e.Request.Pattern; p != "" {
				span.SetName(p)
				span.SetAttributes(semconv.HTTPRoute(p))
			}
			if e.Auth != nil {
				span.SetAttributes(
					attribute.String("enduser.id", e.Auth.Id),
					attribute.String("enduser.collection", e.Auth.Collection().Name),
				)
			}
			if err != nil {
				RecordServerError(e.Request.Context(), err, errorStatus(err))
			}
			return err
		})
		if err := se.Next(); err != nil {
			return err
		}
		// Browser exports would otherwise add a span every few seconds per tab.
		se.Server.Handler = otelhttp.NewHandler(
			se.Server.Handler,
			"credimi",
			otelhttp.WithFilter(
				func(r *http.Request) bool { return r.URL.Path != BrowserTracesPath },
			),
		)
		return nil
	})
}

// RecordServerError records err as an exception on the request span when the
// response is a server error. Client errors (4xx) are expected and stay as the
// span's status code only.
func RecordServerError(ctx context.Context, err error, status int) {
	if err == nil || status < http.StatusInternalServerError {
		return
	}
	span := trace.SpanFromContext(ctx)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// errorStatus is the status PocketBase answers err with.
func errorStatus(err error) int {
	var apiError *apierror.APIError
	if errors.As(err, &apiError) {
		return apiError.Code
	}
	var pbError *router.ApiError
	if errors.As(err, &pbError) {
		return pbError.Status
	}
	return http.StatusInternalServerError
}
