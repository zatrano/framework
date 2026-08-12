package schedule_test

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/schedule"
)

func TestWithoutOverlapping(t *testing.T) {
	s := schedule.New()
	dir := t.TempDir()
	s.SetMutexPath(dir)

	var runs atomic.Int32
	block := make(chan struct{})

	s.Command("long", func() error {
		runs.Add(1)
		<-block
		return nil
	}).EveryMinute().WithoutOverlapping()

	now := time.Now()
	done := make(chan struct{})
	go func() {
		_ = s.RunDue(now)
		close(done)
	}()

	// Wait until lock file appears.
	deadline := time.Now().Add(2 * time.Second)
	for {
		entries, _ := os.ReadDir(dir)
		if len(entries) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected lock file during run")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Second invocation should skip while lock is held.
	_ = s.RunDue(now)
	close(block)
	<-done

	if runs.Load() != 1 {
		t.Fatalf("expected exactly 1 run while overlapped, got %d", runs.Load())
	}

	// After unlock, another run should succeed.
	_ = s.RunDue(now)
	if runs.Load() != 2 {
		t.Fatalf("expected 2 runs after unlock, got %d", runs.Load())
	}
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { return nil })
}
