package bench

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
)

type serveFunc func(net.Conn) error

func TestTier0BodyEqual(t *testing.T) {
	z := mustCapture(t, serveZatranoTier0)
	if z.status != 200 || z.body != hello {
		t.Fatalf("zatrano tier-0 status=%d body=%q", z.status, z.body)
	}
	check := func(name string, serve serveFunc) {
		t.Helper()
		got := mustCapture(t, serve)
		if got.body != z.body || got.status != z.status || got.contentType != z.contentType {
			t.Errorf("%s tier-0 status=%d body=%q type=%q; zatrano status=%d body=%q type=%q",
				name, got.status, got.body, got.contentType, z.status, z.body, z.contentType)
		}
	}
	check("fiber", serveFiberTier0)
	check("gin", serveGinTier0)
	check("echo", serveEchoTier0)
}

func TestTier1HeaderBodyEqual(t *testing.T) {
	z, err := zatranoProbe()
	if err != nil {
		t.Fatal(err)
	}
	if z.status != 200 || z.body != hello {
		t.Fatalf("zatrano tier-1 status=%d body=%q", z.status, z.body)
	}
	if z.requestID != requestID {
		t.Fatalf("zatrano X-Request-ID=%q", z.requestID)
	}
	if len(z.headers) < 6 {
		t.Fatalf("zatrano tier-1 security/CORS headers = %d (%v), want at least 6", len(z.headers), z.headers)
	}
	check := func(name string, serve serveFunc) {
		t.Helper()
		got := mustCapture(t, serve)
		if got.body != z.body {
			t.Errorf("%s body %q", name, got.body)
		}
		if got.status != z.status {
			t.Errorf("%s status %d", name, got.status)
		}
		if got.contentType != z.contentType {
			t.Errorf("%s content-type %q, zatrano %q", name, got.contentType, z.contentType)
		}
		if got.requestID != z.requestID {
			t.Errorf("%s X-Request-ID %q, zatrano %q", name, got.requestID, z.requestID)
		}
		for key, value := range z.headers {
			if got.headers[key] != value {
				t.Errorf("%s %s=%q, zatrano %q", name, key, got.headers[key], value)
			}
		}
		for key, value := range got.headers {
			if _, ok := z.headers[key]; !ok {
				t.Errorf("%s extra security/CORS header %s=%q", name, key, value)
			}
		}
	}
	check("fiber", serveFiberTier1)
	check("gin", serveGinTier1)
	check("echo", serveEchoTier1)
	t.Logf("tier-1 security/CORS (%d): %v", len(z.headers), z.headers)
	t.Logf("content-type %s", z.contentType)
}

func TestTier1GeneratedID(t *testing.T) {
	z := mustCaptureReq(t, serveZatranoTier1Run, requestNoID)
	f := mustCaptureReq(t, serveFiberTier1Gen, requestNoID)
	if z.body != hello || f.body != hello {
		t.Fatalf("body z=%q f=%q", z.body, f.body)
	}
	if !hexIDPattern.MatchString(z.requestID) || !hexIDPattern.MatchString(f.requestID) {
		t.Fatalf("generated ids z=%q f=%q", z.requestID, f.requestID)
	}
	if z.requestID == requestID || f.requestID == requestID {
		t.Fatal("generated id reused the fixed echo value")
	}
	for key, value := range z.headers {
		if f.headers[key] != value {
			t.Errorf("fiber %s=%q, zatrano %q", key, f.headers[key], value)
		}
	}
	again := mustCaptureReq(t, serveZatranoTier1Run, requestNoID)
	if again.requestID == z.requestID {
		t.Fatalf("two generated ids matched: %s", z.requestID)
	}
}

func TestPipelinedThree(t *testing.T) {
	raw := append(append(append([]byte{}, request...), request...), request...)
	var buf bytes.Buffer
	c := &memConn{r: bytes.NewReader(raw), w: &buf}
	err := serveZatranoTier0(c)
	if buf.Len() == 0 {
		t.Fatal(err)
	}
	rest := buf.Bytes()
	n := 0
	for len(rest) > 0 {
		tail, perr := oneResponse(rest)
		if perr != nil {
			t.Fatalf("response %d: %v\n%s", n+1, perr, rest)
		}
		n++
		rest = tail
	}
	if n != 3 {
		t.Fatalf("responses=%d", n)
	}
}

func mustCapture(t *testing.T, serve serveFunc) captured {
	t.Helper()
	return mustCaptureReq(t, serve, request)
}

func mustCaptureReq(t *testing.T, serve serveFunc, req []byte) captured {
	t.Helper()
	c, buf := captureRequest(req)
	err := serve(c)
	if buf.Len() == 0 {
		t.Fatalf("empty response: %v", err)
	}
	got, rerr := readCaptured(buf)
	if rerr != nil {
		t.Fatalf("read response: %v (serve %v)\n%s", rerr, err, buf.Bytes())
	}
	return got
}

func oneResponse(raw []byte) ([]byte, error) {
	br := bufio.NewReader(bytes.NewReader(raw))
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 || string(body) != hello {
		return nil, fmt.Errorf("status %d body %q", resp.StatusCode, body)
	}
	tail, err := io.ReadAll(br)
	if err != nil {
		return nil, err
	}
	return tail, nil
}
