package url

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// SetSigningKey configures the HMAC key used for signed URLs.
func (g *Generator) SetSigningKey(key string) {
	g.signingKey = key
}

// Signed builds a signed absolute URL that expires after duration.
func (g *Generator) Signed(path string, expiresIn time.Duration, query ...map[string]string) (string, error) {
	if g.signingKey == "" {
		return "", fmt.Errorf("signing key is not configured")
	}

	values := url.Values{}
	if len(query) > 0 {
		for key, value := range query[0] {
			values.Set(key, value)
		}
	}
	expires := time.Now().Add(expiresIn).Unix()
	values.Set("expires", strconv.FormatInt(expires, 10))

	unsigned := g.absoluteWithQuery(path, values)
	signature := sign(g.signingKey, unsigned)
	values.Set("signature", signature)
	return g.absoluteWithQuery(path, values), nil
}

// TemporarySignedRoute signs a named route URL.
func (g *Generator) TemporarySignedRoute(name string, expiresIn time.Duration, params map[string]string, query ...map[string]string) (string, error) {
	path, err := g.router.URL(name, params)
	if err != nil {
		return "", err
	}
	return g.Signed(path, expiresIn, query...)
}

// HasValidSignature validates signature and expiry of a full URL or path+query.
func (g *Generator) HasValidSignature(rawURL string) bool {
	if g.signingKey == "" {
		return false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	if parsed.Scheme == "" {
		absolute := g.To(parsed.Path)
		u2, err := url.Parse(absolute)
		if err != nil {
			return false
		}
		u2.RawQuery = parsed.RawQuery
		u2.Fragment = parsed.Fragment
		parsed = u2
	}

	values := parsed.Query()
	signature := values.Get("signature")
	if signature == "" {
		return false
	}

	if expires := values.Get("expires"); expires != "" {
		ts, err := strconv.ParseInt(expires, 10, 64)
		if err != nil || time.Now().Unix() > ts {
			return false
		}
	}

	values.Del("signature")
	parsed.RawQuery = values.Encode()
	expected := sign(g.signingKey, parsed.String())
	return hmac.Equal([]byte(expected), []byte(signature))
}

// TemporaryUploadURL builds a signed URL for temporary file uploads.
func (g *Generator) TemporaryUploadURL(path string, expiresIn time.Duration) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("upload path required")
	}
	return g.Signed("/api/uploads/signed", expiresIn, map[string]string{"path": path})
}

// HasValidSignatureFromRequest validates the current request URL.
func (g *Generator) HasValidSignatureFromRequest(path string, rawQuery string) bool {
	full := g.To(path)
	if rawQuery != "" {
		full += "?" + rawQuery
	}
	return g.HasValidSignature(full)
}

// ValidateSignature middleware rejects requests with invalid signed URLs.
func ValidateSignature(urls *Generator) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			rawQuery := ""
			if req.Raw() != nil && req.Raw().URL != nil {
				rawQuery = req.Raw().URL.RawQuery
			}
			if !urls.HasValidSignatureFromRequest(req.Path(), rawQuery) {
				if req.WantsJSON() {
					return http.JSON(map[string]any{"message": "Invalid signature"}).Status(403)
				}
				return http.Abort(403, "Invalid signature")
			}
			return next(req)
		}
	}
}

func (g *Generator) absoluteWithQuery(path string, values url.Values) string {
	base := g.To(path)
	encoded := values.Encode()
	if encoded == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + encoded
}

func sign(key, payload string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
