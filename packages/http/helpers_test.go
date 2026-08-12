package http_test

import (
	"testing"

	"github.com/zatrano/framework/packages/http"
)

func TestAbortIf(t *testing.T) {
	if http.AbortIf(false, 403) != nil {
		t.Fatal("expected nil")
	}
	resp := http.AbortIf(true, 403, "nope")
	if resp == nil || resp.StatusCode() != 403 {
		t.Fatalf("unexpected %#v", resp)
	}
}

func TestRescue(t *testing.T) {
	resp := http.Rescue(func() *http.Response {
		panic("boom")
	})
	if resp == nil || resp.StatusCode() != 500 {
		t.Fatalf("unexpected %#v", resp)
	}
}
