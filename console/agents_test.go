package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderAgentsIncludesDescribeContracts(t *testing.T) {
	doc, err := BuildDescribeDocument("")
	if err != nil {
		t.Fatal(err)
	}
	doc.Contracts["ExtraSurface"] = ContractType{
		Name: "ExtraSurface",
		File: "contracts/extra.go",
		Methods: []ContractMethod{
			{Name: "Ping", Signature: "Ping() string"},
		},
	}
	doc.Routing.Primitives = append(doc.Routing.Primitives, ContractMethod{
		Name:      "RegisterExtra",
		Signature: "RegisterExtra(fn func(*Router))",
	})
	md := RenderAgentsMarkdown(doc)
	if !strings.Contains(md, "ExtraSurface") || !strings.Contains(md, "Ping() string") {
		t.Fatalf("new contract missing from AGENTS.md:\n%s", md)
	}
	if !strings.Contains(md, "RegisterExtra(fn func(*Router))") {
		t.Fatalf("new routing primitive missing from AGENTS.md:\n%s", md)
	}
	if !strings.Contains(md, "`Router`") {
		t.Fatalf("Router missing:\n%s", md)
	}
	if !strings.Contains(md, "zatrano doctor") {
		t.Fatal("doctor mention missing")
	}
	for _, check := range []string{"routes", "concrete", "layout", "providers"} {
		if !strings.Contains(md, "`"+check+"`") {
			t.Fatalf("doctor check %s missing", check)
		}
	}
}

func TestAgentsGenerateIdempotent(t *testing.T) {
	dir := t.TempDir()
	p1, err := WriteAgentsMarkdown(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, err := os.ReadFile(p1)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := WriteAgentsMarkdown(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p1 != p2 {
		t.Fatalf("path changed %s vs %s", p1, p2)
	}
	b, err := os.ReadFile(p2)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatal("AGENTS.md not idempotent")
	}
}

func TestAgentsGenerateCommandWritesFile(t *testing.T) {
	dir := t.TempDir()
	cmd := &AgentsGenerateCommand{}
	if err := cmd.Handle([]string{dir}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "LayerIntelligence") {
		t.Fatalf("catalog missing:\n%s", body)
	}
}
