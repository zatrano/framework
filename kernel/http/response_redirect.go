package http

import stdhttp "net/http"

// Redirect creates a redirect response.
func Redirect(url string, status ...int) *Response {
	code := stdhttp.StatusFound
	if len(status) > 0 {
		code = status[0]
	}
	return &Response{
		status:      code,
		redirectURL: url,
		headers:     make(stdhttp.Header),
	}
}

// Found creates a 302 redirect.
func Found(url string) *Response {
	return Redirect(url, stdhttp.StatusFound)
}

// RedirectRoute redirects to a path produced for a named route (caller supplies path).
// Prefer routing.Router.RedirectRoute when a router is available.
func RedirectRoute(path string, status ...int) *Response {
	if path == "" {
		path = "/"
	}
	return Redirect(path, status...)
}

// SeeOther creates a 303 redirect.
func SeeOther(url string) *Response {
	return Redirect(url, stdhttp.StatusSeeOther)
}

// TemporaryRedirect creates a 307 redirect.
func TemporaryRedirect(url string) *Response {
	return Redirect(url, stdhttp.StatusTemporaryRedirect)
}
