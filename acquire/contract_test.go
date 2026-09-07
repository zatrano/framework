package acquire

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/manifest"
	"github.com/zatrano/framework/v2/registry"
)

func officialTypeIndex(t *testing.T) registry.Index {
	t.Helper()
	docs := []manifest.Document{
		manifest.Derive(manifest.Input{Name: "session", Kind: manifest.KindService, Layer: manifest.LayerFoundation, Description: "HTTP sessions"}),
		manifest.Derive(manifest.Input{Name: "auth", Kind: manifest.KindService, Layer: manifest.LayerFoundation, Description: "Authentication guards"}),
		manifest.Derive(manifest.Input{Name: "mongo", Kind: manifest.KindService, Layer: manifest.LayerAddon, Heavy: true, Description: "MongoDB client"}),
		manifest.Derive(manifest.Input{Name: "webauthn", Kind: manifest.KindService, Layer: manifest.LayerAddon, Heavy: true, Description: "WebAuthn/passkeys"}),
		manifest.Derive(manifest.Input{Name: "qr", Kind: manifest.KindLibrary, Layer: manifest.LayerAddon, Heavy: true, Description: "QR code generation"}),
		manifest.Derive(manifest.Input{Name: "console", Kind: manifest.KindService, Layer: manifest.LayerFoundation, Description: "CLI application"}),
	}
	idx, err := registry.FromDocuments(docs)
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func mustPlan(t *testing.T, idx registry.Index, q registry.Query) Plan {
	t.Helper()
	got, err := idx.Resolve(q)
	if err != nil {
		t.Fatalf("resolve %+v: %v", q, err)
	}
	p, err := FromResult(got)
	if err != nil {
		t.Fatalf("plan %+v: %v", q, err)
	}
	assertConcretePlan(t, p)
	return p
}

func assertConcretePlan(t *testing.T, p Plan) {
	t.Helper()
	if p.Query == "" || p.Module == "" || p.Name == "" {
		t.Fatalf("incomplete plan %#v", p)
	}
	if strings.Contains(strings.ToLower(p.Query), "latest") || strings.EqualFold(p.Selected, "latest") {
		t.Fatalf("latest must not survive into the plan: %#v", p)
	}
}

func TestResolveLatestOnOfficialTypesBecomesMainPlan(t *testing.T) {
	idx := officialTypeIndex(t)
	session := mustPlan(t, idx, registry.Query{Name: "session", Version: "latest"})
	auth := mustPlan(t, idx, registry.Query{Name: "auth"})
	if session.Module != manifest.DefaultModule || auth.Module != manifest.DefaultModule {
		t.Fatalf("shared module: session=%q auth=%q", session.Module, auth.Module)
	}
	if session.Query != auth.Query || session.Query != manifest.DefaultModule+"@main" {
		t.Fatalf("shared acquisition unit: %q vs %q", session.Query, auth.Query)
	}
	if session.Name == auth.Name {
		t.Fatal("catalog names stay distinct for enablement")
	}
	if session.Selected != registry.ChannelMain {
		t.Fatalf("untagged latest is source main, selected=%q", session.Selected)
	}
}

func TestResolveHeavyPackagesOwnModules(t *testing.T) {
	idx := officialTypeIndex(t)
	want := map[string]string{
		"mongo":    manifest.DefaultModule + "/mongo",
		"webauthn": manifest.DefaultModule + "/webauthn",
		"qr":       manifest.DefaultModule + "/qr",
	}
	queries := map[string]string{}
	for name, mod := range want {
		p := mustPlan(t, idx, registry.Query{Name: name, Version: "main"})
		if !p.Heavy {
			t.Fatalf("%s should be heavy", name)
		}
		if p.Module != mod || p.Query != mod+"@main" {
			t.Fatalf("%s module=%q query=%q want %s@main", name, p.Module, p.Query, mod)
		}
		queries[p.Query] = name
	}
	if len(queries) != 3 {
		t.Fatalf("heavy packages must not share a go get: %v", queries)
	}
	con := mustPlan(t, idx, registry.Query{Name: "console"})
	if con.Module != manifest.FrameworkModule || con.Heavy {
		t.Fatalf("console=%#v", con)
	}
	if con.Query != manifest.FrameworkModule+"@main" {
		t.Fatalf("console query=%q", con.Query)
	}
}

func TestResolveTaggedThenPlanConcreteVersion(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "mongo", Import: manifest.DefaultModule + "/mongo",
			Module: manifest.DefaultModule + "/mongo", Kind: manifest.KindService,
			Layer: manifest.LayerAddon, Heavy: true,
			Releases: []registry.Release{
				{Channel: registry.ChannelMain},
				{Version: "1.0.0"},
				{Version: "1.4.0", FrameworkMin: "2.0.7"},
			},
		}},
	}
	p := mustPlan(t, idx, registry.Query{Name: "mongo", Version: "latest"})
	if p.Selected != "1.4.0" || p.Query != manifest.DefaultModule+"/mongo@v1.4.0" {
		t.Fatalf("concrete latest %#v", p)
	}
	exact := mustPlan(t, idx, registry.Query{Name: "mongo", Version: "v1.0.0"})
	if exact.Selected != "1.0.0" || exact.Query != manifest.DefaultModule+"/mongo@v1.0.0" {
		t.Fatalf("exact %#v", exact)
	}
}

