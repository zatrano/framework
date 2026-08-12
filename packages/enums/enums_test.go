package enums_test

import (
	"testing"

	"github.com/zatrano/framework/packages/enums"
)

func TestStringEnum(t *testing.T) {
	status := enums.NewString("post_status", "draft:Draft", "published:Published")
	if !status.Contains("draft") || status.Label("draft") != "Draft" {
		t.Fatal("draft case failed")
	}
	if _, err := status.Try("nope"); err == nil {
		t.Fatal("expected error")
	}
	reg := enums.NewRegistry()
	reg.Register(status)
	got, ok := reg.Get("post_status")
	if !ok || !got.Contains("published") {
		t.Fatal("registry failed")
	}
}
