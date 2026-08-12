package health

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Status values for a check.
const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	StatusFail     = "fail"
)

// Result is the outcome of one health check.
type Result struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"`
	Message    string  `json:"message,omitempty"`
	Duration   string  `json:"duration"`
	DurationMS float64 `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
}

// Checker runs a named health check.
type Checker struct {
	Name  string
	Check func(ctx context.Context) error
}

// Manager runs registered health checks.
type Manager struct {
	mu       sync.RWMutex
	checkers []Checker
	timeout  time.Duration
}

// New creates a health manager.
func New() *Manager {
	return &Manager{
		checkers: make([]Checker, 0),
		timeout:  3 * time.Second,
	}
}

// SetTimeout configures per-check timeout.
func (m *Manager) SetTimeout(d time.Duration) {
	m.timeout = d
}

// Register adds a checker.
func (m *Manager) Register(name string, check func(ctx context.Context) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers = append(m.checkers, Checker{Name: name, Check: check})
}

// Database registers a database ping check.
func (m *Manager) Database(db *sql.DB) {
	m.Register("database", func(ctx context.Context) error {
		if db == nil {
			return fmt.Errorf("database is not configured")
		}
		return db.PingContext(ctx)
	})
}

// Disk registers a writable directory check.
func (m *Manager) Disk(path string) {
	m.Register("disk", func(ctx context.Context) error {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
		tmp := filepath.Join(path, ".health")
		if err := os.WriteFile(tmp, []byte("ok"), 0o644); err != nil {
			return err
		}
		return os.Remove(tmp)
	})
}

// Custom registers an arbitrary named check.
func (m *Manager) Custom(name string, check func(ctx context.Context) error) {
	m.Register(name, check)
}

// Run executes all checks.
func (m *Manager) Run(ctx context.Context) (overall string, results []Result) {
	m.mu.RLock()
	checkers := append([]Checker{}, m.checkers...)
	timeout := m.timeout
	m.mu.RUnlock()

	results = make([]Result, 0, len(checkers))
	overall = StatusOK
	for _, checker := range checkers {
		start := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		err := checker.Check(checkCtx)
		cancel()
		duration := time.Since(start)
		item := Result{
			Name:       checker.Name,
			Status:     StatusOK,
			Duration:   duration.String(),
			DurationMS: float64(duration.Microseconds()) / 1000.0,
		}
		if err != nil {
			item.Status = StatusFail
			item.Error = err.Error()
			item.Message = err.Error()
			overall = StatusFail
		}
		results = append(results, item)
	}
	if len(results) == 0 {
		overall = StatusOK
	}
	return overall, results
}

// Handler returns an HTTP handler for health checks.
func (m *Manager) Handler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		overall, results := m.Run(req.Raw().Context())
		status := 200
		if overall == StatusFail {
			status = 503
		}
		return http.JSON(map[string]any{
			"status": overall,
			"checks": results,
		}).Status(status)
	}
}
