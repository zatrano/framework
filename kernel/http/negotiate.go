package http

import "strings"

// AttrNegotiatedFormat is the request attribute set by middleware.Negotiate.
const AttrNegotiatedFormat = "negotiated_format"

// Common content-negotiation format names.
const (
	FormatJSON = "json"
	FormatHTML = "html"
	FormatXML  = "xml"
	FormatText = "text"
	FormatAny  = "any"
)

// Negotiate picks the best offered format from Accept via Request.Prefers.
// Empty Accept, */*, or no match returns the first offered value
// (default json, then html). Higher q wins; q=0 is not acceptable.
func Negotiate(req *Request, offered ...string) string {
	if len(offered) == 0 {
		offered = []string{FormatJSON, FormatHTML}
	}
	if req == nil {
		return offered[0]
	}
	if got := req.Prefers(offered...); got != "" {
		return got
	}
	return offered[0]
}

// NegotiatedFormat returns the format stored by middleware.Negotiate.
func NegotiatedFormat(req *Request) string {
	if req == nil {
		return ""
	}
	v, _ := req.Get(AttrNegotiatedFormat).(string)
	return v
}

// WantsFormat reports whether the stored negotiated format equals name.
func WantsFormat(req *Request, name string) bool {
	return strings.EqualFold(NegotiatedFormat(req), name)
}
