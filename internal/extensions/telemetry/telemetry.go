//go:build !no_telemetry

package telemetry

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/zap"

	sentryotel "github.com/getsentry/sentry-go/otel"
)

var dsn = "https://9caaa13919c5dcca72dccb2e14cab9d6@o4506736716021760.ingest.us.sentry.io/4507174897516544"

func SetupTelemetry(name string, version string) func() {
	finalTags := map[string]string{
		"name":    name,
		"version": version,
	}
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Debug:            false,
		EnableTracing:    true,
		TracesSampleRate: 1.0,
		Release:          fmt.Sprint(name, "@", version),
		Tags:             finalTags,
	}); err != nil {
		zap.L().Error("sentry.Init failed", zap.Error(err))
		return func() {}
	}
	defer sentry.CaptureEvent(&sentry.Event{
		Message: "dgate started",
		Level:   sentry.LevelInfo,
	})
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(
			sentryotel.NewSentrySpanProcessor(),
		),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(sentryotel.NewSentryPropagator())
	return func() {
		sentry.Flush(2 * time.Second)
	}
}

func CaptureError(err error) {
	sentry.CaptureException(err)
}