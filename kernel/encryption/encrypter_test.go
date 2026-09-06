package encryption_test

import (
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel/encryption"
)

func TestPasswordIsNotPaddedToAESKey(t *testing.T) {
	_, err := encryption.New("password")
	if err == nil {
		t.Fatal("short APP_KEY was accepted (zero-padded to 32 bytes)")
	}
}

func TestKeyAcceptsAESLengths(t *testing.T) {
	for _, n := range []int{16, 24, 32} {
		enc, err := encryption.New(strings.Repeat("k", n))
		if err != nil {
			t.Fatalf("len %d: %v", n, err)
		}
		ct, err := enc.Encrypt("hello")
		if err != nil {
			t.Fatalf("encrypt len %d: %v", n, err)
		}
		plain, err := enc.Decrypt(ct)
		if err != nil || plain != "hello" {
			t.Fatalf("round-trip len %d: %q %v", n, plain, err)
		}
	}
}

func TestKeyRejectsOtherLengths(t *testing.T) {
	for _, n := range []int{1, 8, 15, 17, 31, 33, 64} {
		if _, err := encryption.New(strings.Repeat("k", n)); err == nil {
			t.Fatalf("len %d was accepted", n)
		}
	}
}

func TestKeyBase64Prefix(t *testing.T) {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	enc, err := encryption.New("base64:" + base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		t.Fatal(err)
	}
	ct, err := enc.Encrypt("x")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := enc.Decrypt(ct); err != nil || got != "x" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := encryption.New("base64:not-valid-base64!!!"); err == nil {
		t.Fatal("invalid base64 prefix must reject, not pad the suffix")
	}
}

func TestKeyHexExactLengths(t *testing.T) {
	raw := strings.Repeat("ab", 16) // 32 hex chars → 16 bytes
	if _, err := encryption.New(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := encryption.New(hex.EncodeToString(make([]byte, 32))); err != nil {
		t.Fatal(err)
	}
	if _, err := encryption.New("aabb"); err == nil {
		t.Fatal("2-byte hex must reject")
	}
}

func TestEmptyKeyRejected(t *testing.T) {
	if _, err := encryption.New(""); err == nil {
		t.Fatal("empty key")
	}
	if _, err := encryption.New("   "); err == nil {
		t.Fatal("whitespace key")
	}
}

func TestLocalDevKeyIsAES256Length(t *testing.T) {
	if len(encryption.LocalDevKey) != 32 {
		t.Fatalf("len=%d", len(encryption.LocalDevKey))
	}
	if _, err := encryption.New(encryption.LocalDevKey); err != nil {
		t.Fatal(err)
	}
}
