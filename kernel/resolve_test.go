package kernel

import "testing"

func TestServiceOfMissingReturnsNil(t *testing.T) {
	app := NewApplication(".")
	if got := Resolve[*struct{}](app, "missing"); got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestResolveOKAndMustResolve(t *testing.T) {
	app := NewApplication(".")
	app.Container().Instance("svc", "hello")
	got, err := ResolveOK[string](app, "svc")
	if err != nil || got != "hello" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if _, err := ResolveOK[string](app, "missing"); err == nil {
		t.Fatal("expected error")
	}
	if MustResolve[string](app, "svc") != "hello" {
		t.Fatal("MustResolve")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustResolve[int](app, "svc")
}

func TestMiddlewareFromMissing(t *testing.T) {
	app := NewApplication(".")
	if middlewareFrom(app, "octane") != nil {
		t.Fatal("expected nil middleware when addon not registered")
	}
}
