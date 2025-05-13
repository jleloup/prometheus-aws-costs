package otel

import (
	"context"
	"fmt"

	"prometheus-aws-costs/src/utils/config"

	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type TelemetryProvider interface {
	GetServiceName() string
	TraceStart(ctx context.Context, name string) (context.Context, oteltrace.Span)
	Shutdown(ctx context.Context)
}

// Wrapper for OTEL providers
type Telemetry struct {
	tp     *trace.TracerProvider
	cfg    config.Config
	tracer oteltrace.Tracer
}

// Telemetry constructor
func NewTelemetry(ctx context.Context, cfg config.Config) (*Telemetry, error) {
	rp := newResource(cfg.ServiceName, cfg.ServiceVersion)

	tp, err := newTracerProvider(ctx, rp)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracer: %w", err)
	}
	tracer := tp.Tracer(cfg.ServiceName)

	return &Telemetry{
		tp:     tp,
		cfg:    cfg,
		tracer: tracer,
	}, nil
}

// GetServiceName returns the name of the service.
func (t *Telemetry) GetServiceName() string {
	return t.cfg.ServiceName
}

// TraceStart starts a new span with the given name. The span must be ended by calling End.
func (t *Telemetry) TraceStart(ctx context.Context, name string) (context.Context, oteltrace.Span) {
	return t.tracer.Start(ctx, name)
}

// Shutdown shuts down all OTEL providers
func (t *Telemetry) Shutdown(ctx context.Context) {
	t.tp.Shutdown(ctx)
}
