package nirikshaai

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// responseWriter wraps http.ResponseWriter to capture the HTTP status code
// written by the downstream handler.
type responseWriter struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code and delegates to the wrapped writer.
func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware wraps an http.Handler with OpenTelemetry tracing.
// It extracts incoming trace context from request headers, creates a server
// span, and records HTTP method, URL, path, status code, and user-agent
// as span attributes using OpenTelemetry semantic conventions v1.26.0.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/", handler)
//	http.ListenAndServe(":8080", nirikshaai.HTTPMiddleware(mux))
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		spanName := r.Method + " " + r.URL.Path
		ctx, span := otel.Tracer("nirikshaai.http").Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		span.SetAttributes(
			semconv.HTTPRequestMethodOriginal(r.Method),
			semconv.URLFull(r.URL.String()),
			semconv.URLPath(r.URL.Path),
			semconv.UserAgentOriginal(r.Header.Get("User-Agent")),
		)

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r.WithContext(ctx))

		span.SetAttributes(semconv.HTTPResponseStatusCode(rw.status))
		if rw.status >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, http.StatusText(rw.status))
		}
	})
}
