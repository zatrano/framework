package orm

import (
	"fmt"
	"strings"
	"sync"
)

// CastHandler converts values when reading from / writing to storage.
type CastHandler struct {
	In  func(value any) (any, error)
	Out func(value any) any
}

var (
	customCastMu sync.RWMutex
	customCasts  = map[string]CastHandler{}
	castCrypt    CastCrypt
)

// CastCrypt encrypts/decrypts string payloads for the "encrypted" cast.
type CastCrypt interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(payload string) (string, error)
}

// RegisterCast registers a custom cast by name (e.g. "money").
func RegisterCast(name string, in func(any) (any, error), out func(any) any) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return
	}
	customCastMu.Lock()
	defer customCastMu.Unlock()
	customCasts[name] = CastHandler{In: in, Out: out}
}

// HasCast reports whether a custom cast is registered.
func HasCast(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	customCastMu.RLock()
	defer customCastMu.RUnlock()
	_, ok := customCasts[name]
	return ok
}

// ClearCasts removes custom casts (tests).
func ClearCasts() {
	customCastMu.Lock()
	defer customCastMu.Unlock()
	customCasts = map[string]CastHandler{}
}

// SetCastEncrypter configures the encrypter used by the "encrypted" cast.
func SetCastEncrypter(crypt CastCrypt) {
	castCrypt = crypt
}

func lookupCustomCast(name string) (CastHandler, bool) {
	customCastMu.RLock()
	defer customCastMu.RUnlock()
	h, ok := customCasts[name]
	return h, ok
}

func castEncryptedIn(value any) (any, error) {
	if castCrypt == nil {
		return fmt.Sprint(value), fmt.Errorf("cast encrypter is not configured")
	}
	plain, err := castCrypt.Decrypt(fmt.Sprint(value))
	if err != nil {
		// value may already be plaintext during tests / first write
		return fmt.Sprint(value), nil
	}
	return plain, nil
}

func castEncryptedOut(value any) any {
	if castCrypt == nil || value == nil {
		return value
	}
	cipher, err := castCrypt.Encrypt(fmt.Sprint(value))
	if err != nil {
		return fmt.Sprint(value)
	}
	return cipher
}
