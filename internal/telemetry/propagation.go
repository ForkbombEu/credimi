// SPDX-FileCopyrightText: 2025 Forkbomb BV
//
// SPDX-License-Identifier: AGPL-3.0-or-later

package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/propagation"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/workflow"
)

const traceContextHeader = "otel-trace-context"

type traceContextKey struct{}

// TraceContextPropagator propagates OpenTelemetry trace context across Temporal boundaries.
type TraceContextPropagator struct {
	propagator propagation.TextMapPropagator
}

// NewTraceContextPropagator creates a new OpenTelemetry trace context propagator.
func NewTraceContextPropagator() workflow.ContextPropagator {
	return &TraceContextPropagator{propagator: propagation.TraceContext{}}
}

// ContextFromWorkflow extracts a Go context with traceparent from workflow context.
func ContextFromWorkflow(ctx workflow.Context) context.Context {
	carrier, ok := ctx.Value(traceContextKey{}).(map[string]string)
	if !ok || len(carrier) == 0 {
		return context.Background()
	}
	return propagation.TraceContext{}.Extract(context.Background(), propagation.MapCarrier(carrier))
}

func (t *TraceContextPropagator) Inject(ctx context.Context, writer workflow.HeaderWriter) error {
	carrier := propagation.MapCarrier{}
	t.propagator.Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil
	}
	payload, err := converter.GetDefaultDataConverter().ToPayload(map[string]string(carrier))
	if err != nil {
		return err
	}
	writer.Set(traceContextHeader, payload)
	return nil
}

func (t *TraceContextPropagator) Extract(ctx context.Context, reader workflow.HeaderReader) (context.Context, error) {
	payload, ok := reader.Get(traceContextHeader)
	if !ok {
		return ctx, nil
	}
	var carrier map[string]string
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &carrier); err != nil {
		return ctx, err
	}
	return t.propagator.Extract(ctx, propagation.MapCarrier(carrier)), nil
}

func (t *TraceContextPropagator) InjectFromWorkflow(ctx workflow.Context, writer workflow.HeaderWriter) error {
	carrier, ok := ctx.Value(traceContextKey{}).(map[string]string)
	if !ok || len(carrier) == 0 {
		return nil
	}
	payload, err := converter.GetDefaultDataConverter().ToPayload(carrier)
	if err != nil {
		return err
	}
	writer.Set(traceContextHeader, payload)
	return nil
}

func (t *TraceContextPropagator) ExtractToWorkflow(ctx workflow.Context, reader workflow.HeaderReader) (workflow.Context, error) {
	payload, ok := reader.Get(traceContextHeader)
	if !ok {
		return ctx, nil
	}
	var carrier map[string]string
	if err := converter.GetDefaultDataConverter().FromPayload(payload, &carrier); err != nil {
		return ctx, err
	}
	return workflow.WithValue(ctx, traceContextKey{}, carrier), nil
}
