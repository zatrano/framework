package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/session"
)

func TestSessionPathTraversalRejected(t *testing.T) {
	dir := t.TempDir()
	mgr := session.NewManager(dir, 120)

	// Crafted cookie must not read/write outside the session directory.
	outside := filepath.Join(dir, "..", "pwned.json")
	_ = os.WriteFile(outside, []byte(`{"values":{"hack":true},"flash":{}}`), 0o600)

	bag, err := mgr.Start("../pwned.json")
	if err != nil {
		t.Fatal(err)
	}
	if bag.Get("hack") != nil {
		t.Fatal("path traversal session id should not load outside file")
	}
	if bag.ID() == "../pwned.json" || bag.ID() == "..\\pwned.json" {
		t.Fatalf("invalid id retained: %q", bag.ID())
	}

	bag2, err := mgr.Start("../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	if len(bag2.ID()) != 32 {
		t.Fatalf("expected new hex session id, got %q", bag2.ID())
	}
}
