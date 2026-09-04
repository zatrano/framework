package report_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/framework/report"
)

func TestReportCapture(t *testing.T) {
	m := report.New(10)
	m.Capture(fmt.Errorf("boom"), nil)
	m.Capture(fmt.Errorf("again"), nil)
	if m.Count() != 2 {
		t.Fatal(m.Count())
	}
	recent := m.Recent(1)
	if recent[0].Message != "again" {
		t.Fatalf("%+v", recent)
	}
}

func TestReportWebhook(t *testing.T) {
	var (
		mu   sync.Mutex
		got  report.Event
		hits int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, _ := io.ReadAll(r.Body)
		var ev report.Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			t.Errorf("decode: %v", err)
			w.WriteHeader(400)
			return
		}
		mu.Lock()
		got = ev
		hits++
		mu.Unlock()
		w.WriteHeader(204)
	}))
	defer srv.Close()

	m := report.New(10)
	m.SetWebhook(srv.URL)
	m.Capture(fmt.Errorf("remote boom"), nil, "warning")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := hits
		mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if hits != 1 {
		t.Fatalf("hits=%d", hits)
	}
	if got.Message != "remote boom" || got.Level != "warning" {
		t.Fatalf("%+v", got)
	}
	if m.Count() != 1 {
		t.Fatal("expected memory capture")
	}
}
