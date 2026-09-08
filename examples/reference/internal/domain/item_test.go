package domain

import (
	"errors"
	"testing"
)

func TestParseIDRejectsBlank(t *testing.T) {
	if _, err := ParseID("  "); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("blank id: %v", err)
	}
}

func TestParseIDTrims(t *testing.T) {
	id, err := ParseID("  alpha  ")
	if err != nil {
		t.Fatal(err)
	}
	if id != "alpha" {
		t.Fatalf("id=%q", id)
	}
}

func TestSentinelsAreDistinct(t *testing.T) {
	if errors.Is(ErrNotFound, ErrInvalidRequest) || errors.Is(ErrUnavailable, ErrConfiguration) {
		t.Fatal("domain sentinels must stay distinguishable")
	}
}
