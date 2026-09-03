package observability

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/observability/timing"
	"github.com/zatrano/framework/packages/routing"
)

// Metrics collects simple request timing statistics.
type Metrics struct {
	mu            sync.RWMutex
	requests      atomic.Int64
	errors        atomic.Int64
	totalDuration atomic.Int64 // nanoseconds
	slowestPath   string
	slowestDur    time.Duration
	byStatus      map[int]int64
}

// New creates a metrics collector.
func New() *Metrics {
	return &Metrics{
		byStatus: make(map[int]int64),
	}
}

// Observe records a request sample.
func (m *Metrics) Observe(method, path string, status int, duration time.Duration) {
	m.requests.Add(1)
	m.totalDuration.Add(duration.Nanoseconds())
	if status >= 400 {
		m.errors.Add(1)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byStatus[status]++
	if duration > m.slowestDur {
		m.slowestDur = duration
		m.slowestPath = method + " " + path
	}
}

// Snapshot returns a copy of current metrics.
func (m *Metrics) Snapshot() map[string]any {
	requests := m.requests.Load()
	total := m.totalDuration.Load()
	avg := time.Duration(0)
	if requests > 0 {
		avg = time.Duration(total / requests)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make(map[string]int64, len(m.byStatus))
	for code, count := range m.byStatus {
		statuses[fmt.Sprintf("%d", code)] = count
	}

	return map[string]any{
		"requests":      requests,
		"errors":        m.errors.Load(),
		"avg_duration":  avg.String(),
		"avg_ms":        float64(avg.Microseconds()) / 1000.0,
		"slowest":       m.slowestPath,
		"slowest_ms":    float64(m.slowestDur.Microseconds()) / 1000.0,
		"status_counts": statuses,
	}
}

// Reset clears collected metrics.
func (m *Metrics) Reset() {
	m.requests.Store(0)
	m.errors.Store(0)
	m.totalDuration.Store(0)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byStatus = make(map[int]int64)
	m.slowestPath = ""
	m.slowestDur = 0
}

// Prometheus renders metrics in Prometheus text exposition format.
func (m *Metrics) Prometheus() string {
	snap := m.Snapshot()
	var b strings.Builder
	b.WriteString("# HELP zatrano_http_requests_total Total HTTP requests.\n")
	b.WriteString("# TYPE zatrano_http_requests_total counter\n")
	b.WriteString(fmt.Sprintf("zatrano_http_requests_total %d\n", snap["requests"]))
	b.WriteString("# HELP zatrano_http_errors_total Total HTTP responses with status >= 400.\n")
	b.WriteString("# TYPE zatrano_http_errors_total counter\n")
	b.WriteString(fmt.Sprintf("zatrano_http_errors_total %d\n", snap["errors"]))
	b.WriteString("# HELP zatrano_http_request_duration_ms Average request duration in milliseconds.\n")
	b.WriteString("# TYPE zatrano_http_request_duration_ms gauge\n")
	b.WriteString(fmt.Sprintf("zatrano_http_request_duration_ms %.3f\n", snap["avg_ms"]))
	b.WriteString("# HELP zatrano_http_slowest_ms Slowest observed request duration in milliseconds.\n")
	b.WriteString("# TYPE zatrano_http_slowest_ms gauge\n")
	b.WriteString(fmt.Sprintf("zatrano_http_slowest_ms %.3f\n", snap["slowest_ms"]))
	b.WriteString("# HELP zatrano_http_responses_total HTTP responses by status code.\n")
	b.WriteString("# TYPE zatrano_http_responses_total counter\n")
	if statuses, ok := snap["status_counts"].(map[string]int64); ok {
		keys := make([]string, 0, len(statuses))
		for code := range statuses {
			keys = append(keys, code)
		}
		sort.Strings(keys)
		for _, code := range keys {
			b.WriteString(fmt.Sprintf("zatrano_http_responses_total{code=%q} %d\n", code, statuses[code]))
		}
	}
	return b.String()
}

// PrometheusHandler serves Prometheus text metrics.
func (m *Metrics) PrometheusHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		resp := http.Text(m.Prometheus())
		resp.SetContent(resp.Content(), "text/plain; version=0.0.4; charset=utf-8")
		return resp
	}
}

// Timing middleware records duration, sets Server-Timing, and feeds metrics.
func Timing(metrics *Metrics, logFn func(format string, args ...any)) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			start := time.Now()
			resp := next(req)
			duration := time.Since(start)
			status := 200
			if resp != nil {
				status = resp.StatusCode()
				ms := float64(duration.Microseconds()) / 1000.0
				resp.Header("X-Response-Time", fmt.Sprintf("%.2fms", ms))
				resp.Header("Server-Timing", timing.Header(req, duration))
			}
			if metrics != nil {
				metrics.Observe(req.Method(), req.Path(), status, duration)
			}
			if logFn != nil {
				logFn("%s %s -> %d (%.2fms)", req.Method(), req.Path(), status, float64(duration.Microseconds())/1000.0)
			}
			return resp
		}
	}
}
