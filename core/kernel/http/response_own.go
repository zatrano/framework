package http

import (
	"maps"
	stdhttp "net/http"
)

// Own returns a new copy of resp that belongs to r. A handler may return one
// *Response for many requests (a cached page, a prebuilt error), and a
// middleware may keep the response its inner handler returned (a response
// cache), so the layer that receives a response must not write to it. The
// router calls Own at every middleware boundary and the kernel calls it before
// Commit; resp itself is only read. Own copies even when resp already belongs
// to r, because the layer that returned it may still hold it.
//
// The copy shares the body bytes, the stream and hijack callbacks, and the
// template data values with resp. Header maps, the cookie list, each cookie,
// and the template data map are copied. The shared body is never handed out:
// Content returns a copy.
func (r *Request) Own(resp *Response) *Response {
	if resp == nil || r == nil {
		return resp
	}
	var out *Response
	if !r.ownedUsed {
		r.ownedUsed = true
		out = &r.owned
	} else {
		out = new(Response)
	}
	*out = *resp
	out.owner = r
	if resp.cold != nil {
		c := *resp.cold
		if c.headers != nil {
			c.headers = c.headers.Clone()
		}
		if c.cookies != nil {
			cookies := make([]*stdhttp.Cookie, len(c.cookies))
			for i, ck := range c.cookies {
				if ck != nil {
					dup := *ck
					cookies[i] = &dup
				}
			}
			c.cookies = cookies
		}
		if c.templateData != nil {
			c.templateData = maps.Clone(c.templateData)
		}
		out.cold = &c
	}
	return out
}

// OwnedBy reports whether resp is a copy that belongs to r.
func (resp *Response) OwnedBy(r *Request) bool {
	return resp != nil && r != nil && resp.owner == r
}
