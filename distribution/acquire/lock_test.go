package acquire

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type holdRunner struct {
	started chan string
	release <-chan struct{}
	active  atomic.Int32
	max     atomic.Int32
}

func (h *holdRunner) Run(ctx context.Context, inv Invocation) (InvocationResult, error) {
	n := h.active.Add(1)
	for {
		old := h.max.Load()
		if n <= old || h.max.CompareAndSwap(old, n) {
			break
		}
	}
	select {
	case h.started <- inv.Dir:
	case <-ctx.Done():
		h.active.Add(-1)
		return InvocationResult{Invocation: inv}, ctx.Err()
	}
	select {
	case <-h.release:
	case <-ctx.Done():
		h.active.Add(-1)
		return InvocationResult{Invocation: inv}, ctx.Err()
	}
	h.active.Add(-1)
	return InvocationResult{Invocation: inv}, nil
}

func TestExecuteSerializesSameModuleRoot(t *testing.T) {
	root := t.TempDir()
	release := make(chan struct{})
	started := make(chan string, 2)
	h := &holdRunner{started: started, release: release}
	errc := make(chan error, 2)
	go func() {
		_, err := execute(context.Background(), h, Request{Root: root, GoGetArg: "example.com/a@v1.0.0"})
		errc <- err
	}()
	go func() {
		_, err := execute(context.Background(), h, Request{Root: root, GoGetArg: "example.com/b@v1.0.0"})
		errc <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first mutation never entered the runner")
	}
	select {
	case d := <-started:
		close(release)
		t.Fatalf("second mutation overlapped on %q", d)
	case <-time.After(75 * time.Millisecond):
	}
	close(release)
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("second mutation did not run after the first released")
	}
	for i := 0; i < 2; i++ {
		if err := <-errc; err != nil {
			t.Fatal(err)
		}
	}
	if h.max.Load() != 1 {
		t.Fatalf("same-root overlap=%d", h.max.Load())
	}
}

func TestExecuteAllowsDifferentModuleRoots(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	release := make(chan struct{})
	started := make(chan string, 2)
	h := &holdRunner{started: started, release: release}
	errc := make(chan error, 2)
	go func() {
		_, err := execute(context.Background(), h, Request{Root: a, GoGetArg: "example.com/a@v1.0.0"})
		errc <- err
	}()
	go func() {
		_, err := execute(context.Background(), h, Request{Root: b, GoGetArg: "example.com/b@v1.0.0"})
		errc <- err
	}()
	seen := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(seen) < 2 {
		select {
		case dir := <-started:
			seen[dir] = true
		case <-deadline:
			t.Fatalf("different roots did not proceed independently: %v", seen)
		}
	}
	if h.max.Load() < 2 {
		t.Fatalf("expected overlapping Run on distinct roots, max=%d", h.max.Load())
	}
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-errc; err != nil {
			t.Fatal(err)
		}
	}
}

func TestInspectDoesNotTakeMutationLock(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n\ngo 1.25.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	release := make(chan struct{})
	started := make(chan string, 1)
	h := &holdRunner{started: started, release: release}
	errc := make(chan error, 1)
	go func() {
		_, err := execute(context.Background(), h, Request{Root: root, GoGetArg: "example.com/a@v1.0.0"})
		errc <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("mutation never started")
	}
	done := make(chan error, 1)
	go func() {
		_, err := Inspect(root)
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("Inspect blocked on the mutation lock")
	}
	close(release)
	if err := <-errc; err != nil {
		t.Fatal(err)
	}
}

func TestLockMutationHonorsCancellation(t *testing.T) {
	root := t.TempDir()
	release := make(chan struct{})
	started := make(chan string, 1)
	h := &holdRunner{started: started, release: release}
	first := make(chan error, 1)
	go func() {
		_, err := execute(context.Background(), h, Request{Root: root, GoGetArg: "example.com/a@v1.0.0"})
		first <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first mutation never started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := execute(ctx, h, Request{Root: root, GoGetArg: "example.com/b@v1.0.0"})
	if err == nil {
		close(release)
		t.Fatal("waiting mutation must observe cancellation")
	}
	close(release)
	if e := <-first; e != nil {
		t.Fatal(e)
	}
}
