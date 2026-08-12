package http

import (
	"net/url"
	"strings"
)

// RedirectBack redirects to the Referer URL when safe, otherwise fallback.
func RedirectBack(req *Request, fallback string) *Response {
	target := fallback
	if fallback == "" {
		target = "/"
	}
	if req != nil {
		if ref := strings.TrimSpace(req.Header("Referer")); ref != "" {
			if safe := safeBackURL(req, ref); safe != "" {
				target = safe
			}
		}
	}
	return Redirect(target)
}

// Away redirects to an absolute URL (typically external).
func Away(url string) *Response {
	if url == "" {
		url = "/"
	}
	return Redirect(url)
}

// Refresh redirects to the current request URL.
func Refresh(req *Request) *Response {
	if req == nil {
		return Redirect("/")
	}
	target := strings.TrimSpace(req.FullURL())
	if target == "" {
		target = req.Path()
	}
	if target == "" {
		target = "/"
	}
	return Redirect(target)
}

// SecureRedirect redirects to an https URL for path on the request host.
func SecureRedirect(req *Request, path string) *Response {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	host := ""
	if req != nil {
		host = req.Host()
	}
	if host == "" {
		return Redirect("https://" + path)
	}
	return Redirect("https://" + host + path)
}

// PermanentRedirect creates a 301 redirect response.
func PermanentRedirect(url string) *Response {
	if url == "" {
		url = "/"
	}
	return Redirect(url, 301)
}

func safeBackURL(req *Request, ref string) string {
	u, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	if u.Scheme != "" || u.Host != "" {
		if req.Raw() == nil {
			return ""
		}
		host := req.Raw().Host
		if host == "" || !strings.EqualFold(u.Host, host) {
			return ""
		}
	}
	path := u.RequestURI()
	if path == "" {
		path = u.Path
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ""
	}
	return path
}
