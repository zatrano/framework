package contracts

import (
	"context"
	"database/sql"
	"time"
)

// RateLimit is a named limiter policy (key functions stay on the concrete type).
type RateLimit struct {
	MaxAttempts int
	Decay       time.Duration
}

// RateLimiter is returned by Application.RateLimiter().
type RateLimiter interface {
	For(name string, limit RateLimit)
}

// ContextStore is the request/app context surface used via Application.Context().
type ContextStore interface {
	Put(key string, value any)
}

// URLGenerator is the URL helper surface used via Application.URL().
type URLGenerator interface {
	To(path string) string
	Route(name string, params ...map[string]string) (string, error)
	Signed(path string, expiresIn time.Duration, query ...map[string]string) (string, error)
}

// Encrypter is the encryption surface used via Application.Encrypter().
type Encrypter interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(payload string) (string, error)
}

// Hasher is the password hashing surface used via Application.Hash().
type Hasher interface {
	Make(value string) (string, error)
}

// Metrics is the observability surface used via Application.Metrics().
type Metrics interface {
	Snapshot() map[string]any
}

// Health is the health-check surface used via Application.Health().
type Health interface {
	Custom(name string, check func(ctx context.Context) error)
	Database(db *sql.DB)
	Handler() any
}

// MaintenancePayload describes an active maintenance window.
type MaintenancePayload struct {
	Message    string
	RetryAfter int
	AllowedIPs []string
	Secret     string
	Time       string
}

// Maintenance is the maintenance-mode surface used via Application.Maintenance().
type Maintenance interface {
	Enable(payload MaintenancePayload) error
	Disable() error
}

// Exceptions is the exception-handler surface used via Application.Exceptions().
type Exceptions interface {
	Middleware() any
}

// ReportEvent is a captured exception report.
type ReportEvent struct {
	ID        int64
	Message   string
	Level     string
	Path      string
	Method    string
	CreatedAt time.Time
}

// Reports is returned by Application.Reports().
type Reports interface {
	Recent(limit int) []ReportEvent
}
