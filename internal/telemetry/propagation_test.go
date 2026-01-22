// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package telemetry

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

type headerCarrier struct {
	fields map[string]*commonpb.Payload
}

func newHeaderCarrier() *headerCarrier {
	return &headerCarrier{fields: make(map[string]*commonpb.Payload)}
}

type baseWorkflowContext struct{}

func (baseWorkflowContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (baseWorkflowContext) Done() workflow.Channel {
	return nil
}

func (baseWorkflowContext) Err() error {
	return nil
}

func (baseWorkflowContext) Value(key interface{}) interface{} {
	return nil
}

func (h *headerCarrier) Set(key string, value *commonpb.Payload) {
	h.fields[key] = value
}

func (h *headerCarrier) Get(key string) (*commonpb.Payload, bool) {
	payload, ok := h.fields[key]
	return payload, ok
}

func (h *headerCarrier) ForEachKey(handler func(string, *commonpb.Payload) error) error {
	for key, value := range h.fields {
		if err := handler(key, value); err != nil {
			return err
		}
	}
	return nil
}

func TestTraceContextPropagatorInjectExtract(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	otel.SetTracerProvider(provider)
	defer func() {
		_ = provider.Shutdown(context.Background())
	}()

	ctx, span := otel.Tracer("test").Start(context.Background(), "root")
	defer span.End()

	propagator := NewTraceContextPropagator()
	header := newHeaderCarrier()

	if err := propagator.Inject(ctx, header); err != nil {
		t.Fatalf("inject failed: %v", err)
	}

	outCtx, err := propagator.Extract(context.Background(), header)
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}

	parent := trace.SpanContextFromContext(ctx)
	extracted := trace.SpanContextFromContext(outCtx)
	if parent.TraceID() != extracted.TraceID() {
		t.Fatalf("expected trace ID %s, got %s", parent.TraceID().String(), extracted.TraceID().String())
	}
}

func TestTraceContextPropagatorWorkflowRoundTrip(t *testing.T) {
	propagator := NewTraceContextPropagator()
	header := newHeaderCarrier()

	carrier := map[string]string{
		"traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	}
	payload, err := converter.GetDefaultDataConverter().ToPayload(carrier)
	if err != nil {
		t.Fatalf("payload encode failed: %v", err)
	}
	header.Set(traceContextHeader, payload)

	wfCtx := workflow.WithValue(baseWorkflowContext{}, traceContextKey{}, map[string]string{})
	wfCtx, err = propagator.ExtractToWorkflow(wfCtx, header)
	if err != nil {
		t.Fatalf("extract to workflow failed: %v", err)
	}

	outHeader := newHeaderCarrier()
	if err := propagator.InjectFromWorkflow(wfCtx, outHeader); err != nil {
		t.Fatalf("inject from workflow failed: %v", err)
	}

	outPayload, ok := outHeader.Get(traceContextHeader)
	if !ok {
		t.Fatal("expected trace context payload")
	}

	var roundTrip map[string]string
	if err := converter.GetDefaultDataConverter().FromPayload(outPayload, &roundTrip); err != nil {
		t.Fatalf("payload decode failed: %v", err)
	}

	if roundTrip["traceparent"] != carrier["traceparent"] {
		t.Fatalf("expected traceparent %s, got %s", carrier["traceparent"], roundTrip["traceparent"])
	}
}
