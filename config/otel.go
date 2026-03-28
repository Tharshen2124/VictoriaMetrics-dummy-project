package config

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
)

func SetupOTelSDK(ctx context.Context) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}

		shutdownFuncs = nil

		return err
	}

	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	meterProvider, err := newMeterProvider(ctx)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	loggerProvider, err := newLoggerProvider(ctx)
	if err != nil {
		handleErr(err)
		return shutdown, err
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	
	global.SetLoggerProvider(loggerProvider)

	return shutdown, err
}

func newMeterProvider(ctx context.Context) (*metric.MeterProvider, error) {
	metricExporter, err := otlpmetrichttp.New(
		ctx, 
		otlpmetrichttp.WithEndpoint("localhost:4318"),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithInterval(3*time.Second),
		)),
	)

	return meterProvider, nil
}

func newLoggerProvider(ctx context.Context) (*log.LoggerProvider, error) {
	logExporter, err := otlploghttp.New(
		ctx,
		otlploghttp.WithEndpoint("localhost:4318"),
		otlploghttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter),
	))

	return loggerProvider, nil
}