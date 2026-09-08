package repository

import (
	"errors"
	"testing"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
)

func TestMemoryGetAndList(t *testing.T) {
	m := NewMemory()
	m.Seed(domain.Item{ID: "1", Name: "alpha"})
	got, err := m.Get("1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "alpha" {
		t.Fatalf("name=%q", got.Name)
	}
	if len(m.List()) != 1 {
		t.Fatalf("list=%d", len(m.List()))
	}
}

func TestMemoryNotFound(t *testing.T) {
	m := NewMemory()
	_, err := m.Get("missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestUnavailableWrapsCause(t *testing.T) {
	cause := errors.New("disk full: token=super-secret-value-xyz")
	u := &Unavailable{Cause: cause}
	_, err := u.Get("1")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("sentinel: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("cause lost: %v", err)
	}
}
