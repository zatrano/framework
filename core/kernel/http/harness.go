package http

import (
	"bytes"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/zatrano/rawhttp"
)

var harnessPort = atomic.Uint32{}

// ServeConnForTest runs one HTTP/1.1 exchange against h using rawhttp.
func ServeConnForTest(h rawhttp.Handler, rawRequest string) (status int, body []byte, err error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, err
	}
	defer ln.Close()

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, aerr := ln.Accept()
		if aerr != nil {
			err = aerr
			return
		}
		defer conn.Close()
		s := &rawhttp.Server{Handler: h}
		_ = s.ServeConn(conn)
	}()

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		return 0, nil, err
	}
	defer c.Close()
	if _, err := c.Write([]byte(rawRequest)); err != nil {
		return 0, nil, err
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
	resp := buf.Bytes()
	// minimal status parse
	if len(resp) < 12 || !bytes.HasPrefix(resp, []byte("HTTP/1.")) {
		return 0, resp, fmt.Errorf("bad response: %q", resp)
	}
	sp := bytes.IndexByte(resp[8:], ' ')
	if sp < 0 {
		return 0, resp, fmt.Errorf("no status: %q", resp)
	}
	codeStart := 8 + sp + 1
	codeEnd := codeStart
	for codeEnd < len(resp) && resp[codeEnd] >= '0' && resp[codeEnd] <= '9' {
		codeEnd++
	}
	var code int
	for i := codeStart; i < codeEnd; i++ {
		code = code*10 + int(resp[i]-'0')
	}
	idx := bytes.Index(resp, []byte("\r\n\r\n"))
	if idx < 0 {
		return code, nil, nil
	}
	return code, resp[idx+4:], nil
}
