package contracts

import (
	"context"
	"database/sql"
	"time"

	"github.com/zatrano/framework/packages/maintenance"
	"github.com/zatrano/framework/packages/ratelimit"
	"github.com/zatrano/framework/packages/report"
	"github.com/zatrano/framework/packages/routing"
)

// RateLimiter is returned by Application.RateLimiter().
// No external call sites exist; For is the method invoked on the same instance during kernel boot.
type RateLimiter interface {
	For(name string, limit ratelimit.Limit)
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
	Handler() routing.HandlerFunc
}

// Maintenance is the maintenance-mode surface used via Application.Maintenance().
type Maintenance interface {
	Enable(payload maintenance.Payload) error
	Disable() error
}

// Exceptions is the exception-handler surface used via Application.Exceptions().
type Exceptions interface {
	Middleware() routing.MiddlewareFunc
}

// Reports is returned by Application.Reports().
// No external call sites exist; Recent is the public read method on the boot-wired instance.
type Reports interface {
	Recent(limit int) []report.Event
}
