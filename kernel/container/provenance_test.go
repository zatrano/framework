package container_test

import (
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/kernel/container"
)

func resolutionErr(t *testing.T, err error) *container.ResolutionError {
	t.Helper()
	var re *container.ResolutionError
	if !errors.As(err, &re) {
		t.Fatalf("errors.As ResolutionError: err=%v", err)
	}
	return re
}

func TestTC01NestedFactoryErrorPath(t *testing.T) {
	c := container.New()
	cause := errors.New("database unavailable")
	c.Bind("C", func() (any, error) { return nil, cause })
	c.Bind("B", func(c *container.Container) (any, error) { return c.Make("C") })
	c.Bind("A", func(c *container.Container) (any, error) { return c.Make("B") })

	_, err := c.Make("A")
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is: err=%v", err)
	}
	re := resolutionErr(t, err)
	if re.Kind != container.KindResolveFailed {
		t.Fatalf("kind=%s", re.Kind)
	}
	if re.Target != "A" {
		t.Fatalf("target=%q", re.Target)
	}
	got := strings.Join(re.Path, " -> ")
	if got != "A -> B -> C" {
		t.Fatalf("path=%q", got)
	}
}

func TestTC02AliasPathPreserved(t *testing.T) {
	c := container.New()
	cause := errors.New("connection refused")
	c.Bind("database", func() (any, error) { return nil, cause })
	c.Alias("database", "bar")
	c.Alias("bar", "foo")

	_, err := c.Make("foo")
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is: err=%v", err)
	}
	re := resolutionErr(t, err)
	if re.Target != "foo" {
		t.Fatalf("target=%q", re.Target)
	}
	got := strings.Join(re.Path, " -> ")
	if got != "foo -> bar -> database" {
		t.Fatalf("path=%q", got)
	}
}

func TestTC03ErrorsIsPreserved(t *testing.T) {
	c := container.New()
	cause := errors.New("nope")
	c.Bind("leaf", func() (any, error) { return nil, cause })
	c.Bind("mid", func(c *container.Container) (any, error) { return c.Make("leaf") })
	_, err := c.Make("mid")
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is: err=%v", err)
	}
}

func TestTC04ErrorsAsResolutionError(t *testing.T) {
	c := container.New()
	cause := errors.New("boom")
	c.Bind("svc", func() (any, error) { return nil, cause })
	_, err := c.Make("svc")
	re := resolutionErr(t, err)
	if re.Kind != container.KindResolveFailed || re.Target != "svc" || re.Cause != cause {
		t.Fatalf("%+v", re)
	}
}

func TestTC05CycleKindNotResolveFailed(t *testing.T) {
	c := container.New()
	c.Bind("a", func(c *container.Container) (any, error) { return c.Make("b") })
	c.Bind("b", func(c *container.Container) (any, error) { return c.Make("a") })
	_, err := c.Make("a")
	re := resolutionErr(t, err)
	if re.Kind != container.KindCircular {
		t.Fatalf("kind=%s want circular", re.Kind)
	}
	if re.Kind == container.KindResolveFailed {
		t.Fatal("cycle must not be resolve_failed")
	}
	if !strings.Contains(strings.Join(re.Path, " -> "), "a") || !strings.Contains(strings.Join(re.Path, " -> "), "b") {
		t.Fatalf("path=%v", re.Path)
	}
}

func TestTC06CrossGoroutineCycleKind(t *testing.T) {
	c := container.New()
	started := make(chan struct{})
	var n atomic.Int32
	release := make(chan struct{})
	c.Singleton("a", func(c *container.Container) (any, error) {
		if n.Add(1) == 2 {
			close(started)
		}
		<-release
		return c.Make("b")
	})
	c.Singleton("b", func(c *container.Container) (any, error) {
		if n.Add(1) == 2 {
			close(started)
		}
		<-release
		return c.Make("a")
	})

	errCh := make(chan error, 2)
	go func() { _, err := c.Make("a"); errCh <- err }()
	go func() { _, err := c.Make("b"); errCh <- err }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("factories never started")
	}
	close(release)

	var sawCircular bool
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			var re *container.ResolutionError
			if err != nil && errors.As(err, &re) && re.Kind == container.KindCircular {
				sawCircular = true
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cross-goroutine circular Make deadlocked")
		}
	}
	if !sawCircular {
		t.Fatal("expected KindCircular")
	}
}

func TestTC07SingletonFailureRetry(t *testing.T) {
	c := container.New()
	cause := errors.New("fail")
	var calls atomic.Int32
	c.Singleton("svc", func() (any, error) {
		if calls.Add(1) == 1 {
			return nil, cause
		}
		return "ok", nil
	})
	_, err := c.Make("svc")
	if !errors.Is(err, cause) {
		t.Fatalf("first: %v", err)
	}
	got, err := c.Make("svc")
	if err != nil || got != "ok" {
		t.Fatalf("retry got=%#v err=%v", got, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestTC08PanicCleanupRetry(t *testing.T) {
	c := container.New()
	var calls atomic.Int32
	c.Singleton("svc", func() any {
		if calls.Add(1) == 1 {
			panic("boom")
		}
		return "ok"
	})
	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		_, _ = c.Make("svc")
	}()
	if !panicked {
		t.Fatal("factory panic must propagate")
	}
	done := make(chan struct{})
	var got any
	var err error
	go func() {
		got, err = c.Make("svc")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Make after singleton panic deadlocked")
	}
	if err != nil || got != "ok" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestAliasNestedFactoryPath(t *testing.T) {
	c := container.New()
	cause := errors.New("down")
	c.Bind("database", func() (any, error) { return nil, cause })
	c.Alias("database", "bar")
	c.Bind("B", func(c *container.Container) (any, error) { return c.Make("bar") })
	c.Alias("B", "A")

	_, err := c.Make("A")
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is: err=%v", err)
	}
	re := resolutionErr(t, err)
	if re.Target != "A" {
		t.Fatalf("target=%q", re.Target)
	}
	got := strings.Join(re.Path, " -> ")
	if got != "A -> B -> bar -> database" {
		t.Fatalf("path=%q", got)
	}
}
