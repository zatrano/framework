package http

import (
	"bytes"
	"fmt"
	"net"
	stdhttp "net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/zatrano/rawhttp"
)

var harnessPort = atomic.Uint32{}

// ExchangeResult is one HTTP/1.1 response from ServeConnForTest / ExchangeForTest.
type ExchangeResult struct {
	Status int
	Header stdhttp.Header
	Body   []byte
	Raw    []byte // full response bytes
}

// ExchangeForTest runs one HTTP/1.1 exchange against h and parses the response.
func ExchangeForTest(h rawhttp.Handler, rawRequest string) (ExchangeResult, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return ExchangeResult{}, err
	}
	defer ln.Close()

	var serveErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, aerr := ln.Accept()
		if aerr != nil {
			serveErr = aerr
			return
		}
		defer conn.Close()
		s := &rawhttp.Server{Handler: h, KeepHijackedConns: true}
		_ = s.ServeConn(conn)
	}()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		return ExchangeResult{}, err
	}
	defer c.Close()
	if _, err := c.Write([]byte(rawRequest)); err != nil {
		return ExchangeResult{}, err
	}
	_ = c.(*net.TCPConn).CloseWrite()

	var buf bytes.Buffer
	tmp := make([]byte, 4096)
	for {
		n, rerr := c.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
		}
		if rerr != nil {
			break
		}
	}
	<-done
	if serveErr != nil {
		return ExchangeResult{Raw: buf.Bytes()}, serveErr
	}

	raw := buf.Bytes()
	return parseExchange(raw)
}

// ServeConnForTest runs one HTTP/1.1 exchange against h using rawhttp.
func ServeConnForTest(h rawhttp.Handler, rawRequest string) (status int, body []byte, err error) {
	er, err := ExchangeForTest(h, rawRequest)
	if err != nil {
		return er.Status, er.Body, err
	}
	return er.Status, er.Body, nil
}

func parseExchange(raw []byte) (ExchangeResult, error) {
	out := ExchangeResult{Raw: raw, Header: make(stdhttp.Header)}
	if len(raw) < 12 || !bytes.HasPrefix(raw, []byte("HTTP/1.")) {
		return out, fmt.Errorf("bad response: %q", raw)
	}
	sp := bytes.IndexByte(raw[8:], ' ')
	if sp < 0 {
		return out, fmt.Errorf("no status: %q", raw)
	}
	codeStart := 8 + sp + 1
	codeEnd := codeStart
	for codeEnd < len(raw) && raw[codeEnd] >= '0' && raw[codeEnd] <= '9' {
		codeEnd++
	}
	var code int
	for i := codeStart; i < codeEnd; i++ {
		code = code*10 + int(raw[i]-'0')
	}
	out.Status = code

	idx := bytes.Index(raw, []byte("\r\n\r\n"))
	if idx < 0 {
		return out, nil
	}
	headerBlock := raw[bytes.IndexByte(raw, '\n')+1 : idx]
	for _, line := range bytes.Split(headerBlock, []byte("\r\n")) {
		if len(line) == 0 {
			continue
		}
		colon := bytes.IndexByte(line, ':')
		if colon <= 0 {
			continue
		}
		key := string(bytes.TrimSpace(line[:colon]))
		val := string(bytes.TrimSpace(line[colon+1:]))
		out.Header.Add(key, val)
	}
	body := raw[idx+4:]
	if cl := out.Header.Get("Content-Length"); cl != "" {
		n, err := strconv.Atoi(strings.TrimSpace(cl))
		if err == nil && n >= 0 && n <= len(body) {
			body = body[:n]
		}
	}
	out.Body = body
	return out, nil
}
