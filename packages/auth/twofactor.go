package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/auth/totp"
	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
)

const (
	twoFactorSessionKey         = "auth.two_factor_user_id"
	twoFactorRememberSessionKey = "auth.two_factor_remember"
	twoFactorDeviceCookiePrefix = "2fa_device_"
)

// ErrTwoFactorRequired indicates that the primary credentials are valid and a second factor is pending.
var ErrTwoFactorRequired = fmt.Errorf("two-factor authentication required")

type twoFactorUser interface {
	GetTwoFactorSecret() string
	GetTwoFactorRecoveryCodes() string
	GetTwoFactorConfirmedAt() *time.Time
}

func (m *Manager) twoFactorValues(user Authenticatable) (secret, recovery string, confirmed bool) {
	if user == nil {
		return "", "", false
	}
	var rawSecret, rawRecovery string
	if u, ok := user.(twoFactorUser); ok {
		rawSecret = u.GetTwoFactorSecret()
		rawRecovery = u.GetTwoFactorRecoveryCodes()
		confirmed = u.GetTwoFactorConfirmedAt() != nil
	} else if u, ok := user.(*GenericUser); ok {
		rawSecret = strings.TrimSpace(fmt.Sprint(u.Get("two_factor_secret")))
		if rawSecret == "<nil>" {
			rawSecret = ""
		}
		rawRecovery = strings.TrimSpace(fmt.Sprint(u.Get("two_factor_recovery_codes")))
		if rawRecovery == "<nil>" {
			rawRecovery = ""
		}
		confirmed = u.Get("two_factor_confirmed_at") != nil && fmt.Sprint(u.Get("two_factor_confirmed_at")) != "" && fmt.Sprint(u.Get("two_factor_confirmed_at")) != "<nil>"
	}
	return m.decryptSensitive(rawSecret), m.decryptSensitive(rawRecovery), confirmed
}

func (m *Manager) encryptSensitive(value string) string {
	if m == nil || m.crypt == nil || strings.TrimSpace(value) == "" {
		return value
	}
	out, err := m.crypt.Encrypt(value)
	if err != nil {
		return value
	}
	return out
}

func (m *Manager) decryptSensitive(value string) string {
	value = strings.TrimSpace(value)
	if m == nil || m.crypt == nil || value == "" {
		return value
	}
	out, err := m.crypt.Decrypt(value)
	if err != nil {
		// Allow plaintext values from before encryption was enabled.
		return value
	}
	return out
}

func (m *Manager) updateTwoFactor(user Authenticatable, attrs map[string]any) error {
	if m == nil || m.Guard() == nil {
		return fmt.Errorf("auth guard unavailable")
	}
	updater, ok := m.Guard().Provider().(AttributeUpdater)
	if !ok {
		return fmt.Errorf("user provider does not support two-factor updates")
	}
	if err := updater.UpdateAttributes(user.AuthID(), attrs); err != nil {
		return err
	}
	if generic, ok := user.(*GenericUser); ok {
		for key, value := range attrs {
			generic.Attributes[key] = value
		}
	}
	return nil
}

func (m *Manager) issuer() string {
	if m == nil || strings.TrimSpace(m.twoFactorIssuer) == "" {
		return "ZATRANO"
	}
	return m.twoFactorIssuer
}

func twoFactorDeviceCookieName(guard string) string {
	if strings.TrimSpace(guard) == "" {
		guard = "web"
	}
	return twoFactorDeviceCookiePrefix + guard
}

// EnableTwoFactor creates an unconfirmed TOTP secret and recovery codes.
func (m *Manager) EnableTwoFactor(user Authenticatable) (secret, otpauthURL string, recoveryCodes []string, err error) {
	if user == nil {
		return "", "", nil, fmt.Errorf("user is required")
	}
	secret, err = totp.GenerateSecret()
	if err != nil {
		return "", "", nil, err
	}
	recoveryCodes, err = makeRecoveryCodes()
	if err != nil {
		return "", "", nil, err
	}
	joined := strings.Join(recoveryCodes, ",")
	if err := m.updateTwoFactor(user, map[string]any{
		"two_factor_secret":         m.encryptSensitive(secret),
		"two_factor_recovery_codes": m.encryptSensitive(joined),
		"two_factor_confirmed_at":   nil,
	}); err != nil {
		return "", "", nil, err
	}
	otpauthURL = totp.OTPAuthURL(m.issuer(), EmailForVerification(user), secret)
	return secret, otpauthURL, recoveryCodes, nil
}

// ConfirmTwoFactor confirms a pending TOTP secret.
func (m *Manager) ConfirmTwoFactor(user Authenticatable, code string) error {
	secret, _, _ := m.twoFactorValues(user)
	if secret == "" || !totp.Verify(secret, strings.TrimSpace(code)) {
		return errors.New("auth.invalid_code")
	}
	return m.updateTwoFactor(user, map[string]any{"two_factor_confirmed_at": time.Now().UTC()})
}

// DisableTwoFactor removes all second-factor data after password confirmation.
func (m *Manager) DisableTwoFactor(user Authenticatable, password string) error {
	if user == nil || !hashing.Check(password, user.AuthPassword()) {
		return ErrCurrentPassword
	}
	return m.updateTwoFactor(user, map[string]any{
		"two_factor_secret": nil, "two_factor_recovery_codes": nil, "two_factor_confirmed_at": nil,
	})
}

// HasTwoFactorEnabled reports whether a user has confirmed second-factor authentication.
func (m *Manager) HasTwoFactorEnabled(user Authenticatable) bool {
	secret, _, confirmed := m.twoFactorValues(user)
	return secret != "" && confirmed
}

