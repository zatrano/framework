package http

import stdhttp "net/http"

// File creates a file-download response (attachment).
func File(path string) *Response {
	return &Response{
		status:   stdhttp.StatusOK,
		filePath: path,
		headers:  make(stdhttp.Header),
	}
}

// PublicFile serves a filesystem file inline (no attachment disposition).
// raw is the original net/http request so HEAD/Range are preserved.
func PublicFile(path string, raw *stdhttp.Request) *Response {
	return &Response{
		status:      stdhttp.StatusOK,
		filePath:    path,
		fileHTTPReq: raw,
		publicFile:  true,
		headers:     make(stdhttp.Header),
	}
}

// Hijack is the HTTP upgrade primitive (status 101). The callback receives the
// ResponseWriter and must type-assert net/http.Hijacker itself.
// HTTP/1.1 servers typically implement Hijacker; HTTP/2 usually does not.
// Frame protocols such as WebSocket live in packages/websocket; the kernel
// does not implement RFC 6455 and does not claim Hijack works on HTTP/2.
func Hijack(fn func(w stdhttp.ResponseWriter) error) *Response {
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
