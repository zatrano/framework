package http

import (
	"net"
	stdhttp "net/http"
)

// File creates a file-download response (attachment).
func File(path string) *Response {
	return &Response{
		status:   stdhttp.StatusOK,
		filePath: path,
		headers:  make(stdhttp.Header),
	}
}

// PublicFile serves a filesystem file inline (no attachment disposition).
// Commit streams via rawhttp.SendFile (If-Modified-Since handled by RawHTTP).
func PublicFile(path string) *Response {
	return &Response{
		status:     stdhttp.StatusOK,
		filePath:   path,
		publicFile: true,
		headers:    make(stdhttp.Header),
	}
}

// HijackFunc takes ownership of the raw connection after rawhttp.Ctx.Hijack.
// leftover is any unread buffered bytes; read them before reading from conn.
// The server does not write an HTTP response after Hijack — write the upgrade
// handshake (e.g. 101) yourself. Frame protocols such as WebSocket live in
// packages/websocket; the kernel does not implement RFC 6455.
type HijackFunc func(conn net.Conn, leftover []byte) error

// Hijack is the HTTP upgrade primitive. Commit calls ctx.Hijack() then fn.
// Application.Run sets Server.KeepHijackedConns so the accept loop does not
// Close the connection after the handler returns.
func Hijack(fn HijackFunc) *Response {
	return &Response{
		status:  101,
		hijack:  fn,
		headers: make(stdhttp.Header),
	}
}

// PartialContent creates a 206 response with raw content.
func PartialContent(content []byte, contentType string) *Response {
	return Bytes(content, contentType).Status(stdhttp.StatusPartialContent)
}

// StreamDownload streams content with an attachment disposition.
func StreamDownload(filename string, writer StreamWriter) *Response {
	resp := Stream("application/octet-stream", writer)
	return resp.AsDownload(filename)
}
