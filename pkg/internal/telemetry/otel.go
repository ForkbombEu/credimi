// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package telemetry

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	setupOnce  sync.Once
	setupErr   error
	shutdownFn func(context.Context) error
)

// SetupTracing configures OpenTelemetry tracing for Credimi.
func SetupTracing(ctx context.Context) (func(context.Context) error, error) {
	setupOnce.Do(func() {
		shutdownFn, setupErr = setupTracing(ctx)
	})
	return shutdownFn, setupErr
}

func setupTracing(ctx context.Context) (func(context.Context) error, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	exporter, useBatch, err := newTraceExporter(ctx)
	if err != nil {
		return nil, err
	}

	res := resource.NewWithAttributes("", attribute.String("service.name", "credimi"))
	processor := sdktrace.NewSimpleSpanProcessor(exporter)
	if useBatch {
		processor = sdktrace.NewBatchSpanProcessor(exporter)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(processor),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return tp.Shutdown, nil
}

type stdoutTraceExporter struct {
	writer io.Writer
	mu     sync.Mutex
}

func newStdoutTraceExporter(writer io.Writer) *stdoutTraceExporter {
	return &stdoutTraceExporter{writer: writer}
}

func (e *stdoutTraceExporter) ExportSpans(_ context.Context, spans []sdktrace.ReadOnlySpan) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, span := range spans {
		parent := span.Parent().SpanID().String()
		duration := span.EndTime().Sub(span.StartTime())
		fmt.Fprintf(
			e.writer,
			"trace span name=%s trace_id=%s span_id=%s parent=%s duration=%s\n",
			span.Name(),
			span.SpanContext().TraceID().String(),
			span.SpanContext().SpanID().String(),
			parent,
			duration.Round(time.Millisecond),
		)
		attributes := span.Attributes()
		if len(attributes) == 0 {
			continue
		}
		fmt.Fprint(e.writer, "  attributes:")
		for _, attr := range attributes {
			fmt.Fprintf(e.writer, " %s=%v", attr.Key, attr.Value.AsInterface())
		}
		fmt.Fprintln(e.writer, "")
	}
	return nil
}

func (e *stdoutTraceExporter) Shutdown(_ context.Context) error {
	return nil
}

func newTraceExporter(ctx context.Context) (sdktrace.SpanExporter, bool, error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return newStdoutTraceExporter(os.Stdout), false, nil
	}

	opts := []otlptracehttp.Option{}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		opts = append(opts, otlptracehttp.WithEndpointURL(endpoint))
		if strings.HasPrefix(endpoint, "http://") {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
	} else {
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint), otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, opts...)
	if err != nil {
		return nil, true, err
	}
	return exporter, true, nil
}
