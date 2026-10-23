package http

import (
	"strings"

	"github.com/zatrano/rawhttp"
)

// hdrSpan locates one request header inside the arena.
type hdrSpan struct{ ko, kl, vo, vl uint32 }

const inlineHeaderSpans = 16

// carrier is the request as it was parsed, copied out of the connection
// buffer. Every string here is a substring of arena, which is built once and
// never written again, so a value returned to the caller cannot change after
// the handler returns and the connection reads its next request.
type carrier struct {
	arena  string
	method string
	path   string
	query  string
	host   string
	spans  []hdrSpan
	inline [inlineHeaderSpans]hdrSpan

	body     []byte
	peerAddr string
	peerIP   string
	tls      bool
}

func (c *carrier) fill(ctx *rawhttp.Ctx) {
	size := len(ctx.Method) + len(ctx.Path) + len(ctx.Query)
	n := 0
	ctx.VisitHeader(func(k, v []byte) {
		size += len(k) + len(v)
		n++
	})
	var b strings.Builder
	b.Grow(size)
	b.Write(ctx.Method)
	pathOff := b.Len()
	b.Write(ctx.Path)
	queryOff := b.Len()
	b.Write(ctx.Query)
	hdrOff := b.Len()

	spans := c.inline[:0]
	if n > inlineHeaderSpans {
		spans = make([]hdrSpan, 0, n)
	}
	hostIdx := -1
	ctx.VisitHeader(func(k, v []byte) {
		v = trimOWS(v)
		ko := b.Len()
		b.Write(k)
		vo := b.Len()
		b.Write(v)
		if len(k) == 4 && asciiEqualFoldBytes(k, "Host") {
			hostIdx = len(spans)
		}
		spans = append(spans, hdrSpan{ko: uint32(ko), kl: uint32(len(k)), vo: uint32(vo), vl: uint32(len(v))})
	})

	c.arena = b.String()
	c.method = commonMethod(ctx.Method)
	if c.method == "" {
		c.method = c.arena[:pathOff]
	}
	c.path = c.arena[pathOff:queryOff]
	c.query = c.arena[queryOff:hdrOff]
	c.spans = spans
	if hostIdx >= 0 {
		s := spans[hostIdx]
		c.host = c.arena[s.vo : s.vo+s.vl]
	}

	if raw := ctx.Body(); len(raw) > 0 {
		c.body = append([]byte(nil), raw...)
	}
	c.peerAddr = ctx.RemoteAddr()
	c.peerIP = ctx.RemoteIP()
	c.tls = ctx.IsTLS()
}

func (c *carrier) key(s hdrSpan) string { return c.arena[s.ko : s.ko+s.kl] }

func (c *carrier) value(s hdrSpan) string { return c.arena[s.vo : s.vo+s.vl] }

// header returns the value rawhttp's Ctx.Header would return for key. rawhttp
// indexes a few common headers and keeps the last occurrence of those; any
// other name returns the first occurrence.
func (c *carrier) header(key string) string {
	last := indexedHeader(key)
	out := ""
	for _, s := range c.spans {
		if int(s.kl) != len(key) || !asciiEqualFold(c.key(s), key) {
			continue
		}
		out = c.value(s)
		if !last {
			return out
		}
	}
	return out
}

// headerValues returns the value of every key line, in request order.
func (c *carrier) headerValues(key string) []string {
	var out []string
	for _, s := range c.spans {
		if int(s.kl) == len(key) && asciiEqualFold(c.key(s), key) {
			out = append(out, c.value(s))
		}
	}
	return out
}

func (c *carrier) visitHeaders(f func(k, v string)) {
	for _, s := range c.spans {
		f(c.key(s), c.value(s))
	}
}

// cookie matches rawhttp's Ctx.Cookie over the copied Cookie header.
func (c *carrier) cookie(name string) (string, bool) {
	if name == "" {
		return "", false
	}
	found := false
	out := ""
	c.visitCookies(func(k, v string, hasValue bool) bool {
		if len(k) == len(name) && asciiEqualFold(k, name) {
			out, found = v, hasValue
			return false
		}
		return true
	})
	return out, found
}

// visitCookies walks every Cookie header line in order, the way net/http
// does. A pair without "=" reports hasValue false.
func (c *carrier) visitCookies(f func(k, v string, hasValue bool) bool) {
	for _, s := range c.spans {
		if s.kl == 6 && asciiEqualFold(c.key(s), "Cookie") {
			if !visitCookieLine(c.value(s), f) {
				return
			}
		}
	}
}

func visitCookieLine(raw string, f func(k, v string, hasValue bool) bool) bool {
	for raw != "" {
		part := raw
		if i := strings.IndexByte(raw, ';'); i >= 0 {
			part, raw = raw[:i], raw[i+1:]
		} else {
			raw = ""
		}
		part = strings.TrimLeft(part, " \t")
		k, v, hasValue := part, "", false
		if eq := strings.IndexByte(part, '='); eq >= 0 {
			k, v, hasValue = part[:eq], part[eq+1:], true
		}
		if k == "" {
			continue
		}
		if !f(k, v, hasValue) {
			return false
		}
	}
	return true
}

func indexedHeader(key string) bool {
	switch len(key) {
	case 4:
		return asciiEqualFold(key, "Host")
	case 6:
		return asciiEqualFold(key, "Accept") || asciiEqualFold(key, "Cookie") || asciiEqualFold(key, "Origin")
	case 7:
		return asciiEqualFold(key, "Referer")
	case 10:
		return asciiEqualFold(key, "User-Agent")
	case 12:
		return asciiEqualFold(key, "Content-Type")
	case 13:
		return asciiEqualFold(key, "Authorization")
	case 15:
		return asciiEqualFold(key, "Accept-Encoding")
	}
	return false
}

func commonMethod(b []byte) string {
	switch string(b) {
	case "GET":
		return "GET"
	case "POST":
		return "POST"
	case "PUT":
		return "PUT"
	case "HEAD":
		return "HEAD"
	case "PATCH":
		return "PATCH"
	case "DELETE":
		return "DELETE"
	case "OPTIONS":
		return "OPTIONS"
	}
	return ""
}

func trimOWS(v []byte) []byte {
	for len(v) > 0 && (v[0] == ' ' || v[0] == '\t') {
		v = v[1:]
	}
	for len(v) > 0 && (v[len(v)-1] == ' ' || v[len(v)-1] == '\t') {
		v = v[:len(v)-1]
	}
	return v
}

func asciiLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

func asciiEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] && asciiLower(a[i]) != asciiLower(b[i]) {
			return false
		}
	}
	return true
}

func asciiEqualFoldBytes(a []byte, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] && asciiLower(a[i]) != asciiLower(b[i]) {
			return false
		}
	}
	return true
}
