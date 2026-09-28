package http

import (
	"fmt"
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
		raw, err := os.ReadFile(r.filePath)
		if err != nil {
			ctx.NotFound()
			return err
		}
		info, _ := os.Stat(r.filePath)
		if info != nil && info.IsDir() {
			ctx.BadRequest()
			return fmt.Errorf("cannot serve directory: %s", r.filePath)
		}
		if !r.publicFile && ctx.Header("Content-Disposition") == nil {
			_ = ctx.SetHeader("Content-Disposition", "attachment; filename="+filepath.Base(r.filePath))
		}
		ctx.SetStatusCode(r.StatusCode())
		if r.contentType != "" {
			ctx.SetContentType(r.contentType)
		}
		ctx.SetBody(raw)
		return nil
	}

	if r.contentType != "" {
		ctx.SetContentType(r.contentType)
	}
	ctx.SetStatusCode(r.StatusCode())
	if r.stream != nil {
		// Streaming via std Flusher is not available on rawhttp Ctx yet;
		// fall back to buffered body if content was prepared.
		if len(r.content) > 0 {
			ctx.SetBody(r.content)
		}
		return nil
	}
	if len(r.content) > 0 {
		ctx.SetBody(r.content)
	}
	return nil
}
