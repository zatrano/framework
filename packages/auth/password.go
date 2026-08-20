package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/session"
)

var (
	// ErrUserNotFound indicates no account matched the reset email (do not leak to clients).
	ErrUserNotFound = errors.New("passwords.user")
	// ErrResetThrottled indicates a reset was requested too recently.
	ErrResetThrottled = errors.New("passwords.throttled")
)

// PasswordUserProvider can look up and update passwords for reset flows.
type PasswordUserProvider interface {
	UserProvider
	UpdatePassword(email, hashedPassword string) error
}

// CanResetPassword is implemented by users that expose an email for reset links.
type CanResetPassword interface {
	GetEmailForPasswordReset() string
}

// TokenRepository stores password reset tokens.
type TokenRepository interface {
	Create(email, token string) error
	Exists(email, token string) bool
	Delete(email string) error
	RecentlyCreated(email string, within time.Duration) bool
}

// MemoryTokenRepository is an in-memory token store (tests).
type MemoryTokenRepository struct {
	mu    sync.Mutex
	items map[string]memoryToken
	ttl   time.Duration
}

type memoryToken struct {
	hash      string
	createdAt time.Time
}

// NewMemoryTokenRepository creates a memory token repository.
func NewMemoryTokenRepository(ttl time.Duration) *MemoryTokenRepository {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &MemoryTokenRepository{items: make(map[string]memoryToken), ttl: ttl}
}

func (r *MemoryTokenRepository) Create(email, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items[strings.ToLower(email)] = memoryToken{hash: hashToken(token), createdAt: time.Now()}
	return nil
}

func (r *MemoryTokenRepository) Exists(email, token string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[strings.ToLower(email)]
	if !ok {
		return false
	}
	if time.Since(item.createdAt) > r.ttl {
		delete(r.items, strings.ToLower(email))
		return false
	}
	return tokenHashEqual(item.hash, token)
}

func (r *MemoryTokenRepository) Delete(email string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.items, strings.ToLower(email))
	return nil
}

