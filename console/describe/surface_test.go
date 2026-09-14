package describe

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
)

func TestDescribeHelpAndText(t *testing.T) {
	var buf bytes.Buffer
	cmd := &DescribeCommand{out: &buf}
	if err := cmd.Handle([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Fatal(buf.String())
	}
	buf.Reset()
	if err := cmd.Handle(nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "ZATRANO describe") && !strings.Contains(buf.String(), "contracts") {
		t.Fatalf("text empty: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "warning: packages ai, rag, and agent are experimental") {
		t.Fatalf("describe must warn that ai/rag/agent are experimental:\n%s", buf.String())
	}
	if err := cmd.Handle([]string{"--format=xml"}); err == nil {
		t.Fatal("bad format")
	}
	if cmd.Name() != "describe" || cmd.Description() == "" {
		t.Fatal("identity")
	}
}

func TestCatalogExportsAndAgentsMarkdown(t *testing.T) {
	if _, ok := Lookup("session"); !ok {
		t.Fatal("session")
	}
	if len(Ecosystem()) == 0 {
		t.Fatal("ecosystem")
	}
	if len(ByLayer(kernel.LayerAddon)) == 0 {
		t.Fatal("addon layer")
	}
	libs := Libraries()
	found := false
	for _, p := range libs {
		if p.Name == "redisx" {
			found = true
		}
	}
	if !found {
		t.Fatal("redisx library")
	}
	dir := t.TempDir()
	path, err := WriteAgentsMarkdown(dir)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "Application engineering") {
		t.Fatalf("%s", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
}
