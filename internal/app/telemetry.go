package app

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// InitTelemetry initializes the OpenTelemetry provider and returns a shutdown function.
func (app *Application) InitTelemetry() (func(context.Context), error) {
	if app.config.OtelCollectorUrl == "" {
		app.logger.Info("OpenTelemetry collector URL not set, skipping initialization")

		return func(context.Context) {}, nil
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("movie-reservation-api"),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(app.config.Env),
		),
	)
	if err != nil {
		return nil, errors.New("failed to create otel resource")
	}

	// Create the trace provider
	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(app.config.OtelCollectorUrl),
	)
	if err != nil {
		return nil, errors.New("failed to create otel trace exporter")
	}

	bsp := trace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
		trace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// Create the meter provider
	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithEndpoint(app.config.OtelCollectorUrl),
	)
	if err != nil {
		return nil, errors.New("failed to create otel metric exporter")
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(metricExporter, metric.WithInterval(15*time.Second))),
	)

	otel.SetMeterProvider(meterProvider)

	shutdown := func(ctx context.Context) {
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		err := errors.Join(
			tracerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
		)
		if err != nil {
			app.logger.Error("failed to shutdown telemetry providers", "error", err)
		}
	}

	return shutdown, nil
}
