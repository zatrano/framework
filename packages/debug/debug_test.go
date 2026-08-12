package debug_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/debug"
)

func TestDump(t *testing.T) {
	out := debug.Dump(map[string]any{"ok": true, "n": 1})
	if !strings.Contains(out, `"ok"`) || !strings.Contains(out, "true") {
		t.Fatalf("dump=%q", out)
	}
}

func TestDD(t *testing.T) {
	var code int
	debug.SetExitFunc(func(c int) { code = c })
	defer debug.SetExitFunc(nil)
	debug.DD("boom")
	if code != 1 {
		t.Fatalf("exit code=%d", code)
	}
}
