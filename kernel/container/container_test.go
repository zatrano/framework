package container_test

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/contracts"
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

func TestConcurrentCircularSingletons(t *testing.T) {
	c := container.New()
	var started sync.WaitGroup
	started.Add(2)
	ready := make(chan struct{})

	c.Singleton("a", func(c *container.Container) (any, error) {
		started.Done()
		<-ready
		return c.Make("b")
	})
	c.Singleton("b", func(c *container.Container) (any, error) {
		started.Done()
		<-ready
		return c.Make("a")
	})

	errCh := make(chan error, 2)
	go func() { _, err := c.Make("a"); errCh <- err }()
	go func() { _, err := c.Make("b"); errCh <- err }()

	done := make(chan struct{})
	go func() {
		started.Wait()
		close(ready)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("factories never started")
	}

	var errA, errB error
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			if errA == nil {
				errA = err
			} else {
				errB = err
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cross-goroutine circular Make deadlocked")
		}
	}
	if errA == nil && errB == nil {
		t.Fatal("expected circular dependency error")
	}
	joined := ""
	sawCircular := false
	for _, err := range []error{errA, errB} {
		if err == nil {
			continue
		}
		joined += err.Error()
		var re *container.ResolutionError
		if errors.As(err, &re) && re.Kind == container.KindCircular {
			sawCircular = true
		}
	}
	if !strings.Contains(joined, "circular") {
		t.Fatalf("errA=%v errB=%v", errA, errB)
	}
	if !sawCircular {
		t.Fatalf("expected KindCircular, errA=%v errB=%v", errA, errB)
	}
}

func TestContainerFreezeAllowsLazyMake(t *testing.T) {
	c := container.New()
	c.Singleton("svc", func() any { return "ok" })
	c.Freeze()
	if !c.Frozen() {
		t.Fatal("expected frozen")
	}
	if c.MustMake("svc") != "ok" {
		t.Fatal("Make after freeze")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected bind panic")
		}
	}()
	c.Bind("late", func() any { return 1 })
}

func TestTransientCircularDependency(t *testing.T) {
	c := container.New()
	c.Bind("a", func(c *container.Container) (any, error) {
		return c.Make("b")
	})
	c.Bind("b", func(c *container.Container) (any, error) {
		return c.Make("a")
	})
	_, err := c.Make("a")
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Fatalf("err=%v", err)
	}
	var re *container.ResolutionError
	if !errors.As(err, &re) || re.Kind != container.KindCircular {
		t.Fatalf("kind: %v", err)
	}
	if re.Kind == container.KindResolveFailed {
		t.Fatal("cycle must not be resolve_failed")
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
	var re *container.ResolutionError
	if !errors.As(err, &re) || re.Kind != container.KindCircular {
		t.Fatalf("kind: %v", err)
	}
	if re.Kind == container.KindResolveFailed {
		t.Fatal("cycle must not be resolve_failed")
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
	cause := errors.New("nope")
	c.Bind("fail", func() (string, error) { return "", cause })
	_, err := c.Make("fail")
	// Error() text is not API; unwrap is.
	if !errors.Is(err, cause) {
		t.Fatalf("errors.Is: err=%v", err)
	}
	var re *container.ResolutionError
	if !errors.As(err, &re) {
		t.Fatalf("expected ResolutionError, err=%v", err)
	}
	if re.Kind != container.KindResolveFailed || re.Target != "fail" {
		t.Fatalf("kind=%s target=%q", re.Kind, re.Target)
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

func TestContainerContractsMakeAndBound(t *testing.T) {
	c := container.New()
	var ct contracts.Container = c
	ct.Instance("svc", "ok")
	if !ct.Bound("svc") {
		t.Fatal("expected Bound")
	}
	got, err := ct.Make("svc")
	if err != nil || got != "ok" {
		t.Fatalf("got=%#v err=%v", got, err)
	}
}

func TestSingletonPanicAllowsRetry(t *testing.T) {
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
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestSingletonPanicUnblocksWaiters(t *testing.T) {
	c := container.New()
	started := make(chan struct{})
	release := make(chan struct{})
	var builds atomic.Int32
	c.Singleton("svc", func() any {
		if builds.Add(1) == 1 {
			close(started)
			<-release
			panic("boom")
		}
		return "ok"
	})

	panicCh := make(chan any, 1)
	go func() {
		defer func() { panicCh <- recover() }()
		_, _ = c.Make("svc")
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("builder never started")
	}

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := c.Make("svc")
			if err != nil || got != "ok" {
				t.Errorf("got=%#v err=%v", got, err)
			}
		}()
	}
	time.Sleep(30 * time.Millisecond)
	close(release)

	select {
	case rec := <-panicCh:
		if rec == nil {
			t.Fatal("builder panic was swallowed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("builder did not panic")
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waiters deadlocked after singleton panic")
	}
}
