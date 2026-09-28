package dirs

import (
	"os"
	"path/filepath"
	"testing"
)

type baseOnly struct{ base string }

func (s baseOnly) BasePath(parts ...string) string {
	return filepath.Join(append([]string{s.base}, parts...)...)
}

// minimalApp embeds enough methods? TemplatesDir only calls BasePath via contracts.App.
// Use a fake by temporarily testing path join logic through Dir helpers with a wrapper.

type fakeApp struct{ baseOnly }

func (fakeApp) Container() interface{}                          { return nil }
func (fakeApp) Make(string) (any, error)                        { return nil, nil }
func (fakeApp) Bound(string) bool                               { return false }
func (fakeApp) Config() interface{}                             { return nil }
func (fakeApp) Router() interface{}                             { return nil }
func (fakeApp) Logger() interface{}                             { return nil }
func (fakeApp) Context() interface{}                            { return nil }
func (fakeApp) Encrypter() interface{}                          { return nil }
func (fakeApp) Exceptions() interface{}                         { return nil }
func (fakeApp) Reports() interface{}                            { return nil }
func (fakeApp) Environment() string                             { return "testing" }
func (fakeApp) IsProduction() bool                              { return false }
func (fakeApp) IsDebug() bool                                   { return true }
func (fakeApp) RegisterProviders(...interface{})                {}
func (fakeApp) Bootstrap() error                                { return nil }
func (fakeApp) BootstrapContext(interface{}) error              { return nil }
func (fakeApp) Start() error                                    { return nil }
func (fakeApp) StartContext(interface{}) error                  { return nil }
func (fakeApp) Stop(interface{}) error                          { return nil }
func (fakeApp) Handle(any)                                      {}
func (fakeApp) Run(string) error                                { return nil }
func (fakeApp) SetHTTPBridge(interface{})                       {}
func (fakeApp) HTTPBridge() interface{}                         { return nil }

func TestTemplatesDir(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "templates"), 0o755)
	// TemplatesDir requires contracts.App; exercise Dir via TemplatesDirForCreate with nil
	if TemplatesDir(nil) != "templates" {
		t.Fatal(TemplatesDir(nil))
	}
	if DatabaseDir(nil) != "database" {
		t.Fatal(DatabaseDir(nil))
	}
}

func TestCanonicalConsumerDirs(t *testing.T) {
	dirs := CanonicalConsumerDirs()
	need := map[string]bool{"core": false, "app/http/handlers/web": false, "templates": false, "database": false}
	for _, d := range dirs {
		if _, ok := need[d]; ok {
			need[d] = true
		}
	}
	for k, ok := range need {
		if !ok {
			t.Fatalf("missing %s", k)
		}
	}
}

func TestHandlerRelAllowed(t *testing.T) {
	if !HandlerRelAllowed("app/http/handlers/web/home_handler.go") {
		t.Fatal("web")
	}
	if !HandlerRelAllowed("app/http/handlers/api/home_handler.go") {
		t.Fatal("api")
	}
	if HandlerRelAllowed("app/http/handlers/home.go") {
		t.Fatal("surface required")
	}
	if HandlerKind("app/http/handlers/api/x.go") != SurfaceAPI {
		t.Fatal("api kind")
	}
}
