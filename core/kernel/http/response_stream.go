package http

import (
	"net"
	stdhttp "net/http"
	"time"
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
		status:             101,
		hijack:             fn,
		headers:            make(stdhttp.Header),
		clearWriteDeadline: true,
	}
}

// ClearWriteDeadline drops the server write deadline before this response is
// written. The caller accepts the slow-reader risk: a peer that stops reading
// can hold the connection open. Hijack always clears the deadline. Stream and
// StreamBody re-arm now+HTTP_WRITE_TIMEOUT before each write and flush unless
// this is called or the write timeout is disabled. The read deadline is unchanged.
func (r *Response) ClearWriteDeadline() *Response {
	if r == nil {
		return nil
	}
	r.clearWriteDeadline = true
	r.chunkWriteTimeout = 0
	return r
}

// PrepareStreamDeadline arms a streaming response from the server write timeout.
// A positive duration is re-applied before each write and flush. Zero or
// negative clears the deadline (the timeout is off). Hijack and an explicit
// ClearWriteDeadline are left unchanged. Non-stream responses are ignored.
func (r *Response) PrepareStreamDeadline(serverWrite time.Duration) {
	if r == nil || !r.IsStream() || r.hijack != nil || r.clearWriteDeadline {
		return
	}
	if serverWrite <= 0 {
		r.clearWriteDeadline = true
		return
	}
	r.chunkWriteTimeout = serverWrite
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
