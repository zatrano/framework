package timing_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/timing"
)

func TestTimingMarks(t *testing.T) {
	r := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	req := http.NewRequest(r)
	timing.Add(req, "db", 12*time.Millisecond, "query")
	timing.Add(req, "view", 5*time.Millisecond)
	header := timing.Header(req, 20*time.Millisecond)
	if !strings.Contains(header, "app;dur=") || !strings.Contains(header, "db;") || !strings.Contains(header, "total;") {
		t.Fatal(header)
	}
}
