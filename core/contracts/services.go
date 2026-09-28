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

// RateLimiter is resolved from the container (ratelimit.From), not from App.
type RateLimiter interface {
	For(name string, limit RateLimit)
}

// ContextStore is the request/app context surface used via Application.Context().
type ContextStore interface {
	Put(key string, value any)
}

// URLGenerator is resolved from the container (url.From), not from App.
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

// Hasher is resolved from the container (hashing.From), not from App.
type Hasher interface {
	Make(value string) (string, error)
}

// Metrics is resolved from the container (observability.From), not from App.
type Metrics interface {
	Snapshot() map[string]any
}

// Health is resolved from the container (health.From), not from App.
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

// Maintenance is resolved from the container (maintenance.From), not from App.
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
