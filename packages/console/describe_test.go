package console

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDescribeJSONHasRouter(t *testing.T) {
	var buf bytes.Buffer
	cmd := &DescribeCommand{out: &buf}
	if err := cmd.Handle([]string{"--format=json"}); err != nil {
		t.Fatal(err)
	}
	raw := buf.Bytes()
	if !json.Valid(raw) {
		t.Fatalf("invalid JSON:\n%s", raw)
	}
	var doc DescribeDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	router, ok := doc.Contracts["Router"]
	if !ok {
		t.Fatalf("contracts.Router missing; have %v", keysOf(doc.Contracts))
	}
	if router.Name != "Router" {
		t.Fatalf("name %q", router.Name)
	}
	foundGet := false
	for _, m := range router.Methods {
		if m.Name == "Get" {
			foundGet = true
			if m.Signature == "" {
				t.Fatal("Router.Get signature empty")
			}
		}
	}
	if !foundGet {
		t.Fatalf("Router.Get missing: %+v", router.Methods)
	}
	if doc.Providers.Interface.Name != "Provider" {
		t.Fatalf("provider interface %q", doc.Providers.Interface.Name)
	}
	if len(doc.Routing.Primitives) < 2 {
		t.Fatalf("routing primitives: %+v", doc.Routing.Primitives)
	}
	var intel []string
	for _, layer := range doc.Catalog.Layers {
		if layer.Constant != "LayerIntelligence" {
			continue
		}
		for _, p := range layer.Packages {
			intel = append(intel, p.Name)
		}
	}
	for _, want := range []string{"ai", "rag", "agent"} {
		if !containsStr(intel, want) {
			t.Fatalf("catalog intelligence missing %s: %v", want, intel)
		}
	}
}

func TestParseContractsIncludesNewInterface(t *testing.T) {
	dir := t.TempDir()
	src := "package contracts\n\ntype ExtraSurface interface {\n\tPing() string\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "extra.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := parseContractInterfaces(dir)
	if err != nil {
		t.Fatal(err)
	}
	extra, ok := got["ExtraSurface"]
	if !ok {
		t.Fatalf("ExtraSurface missing: %v", keysOf(got))
	}
	if len(extra.Methods) != 1 || extra.Methods[0].Name != "Ping" {
		t.Fatalf("methods %+v", extra.Methods)
	}
}

func TestDescribeSampleRoutesFromApp(t *testing.T) {
	root := t.TempDir()
	web := filepath.Join(root, "app", "routes", "web")
	if err := os.MkdirAll(web, 0o755); err != nil {
		t.Fatal(err)
	}
	src := "package web\n\nfunc register(r *Router) {\n\tr.Get(\"/demo\", nil)\n}\n"
	if err := os.WriteFile(filepath.Join(web, "demo.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	doc, err := BuildDescribeDocument(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Routing.SampleRoutes) == 0 {
		t.Fatal("expected sample routes from fixture app")
	}
	found := false
	for _, r := range doc.Routing.SampleRoutes {
		if r.Call == "Get" && r.Path == "/demo" && r.Group == "web" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Get /demo not found: %+v", doc.Routing.SampleRoutes)
	}
}

func keysOf(m map[string]ContractType) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
