package http_test

import (
	"bufio"
	"errors"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

type hijackRecorder struct {
	stdhttp.ResponseWriter
	hijacked bool
}

func (h *hijackRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, errors.New("test hijack unused")
}

func TestHijackWriteToInvokesCallback(t *testing.T) {
	called := false
	resp := http.Hijack(func(w stdhttp.ResponseWriter) error {
		called = true
		if _, ok := w.(stdhttp.Hijacker); !ok {
			return errors.New("hijacking not supported")
		}
		return nil
	})
	if resp.StatusCode() != 101 {
		t.Fatalf("status=%d", resp.StatusCode())
	}

	unsupported := httptest.NewRecorder()
	if err := resp.WriteTo(unsupported); err == nil || !strings.Contains(err.Error(), "hijacking not supported") {
		t.Fatalf("unsupported writer err=%v", err)
	}
	if !called {
		t.Fatal("callback must run even when Hijacker is missing")
	}

	called = false
	ok := http.Hijack(func(w stdhttp.ResponseWriter) error {
		called = true
		hj, ok := w.(stdhttp.Hijacker)
		if !ok {
			return errors.New("hijacking not supported")
		}
		_, _, err := hj.Hijack()
		if err == nil {
			t.Fatal("expected stub Hijack error")
		}
		return nil
	})
	rec := &hijackRecorder{ResponseWriter: httptest.NewRecorder()}
	if err := ok.WriteTo(rec); err != nil {
		t.Fatal(err)
	}
	if !called || !rec.hijacked {
		t.Fatal("expected Hijacker path")
	}
}
