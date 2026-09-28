package http

import (
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"path/filepath"
	"strings"
)

func contentDisposition(kind, filename string) string {
	filename = strings.ReplaceAll(filename, `"`, `'`)
	return fmt.Sprintf(`%s; filename="%s"`, kind, filename)
}

// Download serves a file as an attachment with an optional filename.
func Download(path, filename string) *Response {
	if filename == "" {
		filename = filepath.Base(path)
	}
	return File(path).Header("Content-Disposition", contentDisposition("attachment", filename))
}

// DownloadBytes serves in-memory bytes as a downloadable file.
func DownloadBytes(content []byte, filename, contentType string) *Response {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if filename == "" {
		filename = "download"
	}
	return (&Response{
		status:      200,
		content:     content,
		contentType: contentType,
		headers:     make(stdhttp.Header),
	}).Header("Content-Disposition", contentDisposition("attachment", filename))
}

// DownloadJSON marshals data and serves it as a JSON attachment.
func DownloadJSON(data any, filename string) *Response {
	raw, err := json.Marshal(data)
	if err != nil {
		raw = []byte("null")
	}
	if filename == "" {
		filename = "data.json"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".json") {
		filename += ".json"
	}
	return DownloadBytes(raw, filename, "application/json")
}

// InlineFile serves a file for inline display in the browser.
func InlineFile(path string) *Response {
	return File(path).Header("Content-Disposition", contentDisposition("inline", filepath.Base(path)))
}

// InlineBytes serves in-memory bytes for inline display.
func InlineBytes(content []byte, filename, contentType string) *Response {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if filename == "" {
		filename = "file"
	}
	return (&Response{
		status:      200,
		content:     content,
		contentType: contentType,
		headers:     make(stdhttp.Header),
	}).Header("Content-Disposition", contentDisposition("inline", filename))
}

// AsDownload marks the response as an attachment download.
func (r *Response) AsDownload(filename string) *Response {
	if r == nil {
		return nil
	}
	if filename == "" {
		filename = "download"
	}
	return r.Header("Content-Disposition", contentDisposition("attachment", filename))
}

// AsInline marks the response for inline browser display.
func (r *Response) AsInline(filename string) *Response {
	if r == nil {
		return nil
	}
	if filename == "" {
		filename = "file"
	}
	return r.Header("Content-Disposition", contentDisposition("inline", filename))
}
