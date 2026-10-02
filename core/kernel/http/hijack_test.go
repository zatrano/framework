package http_test

import (
	"net"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestHijackCommitWritesRaw(t *testing.T) {
	t.Parallel()
	gotLeftover := make(chan string, 1)
	er, err := http.ExchangeForTest(func(ctx *rawhttp.Ctx) {
		resp := http.Hijack(func(conn net.Conn, leftover []byte) error {
			defer conn.Close()
			gotLeftover <- string(leftover)
			_, werr := conn.Write([]byte("HIJACKED"))
			return werr
		})
		_ = resp.Commit(ctx)
	}, "GET /ws HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\nPIPELINED")
	if !strings.Contains(string(er.Raw), "HIJACKED") {
		t.Fatalf("err=%v raw=%q", err, er.Raw)
	}
	if strings.Contains(string(er.Raw), "HTTP/1.1") {
		t.Fatalf("server must not write HTTP after hijack: %q", er.Raw)
	}
	select {
	case left := <-gotLeftover:
		if left != "PIPELINED" {
			t.Fatalf("leftover=%q", left)
		}
	default:
		t.Fatal("hijack callback did not run")
	}
}

func TestHijackWriteToRequiresCommit(t *testing.T) {
	t.Parallel()
	resp := http.Hijack(func(conn net.Conn, leftover []byte) error {
		return nil
	})
	err := resp.WriteTo(httptest.NewRecorder())
	if err == nil || !strings.Contains(err.Error(), "rawhttp Commit") {
		t.Fatalf("err=%v", err)
	}
}
