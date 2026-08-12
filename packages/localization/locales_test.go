package localization_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/localization"
)

func TestPublishedAndOptions(t *testing.T) {
	root := t.TempDir()
	if localization.Published(root) {
		t.Fatal("empty dir should not be published")
	}
	if err := os.MkdirAll(filepath.Join(root, "en"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "en", "messages.json"), []byte(`{"x":"y"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "tr"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !localization.Published(root) {
		t.Fatal("expected published")
	}
	opts := localization.Options(root, "tr")
	if len(opts) < 2 {
		t.Fatalf("opts=%v", opts)
	}
	found := false
	for _, opt := range opts {
		if opt["code"] == "tr" && opt["selected"] == true {
			found = true
		}
	}
	if !found {
		t.Fatalf("tr not selected: %#v", opts)
	}
}
