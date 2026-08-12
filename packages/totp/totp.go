package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GenerateSecret creates a random base32 secret (160-bit).
func GenerateSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

// OTPAuthURL builds an otpauth://totp URL for authenticator apps.
func OTPAuthURL(issuer, account, secret string) string {
	issuer = strings.TrimSpace(issuer)
	account = strings.TrimSpace(account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return "otpauth://totp/" + url.PathEscape(issuer) + ":" + url.PathEscape(account) + "?" + q.Encode()
}

// Code generates a 6-digit TOTP for the given time (30s step).
func Code(secret string, at time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	counter := uint64(at.Unix() / 30)
	return hotp(key, counter), nil
}

// Verify checks a TOTP code with optional window skew (default 1).
func Verify(secret, code string, skew ...int) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	window := 1
	if len(skew) > 0 && skew[0] >= 0 {
		window = skew[0]
	}
	now := time.Now()
	for i := -window; i <= window; i++ {
		at := now.Add(time.Duration(i) * 30 * time.Second)
		expected, err := Code(secret, at)
		if err == nil && hmac.Equal([]byte(expected), []byte(code)) {
			return true
		}
	}
	return false
}

func decodeSecret(secret string) ([]byte, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	secret = strings.ReplaceAll(secret, " ", "")
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(secret)
	}
	if err != nil {
		return nil, fmt.Errorf("totp: invalid secret: %w", err)
	}
	return key, nil
}

func hotp(key []byte, counter uint64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	otp := value % 1000000
	return fmt.Sprintf("%06d", otp)
}
