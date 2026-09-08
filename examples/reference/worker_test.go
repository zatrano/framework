package reference

import (
	"context"
	"runtime"
	"testing"
	"time"
)

func TestWorkerStartsOnlyOnStart(t *testing.T) {
	app := Assemble(t.TempDir())
	t.Cleanup(func() {
		_ = app.Stop(context.Background())
		closeLog(t, app)
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	w := mustWorker(t, app)
	if w.Running() {
		t.Fatal("running after Bootstrap")
	}
	before := runtime.NumGoroutine()
	if err := app.Start(); err != nil {
		t.Fatal(err)
	}
	waitRunning(t, w)
	if w.Starts() != 1 {
		t.Fatalf("starts=%d", w.Starts())
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if w.Running() {
		t.Fatal("running after Stop")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+4 {
			return
		}
		time.Sleep(20 * time.Millisecond)
		runtime.Gosched()
	}
	t.Fatalf("possible leak: before=%d after=%d", before, runtime.NumGoroutine())
}
