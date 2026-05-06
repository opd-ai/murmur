// Package telemetry provides OpenTelemetry tracing integration for MURMUR subsystem interactions.
// Per ROADMAP.md v1.0 milestone, operators need distributed tracing visibility to debug
// performance bottlenecks and understand request flows across subsystems.
package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	serviceName    = "murmur"
	serviceVersion = "0.1.0"
)

// Tracer provides tracing functionality for MURMUR subsystems.
type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

// NewTracer initializes OpenTelemetry tracing with stdout exporter.
// In production, configure OTLP exporter to send traces to Jaeger/Tempo.
func NewTracer(ctx context.Context) (*Tracer, error) {
	exporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(provider)

	tracer := provider.Tracer(serviceName)

	return &Tracer{
		provider: provider,
		tracer:   tracer,
	}, nil
}

// Shutdown flushes remaining traces and stops the provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}

// StartSpan creates a new span for tracing subsystem interactions.
// Returns the span and a context with the span embedded.
func (t *Tracer) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// TraceSubsystemInit traces subsystem initialization with timing.
func (t *Tracer) TraceSubsystemInit(ctx context.Context, subsystem string) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "subsystem.init",
		attribute.String("subsystem", subsystem),
	)
}

// TraceGossipPublish traces GossipSub message publication.
func (t *Tracer) TraceGossipPublish(ctx context.Context, topic string, size int) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "gossip.publish",
		attribute.String("topic", topic),
		attribute.Int("size_bytes", size),
	)
}

// TraceGossipReceive traces GossipSub message reception and validation.
func (t *Tracer) TraceGossipReceive(ctx context.Context, topic, messageID string) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "gossip.receive",
		attribute.String("topic", topic),
		attribute.String("message_id", messageID),
	)
}

// TraceShroudCircuit traces Shroud circuit construction.
func (t *Tracer) TraceShroudCircuit(ctx context.Context, circuitID string, hops int) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "shroud.circuit_construct",
		attribute.String("circuit_id", circuitID),
		attribute.Int("hops", hops),
	)
}

// TraceWaveCreation traces Wave creation with PoW computation.
func (t *Tracer) TraceWaveCreation(ctx context.Context, waveType string, difficulty int) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "wave.create",
		attribute.String("wave_type", waveType),
		attribute.Int("difficulty", difficulty),
	)
}

// TraceLayoutIteration traces force-directed layout computation.
func (t *Tracer) TraceLayoutIteration(ctx context.Context, nodeCount int) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "layout.iteration",
		attribute.Int("node_count", nodeCount),
	)
}

// TraceResonanceComputation traces Resonance score update.
func (t *Tracer) TraceResonanceComputation(ctx context.Context, layer, interactionType string) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "resonance.compute",
		attribute.String("layer", layer),
		attribute.String("interaction_type", interactionType),
	)
}

// TraceEventBusFanout traces event bus message dispatch.
func (t *Tracer) TraceEventBusFanout(ctx context.Context, eventType string, subscriberCount int) (context.Context, trace.Span) {
	return t.StartSpan(ctx, "eventbus.fanout",
		attribute.String("event_type", eventType),
		attribute.Int("subscriber_count", subscriberCount),
	)
}
