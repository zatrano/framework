package support_test

import (
	crand "crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/support"
)

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestRandomHex(t *testing.T) {
	s, err := support.RandomHex(16)
	if err != nil {
		t.Fatal(err)
	}
	if len(s) != 32 {
		t.Fatalf("hex length=%d want 32", len(s))
	}
	if _, err := hex.DecodeString(s); err != nil {
		t.Fatal(err)
	}
}

func TestRandomBytesAndBase64(t *testing.T) {
	b, err := support.RandomBytes(8)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 8 {
		t.Fatalf("bytes length=%d want 8", len(b))
	}
	s, err := support.RandomBase64(8)
	if err != nil {
		t.Fatal(err)
	}
	if s == "" {
		t.Fatal("empty base64")
	}
}

func TestMustRandomHex(t *testing.T) {
	s := support.MustRandomHex(8)
	if len(s) != 16 {
		t.Fatalf("hex length=%d want 16", len(s))
	}
}

func TestRandomHexError(t *testing.T) {
	old := crand.Reader
	t.Cleanup(func() { crand.Reader = old })
	crand.Reader = readerFunc(func([]byte) (int, error) {
		return 0, errors.New("entropy exhausted")
	})
	if _, err := support.RandomHex(8); err == nil {
		t.Fatal("expected error")
	}
}

func TestMustRandomHexPanics(t *testing.T) {
	old := crand.Reader
	t.Cleanup(func() { crand.Reader = old })
	crand.Reader = readerFunc(func([]byte) (int, error) {
		return 0, errors.New("entropy exhausted")
	})
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic")
		}
		msg, ok := r.(string)
		if !ok || !strings.Contains(msg, "entropy exhausted") {
			t.Fatalf("panic=%v", r)
		}
	}()
	_ = support.MustRandomHex(8)
}
