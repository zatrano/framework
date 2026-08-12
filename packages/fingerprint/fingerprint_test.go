package fingerprint_test

import (
	"testing"

	"github.com/zatrano/framework/packages/fingerprint"
)

func TestHashStable(t *testing.T) {
	a := fingerprint.Hash("1.1.1.1", "Mozilla/5.0")
	b := fingerprint.Hash("1.1.1.1", "Mozilla/5.0")
	c := fingerprint.Hash("1.1.1.1", "Other")
	if a != b {
		t.Fatal("expected stable hash")
	}
	if a == c {
		t.Fatal("expected different hash")
	}
	if len(fingerprint.Short(a)) != 16 {
		t.Fatalf("short=%q", fingerprint.Short(a))
	}
}
