// Package middlewares contains HTTP middleware for the e-commerce API.
package middlewares

import (
	"fmt"
	"net/http"
	"time"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const name = "vivm_dummy_project"

var meter = otel.Meter(name)
var requestCount, _ = meter.Int64Counter("http.request.count",
	metric.WithDescription("Total number of HTTP requests"),
)
var requestDuration, _ = meter.Float64Histogram("http.request.duration",
	metric.WithDescription("HTTP request duration in milliseconds"),
	metric.WithUnit("ms"),
)
var errorCount, _ = meter.Int64Counter("http.error.count",
	metric.WithDescription("Total number of HTTP errors"),
)

// responseWriter is a thin wrapper around http.ResponseWriter that captures
// the HTTP status code written by the downstream handler so the logging
// middleware can record it after the fact.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader intercepts the status code before delegating to the real writer.
func (responseWriter *responseWriter) WriteHeader(code int) {
	responseWriter.status = code
	responseWriter.ResponseWriter.WriteHeader(code)
}


// Logging is an HTTP middleware that logs the method, path, response status
// code, and elapsed duration of every request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(httpResponseWriter http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseWriter := &responseWriter{
			ResponseWriter: httpResponseWriter,
			status:         http.StatusOK,
		}

		next.ServeHTTP(responseWriter, r)

		route := mux.CurrentRoute(r)
		routeName := r.URL.Path
		if route != nil {
			if tpl, err := route.GetPathTemplate(); err == nil {
				routeName = tpl
			}
		}

		attrs := metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", routeName),
			attribute.Int("status code", responseWriter.status),
		)

		requestCount.Add(r.Context(), 1, attrs)
		requestDuration.Record(r.Context(), float64(time.Since(start).Milliseconds()), attrs)
		if responseWriter.status >= 400 {
			errorCount.Add(r.Context(), 1, attrs)
		}

		fmt.Printf(
			"[HTTP] method=%s path=%s status=%d duration=%s\n",
			r.Method,
			routeName,
			responseWriter.status,
			time.Since(start).Round(time.Microsecond),
		)
	})
}
