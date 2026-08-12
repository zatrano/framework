package negotiate

import (
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const attrKey = "negotiated_format"

// Formats are common content negotiation targets.
const (
	FormatJSON = "json"
	FormatHTML = "html"
	FormatXML  = "xml"
	FormatText = "text"
	FormatAny  = "any"
)

// Negotiate picks the best format from Accept against offered formats (default json,html).
func Negotiate(req *http.Request, offered ...string) string {
	if len(offered) == 0 {
		offered = []string{FormatJSON, FormatHTML}
	}
	accept := req.Header("Accept")
	if accept == "" || accept == "*/*" {
		return offered[0]
	}
	parts := strings.Split(accept, ",")
	for _, part := range parts {
		media := strings.TrimSpace(strings.Split(part, ";")[0])
		media = strings.ToLower(media)
		for _, offer := range offered {
			if matches(media, offer) {
				return offer
			}
		}
	}
	return offered[0]
}

// Middleware stores the negotiated format on the request.
func Middleware(offered ...string) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			format := Negotiate(req, offered...)
			req.Set(attrKey, format)
			resp := next(req)
			if resp != nil {
				resp.Header("Vary", "Accept")
				resp.Header("X-Negotiated-Format", format)
			}
			return resp
		}
	}
}

// Format returns the negotiated format (or empty).
func Format(req *http.Request) string {
	v, _ := req.Get(attrKey).(string)
	return v
}

// Wants reports whether the negotiated format equals name.
func Wants(req *http.Request, name string) bool {
	return strings.EqualFold(Format(req), name)
}

func matches(media, offer string) bool {
	switch offer {
	case FormatJSON:
		return media == "application/json" || strings.HasSuffix(media, "+json") || media == "text/json"
	case FormatHTML:
		return media == "text/html" || media == "application/xhtml+xml"
	case FormatXML:
		return media == "application/xml" || media == "text/xml" || strings.HasSuffix(media, "+xml")
	case FormatText:
		return media == "text/plain"
	case FormatAny:
		return true
	default:
		return media == offer
	}
}
