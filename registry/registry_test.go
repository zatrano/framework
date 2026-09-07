package registry

import (
	"encoding/json"
	"testing"

	"github.com/zatrano/framework/v2/manifest"
)

func sampleDocs() []manifest.Document {
	return []manifest.Document{
		manifest.Derive(manifest.Input{
			Name: "session", Kind: manifest.KindService, Layer: manifest.LayerFoundation,
			Description: "HTTP sessions", Factory: true,
		}),
		manifest.Derive(manifest.Input{
			Name: "collection", Kind: manifest.KindLibrary, Layer: manifest.LayerAddon,
			Description: "Collection helpers",
		}),
		manifest.Derive(manifest.Input{
			Name: "mongo", Kind: manifest.KindService, Layer: manifest.LayerAddon,
			Description: "MongoDB client", Heavy: true, Factory: true,
		}),
		manifest.Derive(manifest.Input{
			Name: "console", Kind: manifest.KindService, Layer: manifest.LayerFoundation,
			Description: "CLI application",
		}),
	}
}

func TestFromDocumentsUniqueAndMainChannel(t *testing.T) {
	idx, err := FromDocuments(sampleDocs())
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Validate(); err != nil {
		t.Fatal(err)
	}
	sess, ok := idx.Lookup("session")
	if !ok || sess.Module != manifest.DefaultModule {
		t.Fatalf("session module=%q", sess.Module)
	}
	if len(sess.Releases) != 1 || sess.Releases[0].Channel != ChannelMain {
		t.Fatalf("session releases=%#v", sess.Releases)
	}
	mongo, _ := idx.Lookup("mongo")
	if mongo.Module != manifest.DefaultModule+"/mongo" || !mongo.Heavy {
		t.Fatalf("mongo=%#v", mongo)
	}
	con, _ := idx.Lookup("console")
	if con.Module != manifest.FrameworkModule {
		t.Fatalf("console module=%q", con.Module)
	}
}

func TestDuplicateNameRejected(t *testing.T) {
	docs := sampleDocs()
	docs = append(docs, docs[0])
	if _, err := FromDocuments(docs); err == nil {
		t.Fatal("expected duplicate name")
	}
}

func TestResolveLatestPrefersSemverThenMain(t *testing.T) {
	idx := Index{
		Schema: SchemaV1,
		Packages: []Package{{
			Name: "session", Import: "github.com/zatrano/packages/session",
			Module: manifest.DefaultModule, Kind: manifest.KindService,
			Layer: manifest.LayerFoundation, Description: "HTTP sessions",
			Releases: []Release{
				{Channel: ChannelMain},
				{Version: "1.0.0"},
				{Version: "1.2.0", FrameworkMin: "2.0.2"},
				{Version: "1.1.0"},
			},
		}},
	}
	got, err := idx.Resolve(Query{Name: "session", Version: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Release.Version != "1.2.0" {
		t.Fatalf("latest=%q", got.Release.Version)
	}
	got, err = idx.Resolve(Query{Name: "session", Version: "latest", Framework: "2.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Release.Version != "1.1.0" {
		t.Fatalf("compat latest=%q", got.Release.Version)
	}
}

func TestResolveLatestFallsBackToMainWhenNoTags(t *testing.T) {
	idx, err := FromDocuments(sampleDocs())
	if err != nil {
		t.Fatal(err)
	}
	got, err := idx.Resolve(Query{Name: "session", Version: "latest"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Release.Version != "" || got.Release.Channel != ChannelMain {
		t.Fatalf("untagged latest must be source channel main, got %#v", got.Release)
	}
}

func TestResolveMainAndExact(t *testing.T) {
	idx, err := FromDocuments(sampleDocs())
	if err != nil {
		t.Fatal(err)
	}
	got, err := idx.Resolve(Query{Name: "collection", Version: "main"})
	if err != nil || got.Release.Channel != ChannelMain {
		t.Fatalf("%#v %v", got, err)
	}
	_, err = idx.Resolve(Query{Name: "collection", Version: "1.0.0"})
	if err == nil {
		t.Fatal("expected missing tagged version")
	}
}

func TestSearchKindAndQuery(t *testing.T) {
	idx, err := FromDocuments(sampleDocs())
	if err != nil {
		t.Fatal(err)
	}
	libs := idx.Search(Filter{Kind: manifest.KindLibrary})
	if len(libs) != 1 || libs[0].Name != "collection" {
		t.Fatalf("libs=%#v", libs)
	}
	heavy := true
	h := idx.Search(Filter{Heavy: &heavy})
	if len(h) != 1 || h[0].Name != "mongo" {
		t.Fatalf("heavy=%#v", h)
	}
	hits := idx.Search(Filter{Query: "session"})
	if len(hits) != 1 {
		t.Fatalf("query=%#v", hits)
	}
}

func TestKindFilterOnResolve(t *testing.T) {
	idx, err := FromDocuments(sampleDocs())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := idx.Resolve(Query{Name: "session", Kind: manifest.KindLibrary}); err == nil {
		t.Fatal("expected kind mismatch")
	}
}

func TestVerifyDigest(t *testing.T) {
	raw := []byte(`{"schema":"zatrano.package/v1"}`)
	sum := DigestSHA256(raw)
	if err := VerifyDigest(sum, raw); err != nil {
		t.Fatal(err)
	}
	if err := VerifyDigest(sum, []byte("other")); err == nil {
		t.Fatal("expected mismatch")
	}
	if err := VerifyDigest("", raw); err != nil {
		t.Fatal(err)
	}
}

func TestIndexJSONRoundTrip(t *testing.T) {
	idx, err := FromDocuments(sampleDocs()[:1])
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(idx)
	if err != nil {
		t.Fatal(err)
	}
	var got Index
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
}
