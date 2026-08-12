package core

import "testing"

func TestServiceOfMissingReturnsNil(t *testing.T) {
	app := NewApplication(".")
	if got := serviceOf[*struct{}](app, "missing"); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestMiddlewareFromMissing(t *testing.T) {
	app := NewApplication(".")
	if middlewareFrom(app, "octane") != nil {
		t.Fatal("expected nil middleware when addon not registered")
	}
}
