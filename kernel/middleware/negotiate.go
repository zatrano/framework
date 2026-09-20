package middleware

import (
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

// Negotiate stores the Accept-selected format on the request and sets Vary.
func Negotiate(offered ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			format := http.Negotiate(req, offered...)
			req.Set(http.AttrNegotiatedFormat, format)
			resp := next(req)
			if resp != nil {
				resp.Header("Vary", "Accept")
				resp.Header("X-Negotiated-Format", format)
			}
			return resp
		}
	}
}
