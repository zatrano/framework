package auth

import (
	"strings"

	"github.com/zatrano/framework/packages/http"
)

const intendedKey = "url.intended"

// SetIntendedURL stores a relative path for post-login redirects.
func SetIntendedURL(req *http.Request, path string) {
	path = sanitizeIntended(path)
	if path == "" {
		return
	}
	sess := req.Session()
	if sess == nil {
		return
	}
	sess.Put(intendedKey, path)
}

// IntendedURL returns the stored intended path without clearing it.
func IntendedURL(req *http.Request, fallback string) string {
	sess := req.Session()
	if sess == nil {
		return fallback
	}
	raw, _ := sess.Get(intendedKey).(string)
	if path := sanitizeIntended(raw); path != "" {
		return path
	}
	return fallback
}

// PullIntendedURL returns and clears the intended path.
func PullIntendedURL(req *http.Request, fallback string) string {
	sess := req.Session()
	if sess == nil {
		return fallback
	}
	raw, _ := sess.Pull(intendedKey).(string)
	if path := sanitizeIntended(raw); path != "" {
		return path
	}
	return fallback
}

// RedirectIntended redirects to the intended URL or fallback.
func RedirectIntended(req *http.Request, fallback string) *http.Response {
	return http.Redirect(PullIntendedURL(req, fallback))
}

// CaptureIntendedFromRequest stores the current GET path as the intended URL.
func CaptureIntendedFromRequest(req *http.Request) {
	if req == nil || req.Raw() == nil {
		return
	}
	if !strings.EqualFold(req.Method(), "GET") {
		return
	}
	u := req.Raw().URL
	path := u.Path
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}
	SetIntendedURL(req, path)
}

func sanitizeIntended(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return ""
	}
	if strings.ContainsAny(path, "\r\n") {
		return ""
	}
	return path
}
