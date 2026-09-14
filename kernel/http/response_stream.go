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

// Hijack creates a response that takes over the underlying connection.
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
