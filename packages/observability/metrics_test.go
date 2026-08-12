package observability_test

import (
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/observability"
)

func TestPrometheusExport(t *testing.T) {
	m := observability.New()
	m.Observe("GET", "/up", 200, 5*time.Millisecond)
	m.Observe("GET", "/x", 500, 10*time.Millisecond)
	out := m.Prometheus()
	if !strings.Contains(out, "zatrano_http_requests_total 2") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "zatrano_http_errors_total 1") {
		t.Fatal(out)
	}
	if !strings.Contains(out, `zatrano_http_responses_total{code="200"}`) {
		t.Fatal(out)
	}
}
