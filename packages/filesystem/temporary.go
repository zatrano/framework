package filesystem

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// SetSigningKey configures HMAC signing for local temporary URLs.
func (d *LocalDisk) SetSigningKey(key string) {
	d.signingKey = key
}

// SetServePath sets the path used when building temporary URLs (default /storage/temporary).
func (d *LocalDisk) SetServePath(path string) {
	if path == "" {
		path = "/storage/temporary"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	d.servePath = path
}

// TemporaryURL builds a signed local temporary URL.
func (d *LocalDisk) TemporaryURL(path string, expires time.Duration) (string, error) {
	if strings.TrimSpace(d.signingKey) == "" {
		return "", fmt.Errorf("local disk signing key is not configured")
	}
	if expires <= 0 {
		expires = time.Hour
	}
	serve := d.servePath
	if serve == "" {
		serve = "/storage/temporary"
	}
	values := url.Values{}
	values.Set("path", strings.TrimPrefix(filepath.ToSlash(path), "/"))
	values.Set("expires", strconv.FormatInt(time.Now().Add(expires).Unix(), 10))
	unsigned := serve + "?" + values.Encode()
	values.Set("signature", signLocal(d.signingKey, unsigned))
	base := strings.TrimRight(d.baseURL, "/")
	if base == "" {
		return serve + "?" + values.Encode(), nil
	}
	// baseURL may already include /storage; prefer absolute app root style
	if strings.HasSuffix(base, "/storage") {
		return strings.TrimSuffix(base, "/storage") + serve + "?" + values.Encode(), nil
	}
	return base + serve + "?" + values.Encode(), nil
}

// HasValidTemporaryURL validates a temporary URL for this disk.
func (d *LocalDisk) HasValidTemporaryURL(rawURL string) bool {
	path, ok := d.ValidateTemporaryURL(rawURL)
	return ok && path != ""
}

// ValidateTemporaryURL returns the relative file path when the signed URL is valid.
func (d *LocalDisk) ValidateTemporaryURL(rawURL string) (string, bool) {
	if strings.TrimSpace(d.signingKey) == "" {
		return "", false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	values := parsed.Query()
	signature := values.Get("signature")
	path := values.Get("path")
	expires := values.Get("expires")
	if signature == "" || path == "" || expires == "" {
		return "", false
	}
	ts, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || time.Now().Unix() > ts {
		return "", false
	}
	check := url.Values{}
	check.Set("path", path)
	check.Set("expires", expires)
	serve := parsed.Path
	if serve == "" {
		serve = d.servePath
	}
	if serve == "" {
		serve = "/storage/temporary"
	}
	unsigned := serve + "?" + check.Encode()
	expected := signLocal(d.signingKey, unsigned)
	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return "", false
	}
	return path, true
}

// ServeTemporary returns middleware-compatible handler that serves a signed local file.
func ServeTemporary(disk *LocalDisk) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		if disk == nil {
			return http.Abort(500, "disk unavailable")
		}
		raw := req.URL()
		if u := req.Raw(); u != nil && u.URL != nil {
			raw = u.URL.RequestURI()
		}
		path, ok := disk.ValidateTemporaryURL(raw)
		if !ok {
			// also try path + query only
			if req.Raw() != nil && req.Raw().URL != nil {
				path, ok = disk.ValidateTemporaryURL(req.Raw().URL.Path + "?" + req.Raw().URL.RawQuery)
			}
		}
		if !ok {
			return http.Abort(403, "Invalid or expired temporary URL")
		}
		if !disk.Exists(path) {
			return http.Abort(404, "File not found")
		}
		return http.File(disk.Path(path)).Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, filepath.Base(path)))
	}
}

func signLocal(key, payload string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
