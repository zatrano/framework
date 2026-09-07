package console

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/distribution/manifest"
	"github.com/zatrano/framework/v2/distribution/registry"
	"github.com/zatrano/framework/v2/kernel"
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

func TestPackageSearchJSONAndTextContract(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageSearchCommand{out: &buf}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	keys := jsonTopKeys(t, buf.Bytes())
	for _, want := range []string{"name", "import", "module", "kind", "layer", "description"} {
		if !keys[want] {
			t.Fatalf("search JSON missing %q: %v", want, keys)
		}
	}
	for _, ban := range []string{"selected", "releases", "release"} {
		if keys[ban] {
			t.Fatalf("search JSON must not include %q", ban)
		}
	}

	buf.Reset()
	if err := cmd.Handle([]string{"session"}); err != nil {
		t.Fatal(err)
	}
	header := strings.Split(buf.String(), "\n")[0]
	for _, col := range []string{"NAME", "KIND", "LAYER", "HEAVY", "MODULE", "DESCRIPTION"} {
		if !strings.Contains(header, col) {
			t.Fatalf("search header missing %s:\n%s", col, header)
		}
	}
	for _, ban := range []string{"SELECTED", "VERSION", "RELEASE"} {
		if strings.Contains(header, ban) {
			t.Fatalf("search must not print %s:\n%s", ban, header)
		}
	}
}

func TestPackageSearchHeavyOfficialModules(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageSearchCommand{out: &buf}
	if err := cmd.Handle([]string{"--heavy", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var hits []searchHit
	if err := json.Unmarshal(buf.Bytes(), &hits); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, h := range hits {
		got[h.Name] = h.Module
		if !h.Heavy {
			t.Fatalf("%s should be heavy", h.Name)
		}
	}
	want := map[string]string{
		"mongo":    manifest.DefaultModule + "/mongo",
		"webauthn": manifest.DefaultModule + "/webauthn",
		"qr":       manifest.DefaultModule + "/qr",
	}
	if len(got) != 3 {
		t.Fatalf("heavy hits=%v", got)
	}
	for name, mod := range want {
		if got[name] != mod {
			t.Fatalf("%s module=%q want %q", name, got[name], mod)
		}
	}
}

func TestPackageInfoJSONContract(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageInfoCommand{out: &buf}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var p registry.Package
	if err := json.Unmarshal(buf.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Name != "session" || p.Import == "" || p.Module != manifest.DefaultModule {
		t.Fatalf("%#v", p)
	}
	if len(p.Releases) == 0 {
		t.Fatal("info lists known releases")
	}
	keys := jsonTopKeys(t, buf.Bytes())
	if keys["selected"] {
		t.Fatal("info must not select a version")
	}
	if !keys["releases"] {
		t.Fatal("info JSON needs releases")
	}
}

func TestPackageInfoUnknownPackage(t *testing.T) {
	cmd := &PackageInfoCommand{out: ioDiscard()}
	if err := cmd.Handle([]string{"does-not-exist"}); err == nil {
		t.Fatal("expected unknown package")
	}
}

func TestPackageInfoHeavyMongoOwnModule(t *testing.T) {
	var buf bytes.Buffer
	cmd := &PackageInfoCommand{out: &buf}
	if err := cmd.Handle([]string{"mongo", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var p registry.Package
	if err := json.Unmarshal(buf.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Heavy || p.Module != manifest.DefaultModule+"/mongo" {
		t.Fatalf("%#v", p)
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
	keys := jsonTopKeys(t, buf.Bytes())
	for _, want := range []string{"name", "import", "module", "kind", "selected", "release"} {
		if !keys[want] {
			t.Fatalf("resolve JSON missing %q: %v", want, keys)
		}
	}
}

func TestPackageResolveIncompatibleFramework(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []registry.Release{
				{Channel: registry.ChannelMain, FrameworkMin: "2.0.0"},
				{Version: "1.0.0", FrameworkMin: "2.0.0"},
			},
		}},
	}
	cmd := &PackageResolveCommand{out: ioDiscard(), index: &idx}
	if err := cmd.Handle([]string{"session", "latest", "--framework=1.6.6"}); err == nil {
		t.Fatal("expected incompatible framework")
	}
}

func TestPackageResolveFallsBackToMainWhenTagsTooNew(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "mongo", Import: "github.com/zatrano/packages/mongo",
			Module: manifest.DefaultModule + "/mongo", Kind: manifest.KindService,
			Layer: manifest.LayerAddon, Heavy: true, Description: "MongoDB client",
			Releases: []registry.Release{
				{Channel: registry.ChannelMain, FrameworkMin: "2.0.0"},
				{Version: "9.0.0", FrameworkMin: "3.0.0"},
			},
		}},
	}
	var buf bytes.Buffer
	cmd := &PackageResolveCommand{out: &buf, index: &idx}
	if err := cmd.Handle([]string{"mongo", "--framework=2.0.4", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var got resolveView
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Selected != registry.ChannelMain || got.Release.Version != "" || !got.Heavy {
		t.Fatalf("heavy latest with incompatible tags must be source main: %#v", got)
	}
	if got.Module != manifest.DefaultModule+"/mongo" {
		t.Fatalf("module=%q", got.Module)
	}
}

func TestPackageResolveEmptySelectorIsLatest(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []registry.Release{
				{Channel: registry.ChannelMain},
				{Version: "1.0.0"},
				{Version: "1.4.0"},
			},
		}},
	}
	var buf bytes.Buffer
	cmd := &PackageResolveCommand{out: &buf, index: &idx}
	if err := cmd.Handle([]string{"session", "--format=json"}); err != nil {
		t.Fatal(err)
	}
	var got resolveView
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Selected != "1.4.0" {
		t.Fatalf("empty selector must be latest via registry: %q", got.Selected)
	}
}

func jsonTopKeys(t *testing.T, raw []byte) map[string]bool {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	switch x := v.(type) {
	case []any:
		if len(x) == 0 {
			t.Fatal("empty JSON array")
		}
		m, ok := x[0].(map[string]any)
		if !ok {
			t.Fatalf("array item %T", x[0])
		}
		obj = m
	case map[string]any:
		obj = x
	default:
		t.Fatalf("JSON %T", v)
	}
	out := map[string]bool{}
	for k := range obj {
		out[k] = true
	}
	return out
}

func ioDiscard() *bytes.Buffer {
	return &bytes.Buffer{}
}
