package bloom_test

import (
	"testing"

	"github.com/zatrano/framework/packages/bloom"
)

func TestBloomFilter(t *testing.T) {
	f := bloom.New(100, 0.01)
	f.Add("alice")
	f.Add("bob")
	if !f.MightContain("alice") || !f.MightContain("bob") {
		t.Fatal("expected members present")
	}
	if f.MightContain("carol") {
		t.Fatal("unexpected member")
	}
	if f.Len() != 2 {
		t.Fatalf("len=%d", f.Len())
	}
	f.Reset()
	if f.MightContain("alice") || f.Len() != 0 {
		t.Fatal("expected empty after reset")
	}
}