func TestIncompatibleResolveYieldsNoPlan(t *testing.T) {
	idx := registry.Index{
		Schema: registry.SchemaV1,
		Packages: []registry.Package{{
			Name: "session", Import: manifest.DefaultModule + "/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation,
			Releases: []registry.Release{
				{Channel: registry.ChannelMain, FrameworkMin: "2.0.0"},
				{Version: "9.0.0", FrameworkMin: "3.0.0"},
			},
		}},
	}
	_, err := idx.Resolve(registry.Query{Name: "session", Version: "latest", Framework: "1.6.6"})
	if err == nil {
		t.Fatal("expected incompatible resolve")
	}
	got, err := idx.Resolve(registry.Query{Name: "session", Version: "latest", Framework: "2.0.7"})
	if err != nil {
		t.Fatal(err)
	}
	p, err := FromResult(got)
	if err != nil {
		t.Fatal(err)
	}
	if p.Selected != registry.ChannelMain {
		t.Fatalf("compatible kernel falls back to main, got %#v", p)
	}
}

func TestUnknownPackageHasNoPlan(t *testing.T) {
	idx := officialTypeIndex(t)
	_, err := idx.Resolve(registry.Query{Name: "does-not-exist"})
	if err == nil {
		t.Fatal("expected unknown package")
	}
}

func TestTargetsCollapsesSharedModuleAndIgnoresInputOrder(t *testing.T) {
	idx := officialTypeIndex(t)
	session := mustPlan(t, idx, registry.Query{Name: "session"})
	auth := mustPlan(t, idx, registry.Query{Name: "auth"})
	mongo := mustPlan(t, idx, registry.Query{Name: "mongo"})
	webauthn := mustPlan(t, idx, registry.Query{Name: "webauthn"})
	qr := mustPlan(t, idx, registry.Query{Name: "qr"})
	console := mustPlan(t, idx, registry.Query{Name: "console"})

	forward, err := Targets([]Plan{session, auth, mongo, webauthn, qr, console})
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := Targets([]Plan{console, qr, webauthn, mongo, auth, session})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		manifest.FrameworkModule + "@main",
		manifest.DefaultModule + "@main",
		manifest.DefaultModule + "/mongo@main",
		manifest.DefaultModule + "/qr@main",
		manifest.DefaultModule + "/webauthn@main",
	}
	if len(forward) != 5 {
		t.Fatalf("session+auth must collapse to one target: %v", forward)
	}
	for i, q := range want {
		if forward[i] != q || reverse[i] != q {
			t.Fatalf("targets[%d]=%q reverse=%q want %q\nforward=%v", i, forward[i], reverse[i], q, forward)
		}
	}
}

func TestTargetsRejectsConflictingPinsForOneModule(t *testing.T) {
	mod := manifest.DefaultModule
	a, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: mod},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := FromResult(registry.Result{
		Package: registry.Package{Name: "auth", Module: mod},
		Release: registry.Release{Version: "1.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Targets([]Plan{a, b}); err == nil {
		t.Fatal("shared module with two pins must not silently merge")
	}
	if _, err := Targets([]Plan{b, a}); err == nil {
		t.Fatal("conflict must not depend on input order")
	}
}

func TestMissingModuleIdentityHasNoPlan(t *testing.T) {
	if _, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session"},
		Release: registry.Release{Channel: registry.ChannelMain},
	}); err == nil {
		t.Fatal("missing module")
	}
	if _, err := FromResult(registry.Result{
		Package: registry.Package{Module: manifest.DefaultModule},
		Release: registry.Release{Channel: registry.ChannelMain},
	}); err == nil {
		t.Fatal("missing name")
	}
	if _, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: manifest.DefaultModule},
		Release: registry.Release{Channel: "dev"},
	}); err == nil {
		t.Fatal("unknown channel")
	}
}
