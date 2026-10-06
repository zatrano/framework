package http

import (
	"net/textproto"
	"strings"
)

// headerKV is one response header stored before Headers builds a map.
type headerKV struct {
	name  string
	value string
}

// AppendFixed appends already-canonical name/value pairs into the inline list.
// Callers validate the pairs once at boot. This does not build a header map.
func (r *Response) AppendFixed(pairs [][2]string) {
	if r == nil || len(pairs) == 0 {
		return
	}
	for i := range pairs {
		r.appendRaw(pairs[i][0], pairs[i][1])
	}
}

// MustSafeHeaders panics when a boot-time header name or value contains CR, LF, or NUL.
func MustSafeHeaders(pairs [][2]string) {
	for _, pair := range pairs {
		if headerTextUnsafe(pair[0]) || headerTextUnsafe(pair[1]) {
			panic("http: unsafe header " + pair[0])
		}
	}
}

func headerTextUnsafe(s string) bool {
	return strings.IndexByte(s, '\r') >= 0 || strings.IndexByte(s, '\n') >= 0 || strings.IndexByte(s, 0) >= 0
}

func (r *Response) setHeader(key, value string, replace bool) {
	if r.headers != nil {
		if replace {
			r.headers.Set(key, value)
		} else {
			r.headers.Add(key, value)
		}
		return
	}
	name := canonicalResponseHeader(key)
	if replace {
		r.deleteList(name)
	}
	r.appendRaw(name, value)
}

func (r *Response) deleteList(name string) {
	n := 0
	for i := 0; i < r.headerLen; i++ {
		if r.headerList[i].name == name {
			continue
		}
		r.headerList[n] = r.headerList[i]
		n++
	}
	r.headerLen = n
	if len(r.headerMore) == 0 {
		return
	}
	dst := r.headerMore[:0]
	for _, pair := range r.headerMore {
		if pair.name != name {
			dst = append(dst, pair)
		}
	}
	r.headerMore = dst
}

func (r *Response) appendRaw(name, value string) {
	if r.headers != nil {
		r.headers.Add(name, value)
		return
	}
	if r.headerLen < len(r.headerList) {
		r.headerList[r.headerLen] = headerKV{name: name, value: value}
		r.headerLen++
		return
	}
	r.headerMore = append(r.headerMore, headerKV{name: name, value: value})
}

func (r *Response) writeHeaders(ctx interface{ AddHeader(string, string) error }) {
	if r == nil {
		return
	}
	if r.headers != nil {
		for key, values := range r.headers {
			for _, value := range values {
				_ = ctx.AddHeader(key, value)
			}
		}
		return
	}
	for i := 0; i < r.headerLen; i++ {
		_ = ctx.AddHeader(r.headerList[i].name, r.headerList[i].value)
	}
	for _, pair := range r.headerMore {
		_ = ctx.AddHeader(pair.name, pair.value)
	}
}

// canonicalResponseHeader returns the net/http canonical spelling.
// Names the framework writes on the hot path skip CanonicalMIMEHeaderKey,
// but they still collapse to that spelling. X-Request-ID, ETag, and
// X-RateLimit-Limit are not MIME-canonical; keeping the raw spelling
// split Add and Del across two list entries.
func canonicalResponseHeader(key string) string {
	switch key {
	case "X-Request-ID", "X-Request-Id":
		return "X-Request-Id"
	case "ETag", "Etag":
		return "Etag"
	case "X-CSRF-TOKEN", "X-Csrf-Token":
		return "X-Csrf-Token"
	case "X-RateLimit-Limit", "X-Ratelimit-Limit":
		return "X-Ratelimit-Limit"
	case "X-RateLimit-Remaining", "X-Ratelimit-Remaining":
		return "X-Ratelimit-Remaining"
	case "X-Frame-Options",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Permissions-Policy",
		"Strict-Transport-Security",
		"Traceparent",
		"Vary",
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Expose-Headers",
		"Access-Control-Allow-Credentials",
		"Access-Control-Max-Age",
		"Cache-Control",
		"Pragma",
		"Expires",
		"Content-Type",
		"Content-Disposition",
		"Location",
		"Allow",
		"Retry-After",
		"Last-Modified",
		"X-Negotiated-Format",
		"Content-Language",
		"Content-Length":
		return key
	default:
		return textproto.CanonicalMIMEHeaderKey(key)
	}
}

// HeaderMapBuilt reports whether Headers has moved the inline list into a map.
func (r *Response) HeaderMapBuilt() bool {
	return r != nil && r.headers != nil
}
