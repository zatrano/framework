package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/session"
)

// LegacySessionKey is the pre–multi-guard session key (read for migration).
const LegacySessionKey = "auth_user_id"

const requestUserKeyPrefix = "auth.user."
const requestLoggedOutKeyPrefix = "auth.logged_out."
const requestViaRememberKeyPrefix = "auth.via_remember."

// SessionUserKey returns the per-guard session key for the authenticated user id.
func SessionUserKey(guard string) string {
	if strings.TrimSpace(guard) == "" {
		guard = "web"
	}
	return "login_" + guard + "_id"
}

func (g *Guard) guardName() string {
	if g == nil || strings.TrimSpace(g.name) == "" {
		return "web"
	}
	return g.name
}

func (g *Guard) sessionUserKey() string {
	return SessionUserKey(g.guardName())
}

func (g *Guard) requestUserKey() string {
	return requestUserKeyPrefix + g.guardName()
}

func (g *Guard) requestLoggedOutKey() string {
	return requestLoggedOutKeyPrefix + g.guardName()
}

func (g *Guard) requestViaRememberKey() string {
	return requestViaRememberKeyPrefix + g.guardName()
}

// UserProvider retrieves users for authentication.
type UserProvider interface {
	RetrieveByID(id any) (Authenticatable, error)
	RetrieveByCredentials(credentials map[string]string) (Authenticatable, error)
	ValidateCredentials(user Authenticatable, credentials map[string]string) bool
}

// Authenticatable represents an authenticatable user.
type Authenticatable interface {
	AuthID() any
	AuthPassword() string
}

// Guard authenticates users for a request/session.
type Guard struct {
	name     string
	provider UserProvider
	manager  *Manager
}

// NewGuard creates an auth guard.
func NewGuard(name string, provider UserProvider) *Guard {
	return &Guard{name: name, provider: provider}
}

// Provider returns the user provider.
func (g *Guard) Provider() UserProvider {
	if g == nil {
		return nil
	}
	return g.provider
}