// GenerateRecoveryCodes creates and stores a new recovery-code set.
func (m *Manager) GenerateRecoveryCodes(user Authenticatable) ([]string, error) {
	if !m.HasTwoFactorEnabled(user) {
		return nil, fmt.Errorf("two-factor authentication is not enabled")
	}
	codes, err := makeRecoveryCodes()
	if err != nil {
		return nil, err
	}
	return codes, m.updateTwoFactor(user, map[string]any{"two_factor_recovery_codes": m.encryptSensitive(strings.Join(codes, ","))})
}

// ReplaceRecoveryCodes is an alias for GenerateRecoveryCodes.
func (m *Manager) ReplaceRecoveryCodes(user Authenticatable) ([]string, error) {
	return m.GenerateRecoveryCodes(user)
}

// VerifyTwoFactorCode validates a TOTP code or consumes a recovery code.
func (m *Manager) VerifyTwoFactorCode(user Authenticatable, code string) bool {
	secret, recovery, confirmed := m.twoFactorValues(user)
	code = strings.TrimSpace(code)
	if !confirmed || secret == "" {
		return false
	}
	if totp.Verify(secret, code) {
		return true
	}
	codes := splitRecoveryCodes(recovery)
	for i, saved := range codes {
		if code != saved {
			continue
		}
		codes = append(codes[:i], codes[i+1:]...)
		return m.updateTwoFactor(user, map[string]any{"two_factor_recovery_codes": m.encryptSensitive(strings.Join(codes, ","))}) == nil
	}
	return false
}

// ChallengeTwoFactor completes a pending two-factor challenge and logs the user in.
// Optional rememberDevice queues a trusted-device cookie when true.
func (m *Manager) ChallengeTwoFactor(req *http.Request, code string, rememberDevice ...bool) (bool, error) {
	if req == nil || req.Session() == nil {
		return false, fmt.Errorf("session not available")
	}
	id := req.Session().Get(twoFactorSessionKey)
	if id == nil || fmt.Sprint(id) == "" {
		return false, fmt.Errorf("two-factor challenge is not pending")
	}
	if m.lockouts != nil && m.lockouts.locked(twoFactorLockoutKey(req, id)) {
		m.dispatch(EventLockout, LockoutEvent{Request: req, Guard: m.Guard().name, At: time.Now().UTC()})
		return false, ErrLockout
	}
	user, err := m.Guard().Provider().RetrieveByID(id)
	if err != nil || user == nil {
		return false, err
	}
	if !m.VerifyTwoFactorCode(user, code) {
		if m.lockouts != nil {
			if m.lockouts.hit(twoFactorLockoutKey(req, id)) {
				m.dispatch(EventLockout, LockoutEvent{Request: req, User: user, Guard: m.Guard().name, At: time.Now().UTC()})
				return false, ErrLockout
			}
		}
		return false, errors.New("auth.invalid_code")
	}
	if m.lockouts != nil {
		m.lockouts.clear(twoFactorLockoutKey(req, id))
	}
	rememberLogin := req.Session().Get(twoFactorRememberSessionKey) == true
	req.Session().Forget(twoFactorSessionKey)
	req.Session().Forget(twoFactorRememberSessionKey)
	if err := m.Login(req, user, rememberLogin); err != nil {
		return false, err
	}
	if len(rememberDevice) > 0 && rememberDevice[0] {
		m.queueTrustedDevice(req, user)
	}
	m.dispatch(EventTwoFactorAuthenticated, TwoFactorAuthenticatedEvent{Request: req, User: user, Guard: m.Guard().name, At: time.Now().UTC()})
	return true, nil
}

// HasTrustedDevice reports whether the request carries a valid remember-device cookie for user.
func (m *Manager) HasTrustedDevice(req *http.Request, user Authenticatable) bool {
	if m == nil || req == nil || user == nil || m.rememberDeviceDays <= 0 {
		return false
	}
	raw := strings.TrimSpace(req.Cookie(twoFactorDeviceCookieName(m.Guard().name)))
	if raw == "" {
		return false
	}
	payload := m.decryptSensitive(raw)
	parts := strings.SplitN(payload, "|", 2)
	if len(parts) != 2 {
		return false
	}
	if parts[0] != fmt.Sprint(user.AuthID()) {
		return false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return true
}

// ForgetTrustedDevice clears the remember-device cookie for the current guard.
func (m *Manager) ForgetTrustedDevice(req *http.Request) {
	if m == nil || req == nil {
		return
	}
	req.Cookies().Forget(twoFactorDeviceCookieName(m.Guard().name))
}

func (m *Manager) queueTrustedDevice(req *http.Request, user Authenticatable) {
	if m == nil || req == nil || user == nil || m.rememberDeviceDays <= 0 {
		return
	}
	exp := time.Now().Add(time.Duration(m.rememberDeviceDays) * 24 * time.Hour)
	payload := fmt.Sprintf("%s|%d", fmt.Sprint(user.AuthID()), exp.Unix())
	value := m.encryptSensitive(payload)
	if value == payload && m.crypt == nil {
		// Without an encrypter, skip trusted-device cookies (would be forgeable).
		return
	}
	minutes := int(time.Until(exp).Minutes())
	if minutes < 1 {
		minutes = 1
	}
	req.Cookies().Queue(twoFactorDeviceCookieName(m.Guard().name), value, minutes)
}

func makeRecoveryCodes() ([]string, error) {
	codes := make([]string, 8)
	for i := range codes {
		raw := make([]byte, 5)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		codes[i] = hex.EncodeToString(raw)
	}
	return codes, nil
}

func splitRecoveryCodes(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if code := strings.TrimSpace(part); code != "" {
			out = append(out, code)
		}
	}
	return out
}
