package http

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	cold := r.cold
	if cold != nil && cold.clearWriteDeadline {
		if conn := ctx.Conn(); conn != nil {
			_ = conn.SetWriteDeadline(time.Time{})
		}
	}

	// Hijack takes the conn; RawHTTP writes no HTTP response afterward.
	if cold != nil && cold.hijack != nil {
		conn, leftover, err := ctx.Hijack()
		if err != nil {
			ctx.SetStatusCode(http.StatusInternalServerError)
			ctx.SetBodyString("hijacking not supported")
			return err
		}
		return cold.hijack(conn, leftover)
	}

	if r.reqIDOn {
		_ = ctx.SetHeaderBytes("X-Request-Id", r.reqID[:32])
	}
	if cold == nil || cold.headers == nil {
		for i := 0; i < r.nInline; i++ {
			_ = ctx.AddHeader(r.inlineHdrs[i].k, r.inlineHdrs[i].v)
		}
	} else {
		for key, values := range r.Headers() {
			for _, value := range values {
				_ = ctx.AddHeader(key, value)
			}
		}
	}
	if cold != nil {
		for _, c := range cold.cookies {
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
		if cold.redirectURL != "" {
			return ctx.Redirect(cold.redirectURL, r.StatusCode())
		}
		if cold.filePath != "" {
			if info, err := os.Stat(cold.filePath); err != nil {
				ctx.NotFound()
				return err
			} else if info.IsDir() {
				ctx.BadRequest()
				return fmt.Errorf("cannot serve directory: %s", cold.filePath)
			}
			if !cold.publicFile {
				_ = ctx.SetHeader("Content-Disposition", "attachment; filename="+filepath.Base(cold.filePath))
			}
			ctx.SetStatusCode(r.StatusCode())
			if r.contentType != "" {
				ctx.SetContentType(r.contentType)
			}
			ctx.SendFile(cold.filePath)
			return nil
		}
	}

	if r.contentType != "" {
		ctx.SetContentType(r.contentType)
	}
	ctx.SetStatusCode(r.StatusCode())
	if cold != nil && cold.streamReader != nil {
		if cold.chunkWriteTimeout > 0 || cold.clearWriteDeadline {
			reader := cold.streamReader
			rw := streamWriterFor(ctx, r)
			ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
				rw.buf = w
				_, _ = io.Copy(rw, reader)
				rw.Flush()
			})
			return nil
		}
		ctx.SetBodyStream(cold.streamReader, cold.streamSize)
		return nil
	}
	if cold != nil && cold.stream != nil {
		writer := cold.stream
		rw := streamWriterFor(ctx, r)
		ctx.SetBodyStreamWriter(func(w *bufio.Writer) {
			rw.buf = w
			rw.header = make(http.Header)
			rw.status = r.StatusCode()
			_ = writer(rw, rw)
			_ = w.Flush()
		})
		return nil
	}
	if r.textBody != "" {
		ctx.SetBodyString(r.textBody)
	} else if len(r.content) > 0 {
		ctx.SetBody(r.content)
	}
	return nil
}

// streamResponseWriter adapts rawhttp's bufio.Writer to the legacy
// StreamWriter(ResponseWriter, Flusher) signature used by SSE helpers.
// A positive chunkTimeout is installed on the conn before each write and flush.
// A negative chunkTimeout clears the write deadline (slow-reader risk).
type streamResponseWriter struct {
	buf          *bufio.Writer
	header       http.Header
	status       int
	conn         net.Conn
	chunkTimeout time.Duration
}

func streamWriterFor(ctx *rawhttp.Ctx, r *Response) *streamResponseWriter {
	rw := &streamResponseWriter{}
	if ctx != nil {
		rw.conn = ctx.Conn()
	}
	if r != nil && r.cold != nil {
		rw.chunkTimeout = r.cold.chunkWriteTimeout
		if r.cold.clearWriteDeadline {
			rw.chunkTimeout = -1
		}
	}
	return rw
}

func (s *streamResponseWriter) arm() {
	if s == nil || s.conn == nil || s.chunkTimeout == 0 {
		return
	}
	if s.chunkTimeout < 0 {
		_ = s.conn.SetWriteDeadline(time.Time{})
		return
	}
	_ = s.conn.SetWriteDeadline(time.Now().Add(s.chunkTimeout))
}

func (s *streamResponseWriter) Header() http.Header  { return s.header }
func (s *streamResponseWriter) WriteHeader(code int) { s.status = code }
func (s *streamResponseWriter) Write(p []byte) (int, error) {
	s.arm()
	return s.buf.Write(p)
}
func (s *streamResponseWriter) Flush() {
	s.arm()
	_ = s.buf.Flush()
}
