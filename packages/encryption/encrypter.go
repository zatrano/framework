package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

// Encrypter encrypts and decrypts string payloads.
type Encrypter struct {
	key []byte
}

// New creates an encrypter from an APP_KEY style string.
func New(appKey string) (*Encrypter, error) {
	key, err := parseKey(appKey)
	if err != nil {
		return nil, err
	}
	return &Encrypter{key: key}, nil
}

// Encrypt encrypts a string payload.
func (e *Encrypter) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a string payload.
func (e *Encrypter) Decrypt(payload string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// EncryptString is an alias for Encrypt.
func (e *Encrypter) EncryptString(plaintext string) (string, error) {
	return e.Encrypt(plaintext)
}

// DecryptString is an alias for Decrypt.
func (e *Encrypter) DecryptString(payload string) (string, error) {
	return e.Decrypt(payload)
}

// MustEncrypt encrypts or panics.
func (e *Encrypter) MustEncrypt(plaintext string) string {
	out, err := e.Encrypt(plaintext)
	if err != nil {
		panic(err)
	}
	return out
}

// MustDecrypt decrypts or panics.
func (e *Encrypter) MustDecrypt(payload string) string {
	out, err := e.Decrypt(payload)
	if err != nil {
		panic(err)
	}
	return out
}

// EncryptJSON marshals and encrypts a value.
func (e *Encrypter) EncryptJSON(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return e.Encrypt(string(raw))
}

// DecryptJSON decrypts and unmarshals into dest.
func (e *Encrypter) DecryptJSON(payload string, dest any) error {
	plain, err := e.Decrypt(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(plain), dest)
}

func parseKey(appKey string) ([]byte, error) {
	appKey = strings.TrimSpace(appKey)
	if appKey == "" {
		return nil, errors.New("APP_KEY is empty")
	}
	if strings.HasPrefix(appKey, "base64:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(appKey, "base64:"))
		if err != nil {
			return normalizeKey([]byte(strings.TrimPrefix(appKey, "base64:"))), nil
		}
		return normalizeKey(decoded), nil
	}
	if decoded, err := hex.DecodeString(appKey); err == nil && len(decoded) >= 16 {
		return normalizeKey(decoded), nil
	}
	return normalizeKey([]byte(appKey)), nil
}

func normalizeKey(key []byte) []byte {
	out := make([]byte, 32)
	copy(out, key)
	return out
}
