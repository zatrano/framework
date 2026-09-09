package console

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestCLIHelpAndVersionFlags(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	cli := New(app)

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cli.Run([]string{"--version"})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), currentRelease) {
		t.Fatalf("version=%q", buf.String())
	}

	r, w, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cli.Run([]string{"--help"})
	_ = w.Close()
	os.Stdout = old
	buf.Reset()
	_, _ = buf.ReadFrom(r)
	if err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	if !strings.Contains(text, "package:doctor") || !strings.Contains(text, "new") || !strings.Contains(text, "serve") {
		t.Fatalf("help missing commands:\n%s", text)
	}
}

func TestCLIListIsSorted(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	cli := New(app)
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	if err := cli.Run(nil); err != nil {
		_ = w.Close()
		os.Stdout = old
		t.Fatal(err)
	}
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	lines := strings.Split(buf.String(), "\n")
	var names []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "ZATRANO") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		names = append(names, fields[0])
	}
	if len(names) < 4 {
		t.Fatalf("expected command list, got %#v from %q", names, buf.String())
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("list must be sorted: %q then %q", names[i-1], names[i])
		}
	}
}

func TestCLIUnknownCommandIsActionable(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	err := New(app).Run([]string{"does-not-exist"})
	if err == nil {
		t.Fatal("expected unknown command")
	}
	msg := err.Error()
	if !strings.Contains(msg, "does-not-exist") || !strings.Contains(msg, "Next:") {
		t.Fatalf("unknown command must name the command and next step: %v", err)
	}
}

func TestServeInvalidPortIsUsage(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	err := (&ServeCommand{app: app}).Handle([]string{"--port", "abc"})
	if err == nil {
		t.Fatal("expected usage error")
	}
	if CodeFromError(err) != ExitUsage {
		t.Fatalf("exit=%d want %d (%v)", CodeFromError(err), ExitUsage, err)
	}
	if !strings.Contains(err.Error(), "port") {
		t.Fatalf("must mention port: %v", err)
	}
}
