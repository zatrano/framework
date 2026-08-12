package pulse_test

import (
	"testing"
	"time"

	"github.com/zatrano/framework/packages/observability"
	"github.com/zatrano/framework/packages/pulse"
)

func TestPulseSnapshot(t *testing.T) {
	metrics := observability.New()
	metrics.Observe("GET", "/api/health", 200, 5*time.Millisecond)
	d := pulse.New(metrics).WithExtra(func() map[string]any {
		return map[string]any{"custom": 1}
	})
	snap := d.Snapshot()
	if snap["requests"].(int64) != 1 {
		t.Fatalf("snap=%#v", snap)
	}
	if snap["custom"] != 1 {
		t.Fatal("missing extra")
	}
	if snap["uptime"] == nil {
		t.Fatal("missing uptime")
	}
}
