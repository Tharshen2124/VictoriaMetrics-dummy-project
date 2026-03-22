// Package middlewares contains HTTP middleware for the e-commerce API.
package middlewares

import (
	"fmt"
	"net/http"
	"time"
)

// responseWriter is a thin wrapper around http.ResponseWriter that captures
// the HTTP status code written by the downstream handler so the logging
// middleware can record it after the fact.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader intercepts the status code before delegating to the real writer.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logging is an HTTP middleware that logs the method, path, response status
// code, and elapsed duration of every request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK, // default if WriteHeader is never called
		}

		next.ServeHTTP(rw, r)

		fmt.Printf(
			"[HTTP] method=%s path=%s status=%d duration=%s\n",
			r.Method,
			r.URL.Path,
			rw.status,
			time.Since(start).Round(time.Microsecond),
		)
	})
}
