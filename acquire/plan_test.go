package acquire

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/manifest"
	"github.com/zatrano/framework/v2/registry"
)

func TestFromResultTaggedVersion(t *testing.T) {
	got, err := FromResult(registry.Result{
		Package: registry.Package{
			Name: "mongo", Import: manifest.DefaultModule + "/mongo",
			Module: manifest.DefaultModule + "/mongo", Kind: manifest.KindService, Heavy: true,
		},
		Release: registry.Release{Version: "1.2.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != SchemaV1 || got.Selected != "1.2.0" {
		t.Fatalf("%#v", got)
	}
	if got.Query != manifest.DefaultModule+"/mongo@v1.2.0" {
		t.Fatalf("query=%q", got.Query)
	}
	if got.GoGetArg() != got.Query || strings.Contains(got.Query, "latest") {
		t.Fatalf("goget=%q", got.GoGetArg())
	}
}

func TestFromResultMainChannel(t *testing.T) {
	got, err := FromResult(registry.Result{
		Package: registry.Package{
			Name: "session", Import: manifest.DefaultModule + "/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
		},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Selected != registry.ChannelMain {
		t.Fatalf("selected=%q", got.Selected)
	}
	if got.Query != manifest.DefaultModule+"@main" {
		t.Fatalf("query=%q — toolchain may rewrite to a pseudo-version in go.mod", got.Query)
	}
}

func TestFromResultNormalizesVPrefix(t *testing.T) {
	got, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: manifest.DefaultModule},
		Release: registry.Release{Version: "v1.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Query != manifest.DefaultModule+"@v1.0.0" || got.Selected != "1.0.0" {
		t.Fatalf("%#v", got)
	}
}

func TestSharedModuleSameQuery(t *testing.T) {
	mod := manifest.DefaultModule
	rel := registry.Release{Channel: registry.ChannelMain}
	a, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: mod},
		Release: rel,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := FromResult(registry.Result{
		Package: registry.Package{Name: "auth", Module: mod},
		Release: rel,
	})
	if err != nil {
		t.Fatal(err)
	}
	if a.Query != b.Query {
		t.Fatalf("shared module must produce one go get: %q vs %q", a.Query, b.Query)
	}
	if a.Name == b.Name {
		t.Fatal("catalog names stay distinct for later enablement")
	}
}

func TestFromResultRejectsLatestAndEmpty(t *testing.T) {
	_, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: manifest.DefaultModule},
		Release: registry.Release{Version: "latest"},
	})
	if err == nil {
		t.Fatal("latest is a resolve selector, not a go get pin")
	}
	_, err = FromResult(registry.Result{})
	if err == nil {
		t.Fatal("expected missing identity")
	}
}

func TestPlanJSONIsNotALockfile(t *testing.T) {
	p, err := FromResult(registry.Result{
		Package: registry.Package{Name: "session", Module: manifest.DefaultModule},
		Release: registry.Release{Channel: registry.ChannelMain},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, ban := range []string{"lock", "gosum", "checksum", "pseudo"} {
		if strings.Contains(strings.ToLower(s), ban) {
			t.Fatalf("plan must not pretend to be lock state (%s):\n%s", ban, s)
		}
	}
	if !strings.Contains(s, `"schema":"zatrano.acquire/v1"`) {
		t.Fatalf("%s", s)
	}
}
