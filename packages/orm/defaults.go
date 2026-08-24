package orm

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/support/uuid"
)

// HasDefaults provides default attribute values applied on Create when missing.
type HasDefaults interface {
	Defaults() map[string]any
}

// UsesUUIDKeys marks models whose primary key should be auto-filled with a UUID string.
type UsesUUIDKeys interface {
	UsesUUIDKeys() bool
}

// UsesULIDKeys marks models whose primary key should be auto-filled with a ULID-like string.
type UsesULIDKeys interface {
	UsesULIDKeys() bool
}

func applyDefaults[T any](attrs map[string]any) map[string]any {
	var zero T
	ptr := reflect.New(reflect.TypeOf(zero)).Interface()
	d, ok := ptr.(HasDefaults)
	if !ok {
		return attrs
	}
	defaults := d.Defaults()
	if len(defaults) == 0 {
		return attrs
	}
	if attrs == nil {
		attrs = map[string]any{}
	}
	out := make(map[string]any, len(attrs)+len(defaults))
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

func ensureKey[T any](attrs map[string]any) map[string]any {
	if attrs == nil {
		attrs = map[string]any{}
	}
	keyName := KeyName[T]()
	if v, ok := attrs[keyName]; ok && !isZeroAny(v) {
		return attrs
	}
	var zero T
	ptr := reflect.New(reflect.TypeOf(zero)).Interface()
	if u, ok := ptr.(UsesUUIDKeys); ok && u.UsesUUIDKeys() {
		attrs[keyName] = uuid.New()
		return attrs
	}
	if u, ok := ptr.(UsesULIDKeys); ok && u.UsesULIDKeys() {
		attrs[keyName] = newULID()
		return attrs
	}
	return attrs
}

// newULID returns a Crockford-ish 26-char ULID-compatible string (time + randomness).
func newULID() string {
	ms := uint64(time.Now().UnixMilli())
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	// 48-bit time + 80-bit random encoded as 26 Crockford base32 chars.
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	out := make([]byte, 26)
	for i := 9; i >= 0; i-- {
		out[i] = alphabet[ms&31]
		ms >>= 5
	}
	var n uint64
	for i := 0; i < 10; i++ {
		n = (n << 8) | uint64(buf[i])
	}
	for i := 25; i >= 10; i-- {
		out[i] = alphabet[n&31]
		n >>= 5
	}
	return string(out)
}

// NewUUID returns a random UUID string (helper for models/tests).
func NewUUID() string { return uuid.New() }

// NewULID returns a new ULID-like identifier.
func NewULID() string { return newULID() }

// IsUUID reports whether s looks like a UUID.
func IsUUID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 36 {
		return false
	}
	_, err := hex.DecodeString(strings.ReplaceAll(s, "-", ""))
	return err == nil && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// MustParseKey converts string keys for Find helpers.
func MustParseKey(v any) string {
	return strings.TrimSpace(fmt.Sprint(v))
}