// Attempt authenticates with credentials and logs the user in.
func (g *Guard) Attempt(req *http.Request, credentials map[string]string, remember ...bool) (bool, error) {
	if g.manager != nil {
		g.manager.dispatch(EventAttempting, AttemptingEvent{Request: req, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
		if g.manager.lockouts.locked(lockoutKey(req, credentials)) {
			g.manager.dispatch(EventLockout, LockoutEvent{Request: req, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
			return false, ErrLockout
		}
	}
	user, err := g.provider.RetrieveByCredentials(credentials)
	if err != nil || user == nil {
		if g.manager != nil {
			locked := g.manager.lockouts.hit(lockoutKey(req, credentials))
			g.manager.dispatch(EventFailed, FailedEvent{Request: req, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
			if locked {
				g.manager.dispatch(EventLockout, LockoutEvent{Request: req, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
				return false, ErrLockout
			}
		}
		return false, err
	}
	if !g.provider.ValidateCredentials(user, credentials) {
		if g.manager != nil {
			locked := g.manager.lockouts.hit(lockoutKey(req, credentials))
			g.manager.dispatch(EventFailed, FailedEvent{Request: req, User: user, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
			if locked {
				g.manager.dispatch(EventLockout, LockoutEvent{Request: req, User: user, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
				return false, ErrLockout
			}
		}
		return false, nil
	}
	if g.manager != nil {
		g.manager.lockouts.clear(lockoutKey(req, credentials))
		if g.manager.HasTwoFactorEnabled(user) {
			if g.manager.HasTrustedDevice(req, user) {
				if err := g.Login(req, user, remember...); err != nil {
					return false, err
				}
				g.maybeRehashPassword(user, credentials["password"])
				return true, nil
			}
			if sess := req.Session(); sess != nil {
				sess.Put(twoFactorSessionKey, fmt.Sprint(user.AuthID()))
				sess.Put(twoFactorRememberSessionKey, wantsRemember(remember))
			}
			g.manager.dispatch(EventTwoFactorChallenged, TwoFactorChallengedEvent{Request: req, User: user, Credentials: credentials, Guard: g.name, At: time.Now().UTC()})
			return false, ErrTwoFactorRequired
		}
	}
	if err := g.Login(req, user, remember...); err != nil {
		return false, err
	}
	g.maybeRehashPassword(user, credentials["password"])
	return true, nil
}

func (g *Guard) maybeRehashPassword(user Authenticatable, plain string) {
	if user == nil || strings.TrimSpace(plain) == "" {
		return
	}
	if !hashing.NeedsRehash(user.AuthPassword()) {
		return
	}
	hashed, err := hashing.Hash(plain)
	if err != nil {
		return
	}
	updater, ok := g.provider.(AttributeUpdater)
	if !ok || updater == nil {
		return
	}
	_ = updater.UpdateAttributes(user.AuthID(), map[string]any{"password": hashed})
	if generic, ok := user.(*GenericUser); ok && generic.Attributes != nil {
		generic.Attributes["password"] = hashed
	}
}

// Login stores the user in the session.
// When remember is true and the provider supports remember tokens, a long-lived cookie is queued.
func (g *Guard) Login(req *http.Request, user Authenticatable, remember ...bool) error {
	sess := req.Session()
	if sess == nil {
		return fmt.Errorf("session not available")
	}
	sess.Put(g.sessionUserKey(), fmt.Sprint(user.AuthID()))
	if err := sess.Regenerate(); err != nil {
		return err
	}
	req.Set(g.requestLoggedOutKey(), false)
	req.Set(g.requestUserKey(), user)
	if wantsRemember(remember) {
		if err := g.queueRememberCookie(req, user); err != nil {
			return err
		}
	}
	if g.manager != nil {
		g.manager.dispatch(EventLogin, LoginEvent{Request: req, User: user, Guard: g.name, At: time.Now().UTC()})
	}
	return nil
}

// Logout clears the authenticated user and forgets the remember cookie/token.
func (g *Guard) Logout(req *http.Request) error {
	user := g.userFromSessionOrCache(req)
	g.clearRememberCookie(req, user)
	sess := req.Session()
	if sess != nil {
		if inv, ok := sess.(interface{ Invalidate() error }); ok {
			_ = inv.Invalidate()
		} else {
			sess.Forget(g.sessionUserKey())
			if g.guardName() == "web" {
				sess.Forget(LegacySessionKey)
			}
			_ = sess.Regenerate()
		}
	}
	req.Set(g.requestUserKey(), nil)
	req.Set(g.requestLoggedOutKey(), true)
	ClearPasswordConfirmation(req)
	if g.manager != nil {
		g.manager.ForgetTrustedDevice(req)
		g.manager.dispatch(EventLogout, LogoutEvent{Request: req, User: user, Guard: g.name, At: time.Now().UTC()})
		g.manager.dispatch(EventCurrentDeviceLogout, CurrentDeviceLogoutEvent{Request: req, User: user, Guard: g.name, At: time.Now().UTC()})
	}
	return nil
}

// Validate checks credentials without changing request or session state.
func (g *Guard) Validate(credentials map[string]string) (bool, error) {
	user, err := g.provider.RetrieveByCredentials(credentials)
	if err != nil || user == nil {
		return false, err
	}
	return g.provider.ValidateCredentials(user, credentials), nil
}

// LoginUsingID retrieves and logs in a user by identifier.
func (g *Guard) LoginUsingID(req *http.Request, id any, remember ...bool) error {
	user, err := g.provider.RetrieveByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	return g.Login(req, user, remember...)
}

// Once authenticates a user for only the current request.
func (g *Guard) Once(req *http.Request, user Authenticatable) error {
	if req == nil || user == nil {
		return fmt.Errorf("user and request are required")
	}
	req.Set(g.requestUserKey(), user)
	req.Set(g.requestLoggedOutKey(), false)
	return nil
}

// OnceUsingID authenticates an existing user only for the current request.
func (g *Guard) OnceUsingID(req *http.Request, id any) error {
	user, err := g.provider.RetrieveByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}
	return g.Once(req, user)
}

// ViaRemember reports whether this request was restored using a remember cookie.
func (g *Guard) ViaRemember(req *http.Request) bool {
	return req != nil && req.Get(g.requestViaRememberKey()) == true
}

// Check reports whether a user is authenticated.
func (g *Guard) Check(req *http.Request) bool {
	return g.User(req) != nil
}

// Guest reports whether no user is authenticated.
func (g *Guard) Guest(req *http.Request) bool {
	return !g.Check(req)
}

// ID returns the authenticated user id.
func (g *Guard) ID(req *http.Request) any {
	user := g.User(req)
	if user == nil {
		return nil
	}
	return user.AuthID()
}

// User returns the authenticated user for this request.
// When the session is empty, a valid remember-me cookie may restore the login.
func (g *Guard) User(req *http.Request) Authenticatable {
	if req != nil && req.Get(g.requestLoggedOutKey()) == true {
		return nil
	}
	if cached := req.Get(g.requestUserKey()); cached != nil {
		if user, ok := cached.(Authenticatable); ok {
			return user
		}
	}

	if user := g.userFromSession(req); user != nil {
		req.Set(g.requestUserKey(), user)
		return user
	}

	if user := g.userFromRememberCookie(req); user != nil {
		if sess := req.Session(); sess != nil {
			sess.Put(g.sessionUserKey(), fmt.Sprint(user.AuthID()))
		}
		req.Set(g.requestUserKey(), user)
		req.Set(g.requestViaRememberKey(), true)
		return user
	}
	return nil
}

func (g *Guard) userFromSession(req *http.Request) Authenticatable {
	if g.provider == nil || req == nil {
		return nil
	}
	sess := req.Session()
	if sess == nil {
		return nil
	}
	raw := sess.Get(g.sessionUserKey())
	if raw == nil || fmt.Sprint(raw) == "" {
		// Migrate legacy single-key sessions for the web guard.
		if g.guardName() == "web" {
			raw = sess.Get(LegacySessionKey)
		}
	}
	if raw == nil || fmt.Sprint(raw) == "" {
		return nil
	}
	user, err := g.provider.RetrieveByID(raw)
	if err != nil || user == nil {
		return nil
	}
	return user
}

func (g *Guard) userFromSessionOrCache(req *http.Request) Authenticatable {
	if req == nil {
		return nil
	}
	if cached := req.Get(g.requestUserKey()); cached != nil {
		if user, ok := cached.(Authenticatable); ok {
			return user
		}
	}
	return g.userFromSession(req)
}

// Manager resolves auth guards.
type Manager struct {
	defaultGuard       string
	guards             map[string]*Guard
	dispatcher         Dispatcher
	sessions           *session.Manager
	lockouts           *lockoutStore
	crypt              Crypt
	twoFactorIssuer    string
	rememberDeviceDays int
}

// Crypt encrypts sensitive auth payloads (two-factor secrets).
type Crypt interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(payload string) (string, error)
}

// NewManager creates an auth manager.
func NewManager(defaultGuard string) *Manager {
	if strings.TrimSpace(defaultGuard) == "" {
		defaultGuard = "web"
	}
	return &Manager{
		defaultGuard:       defaultGuard,
		guards:             make(map[string]*Guard),
		lockouts:           newLockoutStore(5, time.Minute),
		twoFactorIssuer:    "ZATRANO",
		rememberDeviceDays: 30,
	}
}

// Extend registers a guard.
func (m *Manager) Extend(name string, guard *Guard) {
	if guard != nil {
		guard.manager = m
	}
	m.guards[name] = guard
}

// ShouldUse sets the default guard for subsequent auth operations.
func (m *Manager) ShouldUse(name string) {
	if m == nil || strings.TrimSpace(name) == "" {
		return
	}
	m.defaultGuard = name
}

// SetDefaultDriver is an alias for ShouldUse.
func (m *Manager) SetDefaultDriver(name string) { m.ShouldUse(name) }

// GetDefaultDriver returns the default guard name.
func (m *Manager) GetDefaultDriver() string {
	if m == nil {
		return ""
	}
	return m.defaultGuard
}

// SetDispatcher configures lifecycle event dispatching.
func (m *Manager) SetDispatcher(d Dispatcher) { m.dispatcher = d }

// SetSessionManager configures file-session management for device logout.
func (m *Manager) SetSessionManager(s *session.Manager) { m.sessions = s }

// SetEncrypter configures encryption for two-factor secrets and recovery codes.
func (m *Manager) SetEncrypter(c Crypt) { m.crypt = c }

// SetLockout configures login attempt limits.
func (m *Manager) SetLockout(maxAttempts int, decay time.Duration) {
	cache := AttemptCounter(nil)
	if m.lockouts != nil {
		cache = m.lockouts.cache
	}
	m.lockouts = newLockoutStore(maxAttempts, decay)
	if cache != nil {
		m.lockouts.setCache(cache)
	}
}

// SetLockoutCache configures shared lockout storage (e.g. cache.Manager / cache.Store).
func (m *Manager) SetLockoutCache(c AttemptCounter) {
	if m == nil {
		return
	}
	if m.lockouts == nil {
		m.lockouts = newLockoutStore(5, time.Minute)
	}
	m.lockouts.setCache(c)
}

// SetTwoFactorIssuer sets the TOTP otpauth issuer label.
func (m *Manager) SetTwoFactorIssuer(issuer string) {
	if m == nil {
		return
	}
	if strings.TrimSpace(issuer) == "" {
		issuer = "ZATRANO"
	}
	m.twoFactorIssuer = issuer
}

// SetRememberDeviceDays configures how long a trusted 2FA device cookie lasts.
func (m *Manager) SetRememberDeviceDays(days int) {
	if m == nil {
		return
	}
	if days < 0 {
		days = 0
	}
	m.rememberDeviceDays = days
}

// InvalidateSessionsForUser destroys all persisted sessions for a user.
func (m *Manager) InvalidateSessionsForUser(userID any) error {
	if m == nil || m.sessions == nil || userID == nil {
		return nil
	}
	_, err := m.sessions.DestroyOthersForUser(userID, "")
	return err
}

func (m *Manager) dispatch(name string, event any) {
	if m != nil && m.dispatcher != nil {
		_ = m.dispatcher.Dispatch(name, event)
	}
}

// DispatchEvent emits an authentication lifecycle event for integrations.
func (m *Manager) DispatchEvent(name string, event any) {
	m.dispatch(name, event)
}

// Guard returns a named guard.
func (m *Manager) Guard(name ...string) *Guard {
	guardName := m.defaultGuard
	if len(name) > 0 && name[0] != "" {
		guardName = name[0]
	}
	return m.guards[guardName]
}

// Attempt proxies to the default guard.
func (m *Manager) Attempt(req *http.Request, credentials map[string]string, remember ...bool) (bool, error) {
	return m.Guard().Attempt(req, credentials, remember...)
}

// Login proxies to the default guard.
func (m *Manager) Login(req *http.Request, user Authenticatable, remember ...bool) error {
	return m.Guard().Login(req, user, remember...)
}

func (m *Manager) Validate(credentials map[string]string) (bool, error) {
	return m.Guard().Validate(credentials)
}

func (m *Manager) LoginUsingID(req *http.Request, id any, remember ...bool) error {
	return m.Guard().LoginUsingID(req, id, remember...)
}

func (m *Manager) Once(req *http.Request, user Authenticatable) error {
	return m.Guard().Once(req, user)
}
func (m *Manager) OnceUsingID(req *http.Request, id any) error { return m.Guard().OnceUsingID(req, id) }
func (m *Manager) ViaRemember(req *http.Request) bool          { return m.Guard().ViaRemember(req) }

// LogoutOtherDevices invalidates every other persisted session after password confirmation.
func (m *Manager) LogoutOtherDevices(req *http.Request, password string) error {
	user := m.User(req)
	if user == nil {
		return ErrUnauthenticated
	}
	if !hashing.Check(password, user.AuthPassword()) {
		return ErrCurrentPassword
	}
	if rp, ok := m.Guard().Provider().(RememberTokenProvider); ok {
		if err := rp.UpdateRememberToken(user, ""); err != nil {
			return err
		}
		if req != nil {
			req.Cookies().Forget(rememberCookieName(m.Guard().name))
		}
	}
	if m.sessions != nil && req != nil && req.Session() != nil {
		if _, err := m.sessions.DestroyOthersForUser(user.AuthID(), req.Session().ID()); err != nil {
			return err
		}
	}
	m.dispatch(EventOtherDeviceLogout, OtherDeviceLogoutEvent{Request: req, User: user, Guard: m.Guard().name, At: time.Now().UTC()})
	return nil
}

// Logout proxies to the default guard.
func (m *Manager) Logout(req *http.Request) error {
	return m.Guard().Logout(req)
}

// Check proxies to the default guard.
func (m *Manager) Check(req *http.Request) bool {
	return m.Guard().Check(req)
}

// ID proxies to the default guard.
func (m *Manager) ID(req *http.Request) any {
	return m.Guard().ID(req)
}

// User proxies to the default guard.
func (m *Manager) User(req *http.Request) Authenticatable {
	return m.Guard().User(req)
}

// Middleware ensures the user is authenticated.
// Optional guard names try each guard until one authenticates (e.g. "web", "api").
func Middleware(manager *Manager, guards ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			user, guardName := firstAuthenticated(manager, req, guards...)
			if user == nil {
				if req.WantsJSON() {
					return http.JSON(map[string]any{"message": "Unauthenticated."}).Status(401)
				}
				CaptureIntendedFromRequest(req)
				return http.Redirect("/auth/login")
			}
			req.Set("user", user)
			req.Set("auth.guard", guardName)
			return next(req)
		}
	}
}

func firstAuthenticated(manager *Manager, req *http.Request, guards ...string) (Authenticatable, string) {
	if manager == nil {
		return nil, ""
	}
	names := guards
	if len(names) == 0 {
		names = []string{manager.GetDefaultDriver()}
	}
	for _, name := range names {
		g := manager.Guard(name)
		if g == nil {
			continue
		}
		if user := g.User(req); user != nil {
			return user, g.guardName()
		}
	}
	return nil, ""
}

// GuestMiddleware redirects authenticated users away from guest pages.
func GuestMiddleware(manager *Manager, guards ...string) routing.MiddlewareFunc {
	return RedirectIfAuthenticated(manager, "/", guards...)
}

// RedirectIfAuthenticated redirects authenticated users to the given path (default "/").
// Optional trailing args after redirectTo are guard names; if omitted, uses the default guard.
// Prefer RedirectIfAuthenticatedGuards when both a custom redirect and guards are needed.
func RedirectIfAuthenticated(manager *Manager, redirectTo string, guards ...string) routing.MiddlewareFunc {
	target := "/"
	if strings.TrimSpace(redirectTo) != "" {
		target = redirectTo
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			user, _ := firstAuthenticated(manager, req, guards...)
			if user != nil {
				if req.WantsJSON() {
					return http.JSON(map[string]any{
						"message":  "Already authenticated.",
						"redirect": target,
					}).Status(409)
				}
				return http.Redirect(target)
			}
			return next(req)
		}
	}
}

// PasswordValidator validates a password hash.
func PasswordValidator(plain, hashed string) bool {
	return hashing.Check(plain, hashed)
}
