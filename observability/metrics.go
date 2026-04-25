// Package observability — Prometheus metrikleri ve /metrics uç noktası.
// Fiber v3 ile entegre; her işleyicinin gecikme süresi, durum kodu ve
// toplam istek sayısını otomatik olarak ölçer.
package observability

import (
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registry = prometheus.NewRegistry()

	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Toplam HTTP istek sayısı",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP istek süresi (saniye)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Şu an işlenen istek sayısı",
		},
	)

	mailQueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mail_queue_size",
			Help: "Mail kuyruğundaki iş sayısı",
		},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "DB sorgu süresi (saniye)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
		},
		[]string{"operation"},
	)
)

var registerOnce sync.Once

// Register — tüm metrikleri Prometheus varsayılan kayıt alanına kaydeder.
// main() içinde bir kez çağrılmalıdır.
func Register() {
	registerOnce.Do(func() {
		registry.MustRegister(
			httpRequestsTotal,
			httpRequestDuration,
			httpRequestsInFlight,
			mailQueueSize,
			dbQueryDuration,
		)
	})
}

// Middleware — her isteği otomatik olarak ölçer.
// /metrics, /health gibi uç noktaları atlar.
func Middleware() fiber.Handler {
	skipPaths := map[string]bool{
		"/metrics": true,
		"/health":  true,
		"/healthz": true,
		"/readyz":  true,
	}

	return func(c fiber.Ctx) error {
		path := c.Path()
		if skipPaths[path] {
			return c.Next()
		}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		start := time.Now()
		err := c.Next()
		duration := time.Since(start).Seconds()

		status := strconv.Itoa(c.Response().StatusCode())
		method := c.Method()

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path).Observe(duration)

		return err
	}
}

// MetricsHandler — /metrics uç noktası için promhttp.Handler'ı Fiber'a sarar.
func MetricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
}

// ObserveDB — DB sorgu süresini ölçmek için kullanılır.
// Kullanım: defer observability.ObserveDB("find_user")(time.Now())
func ObserveDB(operation string) func(time.Time) {
	return func(start time.Time) {
		dbQueryDuration.WithLabelValues(operation).Observe(time.Since(start).Seconds())
	}
}

// SetMailQueueSize — mail kuyruğu boyutunu günceller (mailqueue paketi tarafından çağrılır).
func SetMailQueueSize(n float64) {
	mailQueueSize.Set(n)
}
