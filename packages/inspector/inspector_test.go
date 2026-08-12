package inspector_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/inspector"
)

func TestInspectorRecordsRequests(t *testing.T) {
	mgr := inspector.New(10)
	handler := mgr.Middleware()(func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	})
	_ = handler(http.NewRequest(httptest.NewRequest("GET", "/api/health", nil)))
	if mgr.Count() != 1 {
		t.Fatalf("count=%d", mgr.Count())
	}
	entries := mgr.Recent(5)
	if entries[0].Path != "/api/health" || entries[0].Status != 200 {
		t.Fatalf("unexpected %#v", entries[0])
	}
}
