// Package bench measures ZATRANO and rival frameworks on one connection,
// without a TCP accept loop. Tier-0 is the router only. Tier-1 is the booted
// kernel stack. Fiber tier-1 copies the security and CORS headers, and
// X-Request-ID, from a ZATRANO probe of the same process.
package bench

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const (
	hello     = "Hello, World!"
	requestID = "bench-fixed-id"
)

// idPattern matches the kernel request-id grammar.
var idPattern = regexp.MustCompile(`^[A-Za-z0-9._\-:]{1,128}$`)

// hexIDPattern is the production generator: 16 crypto/rand bytes, hex encoded.
var hexIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func newHexID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000000000000000000000000000"
	}
	return hex.EncodeToString(b[:])
}

// request is one HTTP/1.1 plaintext GET carrying a valid X-Request-ID.
// requestNoID is the production shape: the server generates the id.
var (
	request     = []byte("GET /plaintext HTTP/1.1\r\nHost: bench\r\nX-Request-ID: " + requestID + "\r\nConnection: keep-alive\r\n\r\n")
	requestNoID = []byte("GET /plaintext HTTP/1.1\r\nHost: bench\r\nConnection: keep-alive\r\n\r\n")
)

// securityCORS are the headers Fiber must copy from the ZATRANO tier-1 probe.
var securityCORS = []string{
	"X-Content-Type-Options",
	"X-Frame-Options",
	"Referrer-Policy",
	"Permissions-Policy",
	"Strict-Transport-Security",
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Credentials",
	"Access-Control-Expose-Headers",
	"Access-Control-Max-Age",
}

var benchAddr = &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9}

// memConn is an in-memory net.Conn. Reads return the pipelined request bytes
// and then EOF. Writes are kept when w is non-nil, otherwise discarded.
type memConn struct {
	r *bytes.Reader
	w *bytes.Buffer
}

func (c *memConn) Read(p []byte) (int, error) { return c.r.Read(p) }

func (c *memConn) Write(p []byte) (int, error) {
	if c.w == nil {
		return len(p), nil
	}
	return c.w.Write(p)
}

func (c *memConn) Close() error                     { return nil }
func (c *memConn) LocalAddr() net.Addr              { return benchAddr }
func (c *memConn) RemoteAddr() net.Addr             { return benchAddr }
func (c *memConn) SetDeadline(time.Time) error      { return nil }
func (c *memConn) SetReadDeadline(time.Time) error  { return nil }
func (c *memConn) SetWriteDeadline(time.Time) error { return nil }

// countConn counts deadline arms. The benchmark path does not use it.
type countConn struct {
	*memConn
	readDL  int
	writeDL int
}

func (c *countConn) SetReadDeadline(time.Time) error {
	c.readDL++
	return nil
}
func (c *countConn) SetWriteDeadline(time.Time) error {
	c.writeDL++
	return nil
}

func discardConn(n int) *memConn {
	return &memConn{r: bytes.NewReader(bytes.Repeat(request, n))}
}

func captureConn() (*memConn, *bytes.Buffer) {
	return captureRequest(request)
}

func captureRequest(req []byte) (*memConn, *bytes.Buffer) {
	var buf bytes.Buffer
	return &memConn{r: bytes.NewReader(req), w: &buf}, &buf
}

type captured struct {
	status      int
	body        string
	contentType string
	requestID   string
	headers     map[string]string
}

func readCaptured(buf *bytes.Buffer) (captured, error) {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(buf.Bytes())), nil)
	if err != nil {
		return captured{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return captured{}, err
	}
	out := captured{
		status:      resp.StatusCode,
		body:        string(body),
		contentType: resp.Header.Get("Content-Type"),
		requestID:   resp.Header.Get("X-Request-ID"),
		headers:     map[string]string{},
	}
	for _, name := range securityCORS {
		if v := resp.Header.Get(name); v != "" {
			out.headers[name] = v
		}
	}
	return out, nil
}

// tier1Probe is the ZATRANO tier-1 response this process must match.
var (
	probeOnce sync.Once
	probeVal  captured
	probeErr  error
)

func zatranoProbe() (captured, error) {
	probeOnce.Do(func() {
		c, buf := captureConn()
		err := serveZatranoTier1(c)
		if buf.Len() == 0 {
			probeErr = errString("zatrano tier-1 wrote no response", err)
			return
		}
		probeVal, probeErr = readCaptured(buf)
	})
	return probeVal, probeErr
}

type strError string

func (e strError) Error() string { return string(e) }

func errString(msg string, err error) error {
	if err == nil {
		return strError(msg)
	}
	return strError(msg + ": " + err.Error())
}
