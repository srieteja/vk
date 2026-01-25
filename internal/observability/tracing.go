package observability

import (
	"context"
	"fmt"

	"vk_backend/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func SetupTracing(ctx context.Context, cfg *config.Config) (func(context.Context) error, bool, error) {
	if cfg.OtelExporterEndpoint == "" {
		return func(context.Context) error { return nil }, false, nil
	}

	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.OtelExporterEndpoint),
	}
	if cfg.OtelExporterInsecure {
		options = append(options, otlptracehttp.WithInsecure())
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.OtelServiceName),
		),
	)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create otel resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return provider.Shutdown, true, nil
}
