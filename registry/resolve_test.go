package registry

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/manifest"
)

func constrainedSession() Index {
	return Index{
		Schema: SchemaV1,
		Packages: []Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []Release{
				{Channel: ChannelMain, FrameworkMin: "2.0.0"},
				{Version: "1.0.0", FrameworkMin: "2.0.0"},
				{Version: "1.1.0", FrameworkMin: "2.0.1"},
				{Version: "1.2.0", FrameworkMin: "2.0.3"},
				{Version: "1.3.0", FrameworkMin: "2.0.4"},
			},
		}},
	}
}

func TestResolveConstraintTable(t *testing.T) {
	idx := constrainedSession()
	type tc struct {
		name    string
		q       Query
		wantVer string
		wantCh  string
		wantErr string
	}
	cases := []tc{
		{name: "empty version is latest", q: Query{Name: "session"}, wantVer: "1.3.0"},
		{name: "latest no framework filter", q: Query{Name: "session", Version: "latest"}, wantVer: "1.3.0"},
		{name: "Latest is case-insensitive", q: Query{Name: "session", Version: "Latest"}, wantVer: "1.3.0"},
		{name: "empty framework still picks highest tag", q: Query{Name: "session", Version: "latest", Framework: ""}, wantVer: "1.3.0"},
		{name: "framework 2.0.0 skips newer mins", q: Query{Name: "session", Version: "latest", Framework: "2.0.0"}, wantVer: "1.0.0"},
		{name: "framework 2.0.1", q: Query{Name: "session", Version: "latest", Framework: "2.0.1"}, wantVer: "1.1.0"},
		{name: "framework 2.0.3", q: Query{Name: "session", Version: "latest", Framework: "2.0.3"}, wantVer: "1.2.0"},
		{name: "framework 2.0.4", q: Query{Name: "session", Version: "latest", Framework: "v2.0.4"}, wantVer: "1.3.0"},
		{name: "exact v-prefix", q: Query{Name: "session", Version: "v1.1.0"}, wantVer: "1.1.0"},
		{name: "exact with compatible framework", q: Query{Name: "session", Version: "1.2.0", Framework: "2.0.3"}, wantVer: "1.2.0"},
		{name: "main when compatible", q: Query{Name: "session", Version: "main", Framework: "2.0.0"}, wantCh: ChannelMain},
		{name: "unknown name", q: Query{Name: "nope"}, wantErr: "unknown package"},
		{name: "kind mismatch", q: Query{Name: "session", Kind: manifest.KindLibrary}, wantErr: "kind is"},
		{name: "exact tag too new for framework", q: Query{Name: "session", Version: "1.3.0", Framework: "2.0.3"}, wantErr: "needs framework"},
		{name: "main too new for framework", q: Query{Name: "session", Version: "main", Framework: "1.9.0"}, wantErr: "needs framework"},
		{name: "missing exact tag", q: Query{Name: "session", Version: "9.9.9"}, wantErr: "has no version"},
		{name: "incompatible latest including main", q: Query{Name: "session", Version: "latest", Framework: "1.6.6"}, wantErr: "no compatible release"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := idx.Resolve(c.q)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("err=%v want substring %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.wantVer != "" && got.Release.Version != c.wantVer {
				t.Fatalf("version=%q want %q", got.Release.Version, c.wantVer)
			}
			if c.wantCh != "" && got.Release.Channel != c.wantCh {
				t.Fatalf("channel=%q want %q", got.Release.Channel, c.wantCh)
			}
			if c.wantVer != "" && got.Release.Channel != "" {
				t.Fatalf("tagged result must not also be a channel: %#v", got.Release)
			}
		})
	}
}

func TestResolveLatestFallsBackToMainWhenTagsIncompatible(t *testing.T) {
	idx := Index{
		Schema: SchemaV1,
		Packages: []Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []Release{
				{Channel: ChannelMain, FrameworkMin: "2.0.0"},
				{Version: "8.0.0", FrameworkMin: "3.0.0"},
				{Version: "9.0.0", FrameworkMin: "3.0.0"},
			},
		}},
	}
	got, err := idx.Resolve(Query{Name: "session", Version: "latest", Framework: "2.0.4"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Release.Version != "" || got.Release.Channel != ChannelMain {
		t.Fatalf("incompatible tags must fall back to source main, got %#v", got.Release)
	}
}

func TestResolveNoMainAndIncompatibleTagsErrors(t *testing.T) {
	idx := Index{
		Schema: SchemaV1,
		Packages: []Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []Release{
				{Version: "1.0.0", FrameworkMin: "3.0.0"},
			},
		}},
	}
	_, err := idx.Resolve(Query{Name: "session", Version: "latest", Framework: "2.0.4"})
	if err == nil || !strings.Contains(err.Error(), "no compatible release") {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveHeavyModuleOwnVersionStream(t *testing.T) {
	idx := Index{
		Schema: SchemaV1,
		Packages: []Package{{
			Name: "mongo", Import: "github.com/zatrano/packages/mongo",
			Module: manifest.DefaultModule + "/mongo", Kind: manifest.KindService,
			Layer: manifest.LayerAddon, Heavy: true, Description: "MongoDB client",
			Releases: []Release{
				{Channel: ChannelMain},
				{Version: "1.0.0"},
				{Version: "1.1.0", FrameworkMin: "2.0.4"},
			},
		}},
	}
	got, err := idx.Resolve(Query{Name: "mongo", Version: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Package.Module != manifest.DefaultModule+"/mongo" || !got.Package.Heavy {
		t.Fatalf("heavy module=%q heavy=%v", got.Package.Module, got.Package.Heavy)
	}
	if got.Release.Version != "1.1.0" {
		t.Fatalf("latest=%q", got.Release.Version)
	}
	got, err = idx.Resolve(Query{Name: "mongo", Version: "latest", Framework: "2.0.3"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Release.Version != "1.0.0" {
		t.Fatalf("compat latest=%q", got.Release.Version)
	}
}

func TestSearchDoesNotPickAVersion(t *testing.T) {
	idx := constrainedSession()
	hits := idx.Search(Filter{Query: "session"})
	if len(hits) != 1 {
		t.Fatalf("hits=%d", len(hits))
	}
	if len(hits[0].Releases) < 2 {
		t.Fatalf("search returns identity including all known releases, not one selected: %#v", hits[0].Releases)
	}
}

func TestSearchLayerDescriptionAndHeavyFalse(t *testing.T) {
	idx, err := FromDocuments(sampleDocs())
	if err != nil {
		t.Fatal(err)
	}
	found := idx.Search(Filter{Query: "http sessions"})
	if len(found) != 1 || found[0].Name != "session" {
		t.Fatalf("description query=%#v", found)
	}
	found = idx.Search(Filter{Layer: manifest.LayerFoundation})
	if len(found) != 2 {
		t.Fatalf("foundation=%d", len(found))
	}
	heavy := false
	found = idx.Search(Filter{Heavy: &heavy})
	for _, p := range found {
		if p.Heavy {
			t.Fatalf("Heavy=false must exclude mongo: %#v", p)
		}
	}
	if len(found) != 3 {
		t.Fatalf("non-heavy=%d want 3", len(found))
	}
	all := idx.Search(Filter{})
	if len(all) != 4 {
		t.Fatalf("empty filter=%d", len(all))
	}
	found = idx.Search(Filter{Kind: "Library"})
	if len(found) != 1 || found[0].Name != "collection" {
		t.Fatalf("kind case=%#v", found)
	}
}
