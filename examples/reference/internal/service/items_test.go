package service

import (
	"errors"
	"testing"

	"github.com/zatrano/framework/v2/examples/reference/internal/domain"
	"github.com/zatrano/framework/v2/examples/reference/internal/repository"
)

func TestGetValidatesID(t *testing.T) {
	s := New(repository.NewMemory())
	_, err := s.Get(" ")
	if !errors.Is(err, domain.ErrInvalidRequest) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	s := New(repository.NewMemory())
	_, err := s.Get("missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestGetReturnsSeededItem(t *testing.T) {
	mem := repository.NewMemory()
	mem.Seed(domain.Item{ID: "1", Name: "alpha"})
	s := New(mem)
	item, err := s.Get("1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "alpha" {
		t.Fatalf("name=%q", item.Name)
	}
}

func TestGetPreservesUnavailableCause(t *testing.T) {
	cause := errors.New("backend down")
	s := New(&repository.Unavailable{Cause: cause})
	_, err := s.Get("1")
	if !errors.Is(err, domain.ErrUnavailable) {
		t.Fatalf("sentinel: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("cause lost: %v", err)
	}
}
