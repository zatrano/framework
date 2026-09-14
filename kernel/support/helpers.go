package support

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

// BasePath returns the application base path.
func BasePath(parts ...string) string {
	base, err := os.Getwd()
	if err != nil {
		base = "."
	}
	return filepath.Join(append([]string{base}, parts...)...)
}

// RandomBytes returns n random bytes.
func RandomBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(rand.Reader, buf)
	return buf, err
}

// RandomHex returns a random hex string of n bytes.
func RandomHex(n int) (string, error) {
	buf, err := RandomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// RandomBase64 returns a random base64 string of n bytes.
func RandomBase64(n int) (string, error) {
	buf, err := RandomBytes(n)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf), nil
}

// MustRandomHex returns a random hex string of n bytes, or panics.
func MustRandomHex(n int) string {
	s, err := RandomHex(n)
	if err != nil {
		panic("support: " + err.Error())
	}
	return s
}

// ValueOr returns value when not empty, otherwise fallback.
func ValueOr[T comparable](value, fallback T) T {
	var zero T
	if value == zero {
		return fallback
	}
	return value
}
