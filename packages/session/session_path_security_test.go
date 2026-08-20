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

func TestSessionConcurrentSave(t *testing.T) {
	dir := t.TempDir()
	mgr := session.NewManager(dir, 120)
	bag, err := mgr.Start("")
	if err != nil {
		t.Fatal(err)
	}
	bag.Put("n", 0)
	if err := mgr.Save(bag); err != nil {
		t.Fatal(err)
	}

	const workers = 8
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func(n int) {
			b, err := mgr.Start(bag.ID())
			if err != nil {
				errCh <- err
				return
			}
			b.Put("n", n)
			errCh <- mgr.Save(b)
		}(i)
	}
	for i := 0; i < workers; i++ {
		if err := <-errCh; err != nil {
			t.Fatal(err)
		}
	}
	final, err := mgr.Start(bag.ID())
	if err != nil {
		t.Fatal(err)
	}
	if final.Get("n") == nil {
		t.Fatal("expected persisted value after concurrent saves")
	}
}
