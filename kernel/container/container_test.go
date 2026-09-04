package container_test

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/kernel/container"
)

func TestMakeNestedFactoryDoesNotDeadlock(t *testing.T) {
	c := container.New()
	c.Singleton("log", func(c *container.Container) any {
		return "logger"
	})
	c.Singleton("database", func(c *container.Container) any {
		log, err := c.Make("log")
		if err != nil {
			t.Errorf("nested Make: %v", err)
			return nil
		}
		return "db:" + log.(string)
	})

	done := make(chan struct{})
	var got any
	var err error
	go func() {
		got, err = c.Make("database")
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Make deadlocked while resolving nested factory")
	}
	if err != nil {
		t.Fatal(err)
	}
	if got != "db:logger" {
		t.Fatalf("got %#v", got)
	}
}

func TestConcurrentSingletonBuildsOnce(t *testing.T) {
	c := container.New()
	var builds atomic.Int32
	c.Singleton("svc", func(c *container.Container) any {
		builds.Add(1)
		time.Sleep(30 * time.Millisecond)
		return "ok"
	})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if c.MustMake("svc") != "ok" {
				t.Error("unexpected value")
			}
		}()
	}
	wg.Wait()
	if builds.Load() != 1 {
		t.Fatalf("builds=%d, want 1", builds.Load())
	}
}

func TestCircularDependency(t *testing.T) {
	c := container.New()
	c.Singleton("a", func(c *container.Container) (any, error) {
		return c.Make("b")
	})
	c.Singleton("b", func(c *container.Container) (any, error) {
		return c.Make("a")
	})
	_, err := c.Make("a")
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Fatalf("err=%v", err)
	}
}

func TestAliasChainAndCycle(t *testing.T) {
	c := container.New()
	c.Instance("service", "value")
	c.Alias("service", "bar")
	c.Alias("bar", "foo")
	got, err := c.Make("foo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "value" {
		t.Fatalf("got %#v", got)
	}

	c.Alias("ping", "pong")
	c.Alias("pong", "ping")
	if _, err := c.Make("ping"); err == nil {
		t.Fatal("expected alias cycle")
	}
}

func TestInvalidFactorySignatures(t *testing.T) {
	c := container.New()
	c.Bind("args", func(x int) any { return x })
	if _, err := c.Make("args"); err == nil {
		t.Fatal("expected error for parameterized factory")
	}
	c.Bind("badpair", func() (string, int) { return "x", 1 })
	if _, err := c.Make("badpair"); err == nil {
		t.Fatal("expected error for non-error second return")
	}
}

func TestTypedFactoryError(t *testing.T) {
	c := container.New()
	c.Bind("fail", func() (string, error) { return "", errors.New("nope") })
	_, err := c.Make("fail")
	if err == nil || err.Error() != "nope" {
		t.Fatalf("err=%v", err)
	}
	c.Bind("ok", func() (string, error) { return "yes", nil })
	got, err := c.Make("ok")
	if err != nil || got != "yes" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestBindFactoryWithContainerError(t *testing.T) {
	c := container.New()
	c.Singleton("svc", func(c *container.Container) (any, error) {
		return "ok", nil
	})
	got, err := c.Make("svc")
	if err != nil || got != "ok" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestMustMakePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	container.New().MustMake("missing")
}

func TestTransientRebuilds(t *testing.T) {
	c := container.New()
	var n int
	c.Bind("n", func() any {
		n++
		return n
	})
	if c.MustMake("n") != 1 || c.MustMake("n") != 2 {
		t.Fatalf("n=%d", n)
	}
}

func TestBound(t *testing.T) {
	c := container.New()
	if c.Bound("x") {
		t.Fatal("empty")
	}
	c.Instance("x", 1)
	c.Alias("x", "y")
	if !c.Bound("x") || !c.Bound("y") {
		t.Fatal("expected bound")
	}
}
