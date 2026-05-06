package telemetry

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func TestNewTracer(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	if tracer.provider == nil {
		t.Error("Tracer provider is nil")
	}
	if tracer.tracer == nil {
		t.Error("Tracer instance is nil")
	}
}

func TestStartSpan(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.StartSpan(ctx, "test.operation",
		attribute.String("test_key", "test_value"),
	)
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if span == nil {
		t.Error("Span is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceSubsystemInit(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceSubsystemInit(ctx, "networking")
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceGossipPublish(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceGossipPublish(ctx, "/murmur/waves/1", 2048)
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceGossipReceive(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceGossipReceive(ctx, "/murmur/waves/1", "deadbeef")
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceShroudCircuit(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceShroudCircuit(ctx, "circuit-123", 3)
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceWaveCreation(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceWaveCreation(ctx, "surface", 20)
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceLayoutIteration(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceLayoutIteration(ctx, 500)
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceResonanceComputation(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceResonanceComputation(ctx, "surface", "wave_published")
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestTraceEventBusFanout(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	spanCtx, span := tracer.TraceEventBusFanout(ctx, "wave_received", 12)
	defer span.End()

	if spanCtx == nil {
		t.Error("Span context is nil")
	}
	if !span.SpanContext().IsValid() {
		t.Error("Span context is not valid")
	}
}

func TestShutdown(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := tracer.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown() failed: %v", err)
	}
}

func TestNestedSpans(t *testing.T) {
	ctx := context.Background()
	tracer, err := NewTracer(ctx)
	if err != nil {
		t.Fatalf("NewTracer() failed: %v", err)
	}
	defer tracer.Shutdown(ctx)

	// Create parent span
	parentCtx, parentSpan := tracer.StartSpan(ctx, "parent.operation")
	defer parentSpan.End()

	// Create child span using parent context
	childCtx, childSpan := tracer.StartSpan(parentCtx, "child.operation")
	defer childSpan.End()

	if !parentSpan.SpanContext().IsValid() {
		t.Error("Parent span context is not valid")
	}
	if !childSpan.SpanContext().IsValid() {
		t.Error("Child span context is not valid")
	}
	if childCtx == nil {
		t.Error("Child span context is nil")
	}
}
