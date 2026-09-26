package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Minimal Prometheus text-format metrics without external dependencies.
//
//   - shule360_http_requests_total{method,route,status} (counter)
//   - shule360_http_request_duration_seconds{method,route} (histogram)
//
// Route PATTERNS are used as the label (e.g. /learners/{id}), never raw
// paths, so label cardinality stays bounded by the route table.

type statusKey struct {
	method string
	route  string
	status string
}

type routeKey struct {
	method string
	route  string
}

type metricsRegistry struct {
	mu       sync.Mutex
	byStatus map[statusKey]uint64
	byRoute  map[routeKey]*routeSeries
}

type routeSeries struct {
	count   uint64
	sum     float64
	buckets []uint64 // cumulative counts per bucket upper bound
}

var buckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

var metrics = &metricsRegistry{
	byStatus: map[statusKey]uint64{},
	byRoute:  map[routeKey]*routeSeries{},
}

func (m *metricsRegistry) record(method, route, status string, seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byStatus[statusKey{method, route, status}]++
	rs, ok := m.byRoute[routeKey{method, route}]
	if !ok {
		rs = &routeSeries{buckets: make([]uint64, len(buckets))}
		m.byRoute[routeKey{method, route}] = rs
	}
	rs.count++
	rs.sum += seconds
	for i, b := range buckets {
		if seconds <= b {
			rs.buckets[i]++
		}
	}
}

// routePattern returns the chi route pattern for the request (bounded
// cardinality), falling back to "unmatched" for requests without a matched
// route.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	if r.URL.Path == "/metrics" || r.URL.Path == "/health" {
		return r.URL.Path
	}
	return "unmatched"
}

// Metrics instruments every request with duration/count metrics and reports
// 5xx responses to Sentry when the SDK is enabled (panics are captured
// separately in RecoverMiddleware).
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		seconds := time.Since(start).Seconds()
		route := routePattern(r)
		metrics.record(r.Method, route, strconv.Itoa(ww.Status()), seconds)

		if ww.Status() >= 500 {
			captureServerEvent(fmt.Sprintf("5xx %s %s -> %d", r.Method, route, ww.Status()), r)
		}
	})
}

// --- Sentry glue (enabled only when SENTRY_DSN is configured) ---
// The middleware package never imports the Sentry SDK directly; main.go
// installs the capture function after a successful sentry.Init.

var sentryCaptureEnabled bool

var captureFn = func(message string, r *http.Request) {}

// EnableSentryCapture turns on 5xx capture (called from main after sentry.Init).
func EnableSentryCapture(fn func(message string, r *http.Request)) {
	captureFn = fn
	sentryCaptureEnabled = true
}

func captureServerEvent(message string, r *http.Request) {
	if sentryCaptureEnabled {
		captureFn(message, r)
	}
}

// MetricsHandler serves the Prometheus text exposition at /metrics.
func MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	metrics.mu.Lock()
	byStatus := make(map[statusKey]uint64, len(metrics.byStatus))
	for k, v := range metrics.byStatus {
		byStatus[k] = v
	}
	byRoute := make(map[routeKey]routeSeries, len(metrics.byRoute))
	for k, v := range metrics.byRoute {
		byRoute[k] = *v
	}
	metrics.mu.Unlock()

	var b []byte
	b = append(b, "# HELP shule360_http_requests_total HTTP requests processed.\n"...)
	b = append(b, "# TYPE shule360_http_requests_total counter\n"...)
	for k, count := range byStatus {
		b = append(b, fmt.Sprintf("shule360_http_requests_total{method=%q,route=%q,status=%q} %d\n",
			k.method, k.route, k.status, count)...)
	}

	b = append(b, "# HELP shule360_http_request_duration_seconds Request duration distribution.\n"...)
	b = append(b, "# TYPE shule360_http_request_duration_seconds histogram\n"...)
	for k, rs := range byRoute {
		label := fmt.Sprintf("method=%q,route=%q", k.method, k.route)
		for i, bound := range buckets {
			b = append(b, fmt.Sprintf("shule360_http_request_duration_seconds_bucket{%s,le=%q} %d\n",
				label, strconv.FormatFloat(bound, 'g', -1, 64), rs.buckets[i])...)
		}
		b = append(b, fmt.Sprintf("shule360_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", label, rs.count)...)
		b = append(b, fmt.Sprintf("shule360_http_request_duration_seconds_sum{%s} %f\n", label, rs.sum)...)
		b = append(b, fmt.Sprintf("shule360_http_request_duration_seconds_count{%s} %d\n", label, rs.count)...)
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write(b)
}
