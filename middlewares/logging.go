// Package middlewares contains HTTP middleware for the e-commerce API.
package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const name = "vivm_dummy_project"

// Context keys for storing request-scoped data.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
)

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

var tracer = otel.Tracer(name)

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

// GetRequestID retrieves the request ID from the context.
// Returns empty string if not found.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// Logging is an HTTP middleware that logs the method, path, response status
// code, and elapsed duration of every request.
// It extracts or generates a request ID and stores it in the context for
// downstream handlers to use in wide events.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(httpResponseWriter http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extract or generate request ID for distributed tracing.
		requestID := r.Header.Get("x-request-id")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store request ID in context for handlers to access.
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		r = r.WithContext(ctx)

		// Add request ID to response headers for client correlation.
		httpResponseWriter.Header().Set("x-request-id", requestID)

		ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		defer span.End()

		r = r.WithContext(ctx)

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

		// Emit a single wide event per request with high cardinality fields.
		wideEvent := map[string]any{
			"timestamp":    time.Now().UTC().Format(time.RFC3339Nano),
			"level":        "info",
			"method":       r.Method,
			"path":         routeName,
			"status_code":  responseWriter.status,
			"duration_ms":  time.Since(start).Milliseconds(),
			"request_id":   requestID,
			"user_agent":   r.UserAgent(),
			"remote_addr":  r.RemoteAddr,
			"service":      name,
		}

		if responseWriter.status >= 400 {
			wideEvent["level"] = "error"
			wideEvent["outcome"] = "error"
		} else {
			wideEvent["outcome"] = "success"
		}

		// This is the canonical log line - one event per request.
		fmt.Printf("[HTTP] %s %s %d %s request_id=%s\n",
			r.Method,
			routeName,
			responseWriter.status,
			time.Since(start).Round(time.Microsecond),
			requestID,
		)
	})
}
