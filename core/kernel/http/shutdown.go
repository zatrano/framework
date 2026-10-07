package http

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type shutdownHook struct {
	name string
	fn   func(context.Context) error
}

var (
	shutdownMu      sync.Mutex
	shutdownHooks   []shutdownHook
	shutdownRunning bool
	shutdownDone    bool
)

// RegisterShutdownHook registers fn under name. RunListen runs every hook
// before rawhttp Server.Shutdown. Hooks for one shutdown run concurrently.
// Each panic is recovered. A returned error is reported to the caller and does
// not stop the shutdown. The context they receive ends at min(5s, one third of
// the time left on the shutdown context).
//
// A second RunShutdownHooks call for the same shutdown is a no-op, and a hook
// registered after that shutdown has started is ignored. Registering the same
// name again before shutdown replaces the function. Hook functions should
// no-op if they are invoked twice.
func RegisterShutdownHook(name string, fn func(context.Context) error) {
	if name == "" || fn == nil {
		return
	}
	shutdownMu.Lock()
	defer shutdownMu.Unlock()
	if shutdownRunning || shutdownDone {
		return
	}
	for i := range shutdownHooks {
		if shutdownHooks[i].name == name {
			shutdownHooks[i].fn = fn
			return
		}
	}
	shutdownHooks = append(shutdownHooks, shutdownHook{name: name, fn: fn})
}

// RunShutdownHooks runs the registered hooks once. The second call returns nil
// and does not invoke them. The caller logs the errors and continues.
func RunShutdownHooks(parent context.Context) []error {
	shutdownMu.Lock()
	if shutdownDone {
		shutdownMu.Unlock()
		return nil
	}
	shutdownRunning = true
	hooks := append([]shutdownHook(nil), shutdownHooks...)
	shutdownMu.Unlock()

	if parent == nil {
		parent = context.Background()
	}
	budget := hookBudget(parent)
	var hctx context.Context
	var cancel context.CancelFunc
	if budget <= 0 {
		hctx, cancel = context.WithCancel(parent)
		cancel()
	} else {
		hctx, cancel = context.WithTimeout(parent, budget)
	}
	defer cancel()

	var mu sync.Mutex
	var out []error
	add := func(err error) {
		if err == nil {
			return
		}
		mu.Lock()
		out = append(out, err)
		mu.Unlock()
	}
	var wg sync.WaitGroup
	for _, h := range hooks {
		wg.Add(1)
		go func(h shutdownHook) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					add(fmt.Errorf("shutdown hook %s: panic: %v", h.name, r))
				}
			}()
			if err := h.fn(hctx); err != nil {
				add(fmt.Errorf("shutdown hook %s: %w", h.name, err))
			}
		}(h)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-hctx.Done():
		select {
		case <-done:
		case <-time.After(50 * time.Millisecond):
		}
	}
	shutdownMu.Lock()
	shutdownDone = true
	shutdownRunning = false
	shutdownMu.Unlock()
	mu.Lock()
	defer mu.Unlock()
	return append([]error(nil), out...)
}

func hookBudget(parent context.Context) time.Duration {
	const cap = 5 * time.Second
	if parent == nil {
		return cap
	}
	deadline, ok := parent.Deadline()
	if !ok {
		return cap
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return 0
	}
	third := remaining / 3
	if third > cap {
		return cap
	}
	return third
}

// ResetShutdownState clears the one-shot latch and the hook list so another
// test can shut down. Outside a test it does nothing.
func ResetShutdownState() {
	if !testing.Testing() {
		return
	}
	shutdownMu.Lock()
	shutdownHooks = nil
	shutdownRunning = false
	shutdownDone = false
	shutdownMu.Unlock()
}