func (r *MemoryTokenRepository) RecentlyCreated(email string, within time.Duration) bool {
	if within <= 0 {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[strings.ToLower(email)]
	if !ok {
		return false
	}
	return time.Since(item.createdAt) < within
}

// DatabaseTokenRepository stores tokens in password_reset_tokens.
type DatabaseTokenRepository struct {
	db     *sql.DB
	driver string
	table  string
	ttl    time.Duration
}

// NewDatabaseTokenRepository creates a database-backed token repository.
func NewDatabaseTokenRepository(db *sql.DB, driver string, ttl time.Duration) *DatabaseTokenRepository {
	return NewDatabaseTokenRepositoryTable(db, driver, "password_reset_tokens", ttl)
}

// NewDatabaseTokenRepositoryTable creates a database-backed token repository with a custom table.
func NewDatabaseTokenRepositoryTable(db *sql.DB, driver, table string, ttl time.Duration) *DatabaseTokenRepository {
	if ttl <= 0 {
		ttl = time.Hour
	}
	if strings.TrimSpace(table) == "" {
		table = "password_reset_tokens"
	}
	return &DatabaseTokenRepository{db: db, driver: driver, table: table, ttl: ttl}
}

func (r *DatabaseTokenRepository) Create(email, token string) error {
	email = strings.ToLower(email)
	_ = r.Delete(email)
	_, err := query.New(r.db, r.driver, r.table).Insert(map[string]any{
		"email":      email,
		"token":      hashToken(token),
		"created_at": time.Now().UTC(),
	})
	return err
}

func (r *DatabaseTokenRepository) Exists(email, token string) bool {
	email = strings.ToLower(email)
	row, err := query.New(r.db, r.driver, r.table).Where("email", email).First()
	if err != nil || row == nil {
		return false
	}
	stored := fmt.Sprint(row["token"])
	createdAt, _ := parseTime(row["created_at"])
	if !createdAt.IsZero() && time.Since(createdAt) > r.ttl {
		_ = r.Delete(email)
		return false
	}
	return tokenHashEqual(stored, token)
}

func (r *DatabaseTokenRepository) Delete(email string) error {
	_, err := query.New(r.db, r.driver, r.table).Where("email", strings.ToLower(email)).Delete()
	return err
}

func (r *DatabaseTokenRepository) RecentlyCreated(email string, within time.Duration) bool {
	if within <= 0 {
		return false
	}
	email = strings.ToLower(email)
	row, err := query.New(r.db, r.driver, r.table).Where("email", email).First()
	if err != nil || row == nil {
		return false
	}
	createdAt, _ := parseTime(row["created_at"])
	if createdAt.IsZero() {
		return false
	}
	return time.Since(createdAt) < within
}

// PasswordBroker creates and consumes password reset tokens.
type PasswordBroker struct {
	tokens TokenRepository
	users  PasswordUserProvider
	// notifier delivers the reset link (should return quickly; prefer async notification.Send).
	notifier   func(email, token, resetURL string) error
	ttl        time.Duration
	throttle   time.Duration
	dispatcher Dispatcher
	sessions   *session.Manager
}

// NewPasswordBroker creates a password broker.
func NewPasswordBroker(tokens TokenRepository, users PasswordUserProvider, ttl time.Duration) *PasswordBroker {
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &PasswordBroker{tokens: tokens, users: users, ttl: ttl, throttle: time.Minute}
}

// SetThrottle configures minimum time between reset requests for the same email.
func (b *PasswordBroker) SetThrottle(d time.Duration) {
	if b == nil {
		return
	}
	b.throttle = d
}

// SetNotifier configures reset-link delivery (email/SMS/etc via notification).
// The callback should enqueue asynchronously and return without waiting on transport I/O.
func (b *PasswordBroker) SetNotifier(fn func(email, token, resetURL string) error) {
	b.notifier = fn
}

// SetMailer is an alias for SetNotifier kept for older call sites.
func (b *PasswordBroker) SetMailer(fn func(email, token, resetURL string) error) {
	b.SetNotifier(fn)
}

// SetDispatcher configures password-reset lifecycle event dispatching.
func (b *PasswordBroker) SetDispatcher(d Dispatcher) {
	b.dispatcher = d
}

// SetSessionManager configures session invalidation after password reset.
func (b *PasswordBroker) SetSessionManager(s *session.Manager) {
	b.sessions = s
}

// CreateToken creates a plain reset token for email (does not mail).
// Returns ErrUserNotFound when no account matches (callers should not leak this).
func (b *PasswordBroker) CreateToken(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := b.users.RetrieveByCredentials(map[string]string{"email": email})
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	if b.throttle > 0 && b.tokens.RecentlyCreated(email, b.throttle) {
		return "", ErrResetThrottled
	}
	token, err := randomToken(40)
	if err != nil {
		return "", err
	}
	if err := b.tokens.Create(email, token); err != nil {
		return "", err
	}
	return token, nil
}

// SendResetLink creates a token and notifies the user with the reset URL.
// Unknown emails succeed silently to avoid account enumeration.
// Delivery is expected to be asynchronous (notification.Send); this call should not wait on SMTP/SMS.
func (b *PasswordBroker) SendResetLink(email, resetURL string) error {
	token, err := b.CreateToken(email)
	if errors.Is(err, ErrUserNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if b.notifier == nil {
		return nil
	}
	return b.notifier(strings.ToLower(email), token, resetURL)
}

// Reset validates the token and updates the password.
func (b *PasswordBroker) Reset(email, token, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !b.tokens.Exists(email, token) {
		return ErrResetTokenInvalid
	}
	user, err := b.users.RetrieveByCredentials(map[string]string{"email": email})
	if err != nil {
		return err
	}
	hashed, err := hashing.Hash(password)
	if err != nil {
		return err
	}
	if err := b.users.UpdatePassword(email, hashed); err != nil {
		return err
	}
	if err := b.tokens.Delete(email); err != nil {
		return err
	}
	if user != nil {
		if rp, ok := b.users.(RememberTokenProvider); ok {
			_ = rp.UpdateRememberToken(user, "")
		}
		if b.sessions != nil {
			_, _ = b.sessions.DestroyOthersForUser(user.AuthID(), "")
		}
	}
	if b.dispatcher != nil {
		_ = b.dispatcher.Dispatch(EventPasswordReset, PasswordResetEvent{User: user, At: time.Now().UTC()})
	}
	return nil
}

// TokenValid reports whether a reset token is valid.
func (b *PasswordBroker) TokenValid(email, token string) bool {
	return b.tokens.Exists(strings.ToLower(email), token)
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// tokenHashEqual compares a stored SHA-256 hex digest to hashToken(token) in constant time.
func tokenHashEqual(stored, token string) bool {
	want := hashToken(token)
	if len(stored) != len(want) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(want)) == 1
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func parseTime(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, v); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}
