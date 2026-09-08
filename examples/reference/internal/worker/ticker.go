package worker

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Ticker is a single background loop. It is started from LifecycleProvider.Start
// and stopped from Stop. It must not run during Bootstrap.
type Ticker struct {
	mu      sync.Mutex
	cancel  context.CancelFunc
	done    chan struct{}
	ticks   atomic.Int64
	starts  atomic.Int32
	running atomic.Bool
}

// Start launches the loop once. A second call is a no-op.
func (t *Ticker) Start(interval time.Duration) error {
	if t == nil {
		return nil
	}
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.done = make(chan struct{})
	t.starts.Add(1)
	t.running.Store(true)
	go t.loop(ctx, interval)
	return nil
}

func (t *Ticker) loop(ctx context.Context, interval time.Duration) {
	defer close(t.done)
	defer t.running.Store(false)
	tick := time.NewTicker(interval)
	defer tick.Stop()
	t.ticks.Add(1)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			t.ticks.Add(1)
		}
	}
}

// Stop cancels the loop and waits for it to exit. It is safe to call when
// the ticker was never started or after it already stopped.
func (t *Ticker) Stop(ctx context.Context) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	cancel := t.cancel
	done := t.done
	t.cancel = nil
	t.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		select {
		case <-done:
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return ctx.Err()
		}
	}
}

// Running reports whether the loop is active.
func (t *Ticker) Running() bool {
	if t == nil {
		return false
	}
	return t.running.Load()
}

// Starts returns how many times Start launched a goroutine.
func (t *Ticker) Starts() int {
	if t == nil {
		return 0
	}
	return int(t.starts.Load())
}

// Ticks returns how many loop iterations have run.
func (t *Ticker) Ticks() int64 {
	if t == nil {
		return 0
	}
	return t.ticks.Load()
}
