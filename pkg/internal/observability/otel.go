// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package observability exports Credimi traces and metrics over OTLP/HTTP
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
	"os"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

var enabled atomic.Bool

// Enabled reports whether Init configured OTLP export.
func Enabled() bool { return enabled.Load() }

// start installs the global tracer and meter providers and starts Go runtime
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

	tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExp))
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if err := runtime.Start(); err != nil {
		return nil, fmt.Errorf("otel runtime metrics: %w", err)
	}

	enabled.Store(true)
	return func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
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
		se.Router.BindFunc(func(e *core.RequestEvent) error {
			err := e.Next()
			span := trace.SpanFromContext(e.Request.Context())
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
				span.RecordError(err)
			}
			return err
		})
		if err := se.Next(); err != nil {
			return err
		}
		se.Server.Handler = otelhttp.NewHandler(se.Server.Handler, "credimi")
		return nil
	})
}
