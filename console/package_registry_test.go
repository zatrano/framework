package console

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel"
	"github.com/zatrano/framework/v2/manifest"
	"github.com/zatrano/framework/v2/registry"
)

func TestPackageSearchDiscoversWithoutSelectingVersion(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageSearchCommand{out: &buf}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var hits []searchHit
	if err := json.Unmarshal(buf.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Name != "session" {
		t.Fatalf("hits=%#v", hits)
	}
	if hits[0].Module != manifest.DefaultModule {
		t.Fatalf("module=%q", hits[0].Module)
	}
	raw := buf.String()
	if strings.Contains(raw, `"selected"`) || strings.Contains(raw, `"releases"`) {
		t.Fatalf("search must not pick or list releases:\n%s", raw)
	}
}

func TestPackageSearchKindLibraryExcludesSession(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageSearchCommand{out: &buf}
	if err := cmd.Handle([]string{"--kind=library", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var hits []searchHit
	if err := json.Unmarshal(buf.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) < 30 {
		t.Fatalf("expected many libraries, got %d", len(hits))
	}
	for _, h := range hits {
		if h.Name == "session" {
			t.Fatal("session is a service, not a library search hit")
		}
		if h.Kind != manifest.KindLibrary {
			t.Fatalf("%s kind=%q", h.Name, h.Kind)
		}
	}
}

func TestPackageInfoListsReleasesWithoutSelecting(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageInfoCommand{out: &buf}
	if err := cmd.Handle([]string{"mongo"}); err != nil {
		t.Fatal(err)
	}
	text := buf.String()
	if !strings.Contains(text, "name: mongo") || !strings.Contains(text, "module: "+manifest.DefaultModule+"/mongo") {
		t.Fatalf("info:\n%s", text)
	}
	if !strings.Contains(text, "channel: main") && !strings.Contains(text, "- main") {
		t.Fatalf("expected known releases, not a selected version:\n%s", text)
	}
	if strings.Contains(text, "selected:") {
		t.Fatalf("info must not select a version:\n%s", text)
	}
}

func TestPackageResolveLatestFallsBackToMain(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageResolveCommand{out: &buf}
	if err := cmd.Handle([]string{"session", "latest", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var got resolveView
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "session" || got.Selected != registry.ChannelMain {
		t.Fatalf("untagged official latest must be source channel main: %#v", got)
	}
	if got.Release.Version != "" {
		t.Fatalf("main fallback is not a published version: %#v", got.Release)
	}
}

func TestPackageResolveNameAtVersion(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageResolveCommand{out: &buf}
	if err := cmd.Handle([]string{"session@main"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "selected: main") {
		t.Fatalf("got %s", buf.String())
	}
}

func TestPackageResolveUnknownPackage(t *testing.T) {
	cmd := &PackageResolveCommand{out: ioDiscard()}
	if err := cmd.Handle([]string{"does-not-exist"}); err == nil {
		t.Fatal("expected unknown package")
	}
}

func TestPackageResolveDoesNotMutateTree(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	var buf bytes.Buffer
	cmd := &PackageResolveCommand{app: app, out: &buf}
	if err := cmd.Handle([]string{"collection"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "zatrano") {
		t.Fatalf("resolve must not write go.mod:\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(dir, "bootstrap")); !os.IsNotExist(err) {
		t.Fatal("resolve must not write bootstrap/")
	}
}

func TestPackageResolveUsesRegistryNotLocalPicking(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []registry.Release{
				{Channel: registry.ChannelMain},
				{Version: "1.0.0"},
				{Version: "1.2.0"},
			},
		}},
	}
	var buf bytes.Buffer
	cmd := &PackageResolveCommand{out: &buf, index: &idx}
	if err := cmd.Handle([]string{"session", "latest", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var got resolveView
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Selected != "1.2.0" {
		t.Fatalf("CLI must use registry.Resolve, selected=%q", got.Selected)
	}
}

func ioDiscard() *bytes.Buffer {
	return &bytes.Buffer{}
}
