package worker

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestTickerStartsOnceAndStops(t *testing.T) {
	w := &Ticker{}
	if w.Running() {
		t.Fatal("must not run before Start")
	}
	if err := w.Start(20 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := w.Start(20 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if w.Starts() != 1 {
		t.Fatalf("starts=%d", w.Starts())
	}
	deadline := time.Now().Add(time.Second)
	for !w.Running() && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if !w.Running() {
		t.Fatal("expected running")
	}
	if err := w.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w.Running() {
		t.Fatal("expected stopped")
	}
	after := w.Ticks()
	time.Sleep(60 * time.Millisecond)
	if w.Ticks() != after {
		t.Fatalf("ticks after stop: before=%d after=%d", after, w.Ticks())
	}
}

func TestTickerStopIsSafeAndIdempotent(t *testing.T) {
	w := &Ticker{}
	if err := w.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := w.Start(10 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := w.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := w.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w.Running() {
		t.Fatal("still running")
	}
}

func TestTickerRespectsCancellation(t *testing.T) {
	w := &Ticker{}
	if err := w.Start(5 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := w.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if w.Running() {
		t.Fatal("running after cancel")
	}
}

func TestTickerDoesNotLeakGoroutine(t *testing.T) {
	before := runtime.NumGoroutine()
	w := &Ticker{}
	if err := w.Start(5 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := w.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
		runtime.Gosched()
	}
	t.Fatalf("goroutines before=%d after=%d", before, runtime.NumGoroutine())
}
