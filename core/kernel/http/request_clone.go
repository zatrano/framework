package http

import (
	"context"
	stdhttp "net/http"
)

// Clone returns a deep copy that stays valid after the handler returns.
// It copies the method, path, query, host, remote address, scheme, headers,
// cookies, route params, and body. The copy does not alias the carrier
// buffer and is not stored in the request pool. Attributes are shallow-copied.
//
// Call Clone before returning. On a poison build the original panics if it
// is used after the handler returns; the clone does not.
//
//	clone := req.Clone()
//	go func() {
//	    path := clone.Path()
//	    id, _ := clone.HeaderValue("X-Request-ID")
//	    _ = path
//	    _ = id
//	}()
func (r *Request) Clone() *Request {
	r.poisonCheck()
	if r == nil {
		return nil
	}
	c := &Request{stdCtx: r.Context()}
	if c.stdCtx == nil {
		c.stdCtx = context.Background()
	}
	if method := r.Method(); method != "" {
		c.method = method
		c.methodSet = true
	}
	c.path = r.Path()
	c.pathSet = true
	c.query = r.QueryString()
	c.querySet = true
	c.host = r.Host()
	if ip := r.RemoteIP(); ip != "" {
		c.remoteAddr = ip
	}
	c.secure = r.Secure()
	c.secureSet = true

	headers := r.HeadersMap()
	if len(headers) > 0 {
		c.headerOverlay = make(stdhttp.Header, len(headers))
		for key, value := range headers {
			c.headerOverlay.Set(key, value)
		}
	}
	if cookieHeader := headers["Cookie"]; cookieHeader != "" {
		parsed := &stdhttp.Request{Header: stdhttp.Header{"Cookie": {cookieHeader}}}
		for _, cookie := range parsed.Cookies() {
			c.SetCookie(cookie.Name, cookie.Value)
		}
	}
	if r.cookieOverlay != nil {
		for name, value := range r.cookieOverlay {
			c.SetCookie(name, value)
		}
	}
	if body, err := r.Body(); err == nil && len(body) > 0 {
		c.SetBody(append([]byte(nil), body...))
	}
	if params := r.RouteParams(); len(params) > 0 {
		c.route = make(map[string]string, len(params))
		for key, value := range params {
			c.route[key] = value
		}
	}
	if r.attrs != nil {
		c.attrs = make(map[string]any, len(r.attrs))
		for key, value := range r.attrs {
			c.attrs[key] = value
		}
	}
	return c
}
