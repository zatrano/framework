package http

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/zatrano/rawhttp"
)

// Commit writes the response onto a rawhttp context.
func (r *Response) Commit(ctx *rawhttp.Ctx) error {
	if ctx == nil {
		return nil
	}
	if r == nil {
		ctx.SetStatusCode(204)
		return nil
	}

	// Hijack takes the conn; RawHTTP writes no HTTP response afterward.
	if r.hijack != nil {
		conn, leftover, err := ctx.Hijack()
		if err != nil {
			ctx.SetStatusCode(http.StatusInternalServerError)
			ctx.SetBodyString("hijacking not supported")
			return err
		}
		return r.hijack(conn, leftover)
	}

	for key, values := range r.Headers() {
		for _, value := range values {
			_ = ctx.AddHeader(key, value)
		}
	}
	for _, c := range r.cookies {
		if c == nil {
			continue
		}
		rc := rawhttp.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Path:     c.Path,
			Domain:   c.Domain,
			MaxAge:   c.MaxAge,
			Expires:  c.Expires,
			Secure:   c.Secure,
			HTTPOnly: c.HttpOnly,
			SameSite: rawhttp.SameSite(c.SameSite),
		}
		ctx.SetCookie(&rc)
	}

	if r.redirectURL != "" {
		return ctx.Redirect(r.redirectURL, r.StatusCode())
	}

	if r.filePath != "" {
		if info, err := os.Stat(r.filePath); err != nil {
			ctx.NotFound()
			return err
		} else if info.IsDir() {
			ctx.BadRequest()
			return fmt.Errorf("cannot serve directory: %s", r.filePath)
		}
		if !r.publicFile {
			_ = ctx.SetHeader("Content-Disposition", "attachment; filename="+filepath.Base(r.filePath))
		}
		ctx.SetStatusCode(r.StatusCode())
		if r.contentType != "" {
			ctx.SetContentType(r.contentType)
		}
		// Stream from disk via RawHTTP (no full ReadFile into memory).
		ctx.SendFile(r.filePath)
		return nil
	}

	if r.contentType != "" {
		ctx.SetContentType(r.contentType)
	}
	ctx.SetStatusCode(r.StatusCode())
	if r.streamReader != nil {
		ctx.SetBodyStream(r.streamReader, r.streamSize)
		return nil
	}
	if r.stream != nil {
		writer := r.stream
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			rw := &streamResponseWriter{buf: w, header: make(http.Header), status: r.StatusCode()}
			_ = writer(rw, rw)
			_ = w.Flush()
		})
		return nil
	}
	if len(r.content) > 0 {
		ctx.SetBody(r.content)
	}
	return nil
}

// streamResponseWriter adapts rawhttp's bufio.Writer to the legacy
// StreamWriter(ResponseWriter, Flusher) signature used by SSE helpers.
type streamResponseWriter struct {
	buf    *bufio.Writer
	header http.Header
	status int
}

func (s *streamResponseWriter) Header() http.Header         { return s.header }
func (s *streamResponseWriter) WriteHeader(code int)        { s.status = code }
func (s *streamResponseWriter) Write(p []byte) (int, error) { return s.buf.Write(p) }
func (s *streamResponseWriter) Flush()                      { _ = s.buf.Flush() }
