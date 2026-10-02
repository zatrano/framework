package middleware

import (
	stdhttp "net/http"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/encryption"
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
)

const encryptedPrefix = "ZATRANO:"

// EncryptCookies encrypts selected response cookies and decrypts matching request cookies.
// Encrypt failure drops that cookie (plaintext is never sent). Decrypt failure
// leaves the ciphertext on the request. Pass nil encrypter for a no-op.
func EncryptCookies(encrypter *encryption.Encrypter, names ...string) routing.MiddlewareFunc {
	wanted := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" {
			wanted[name] = true
		}
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if encrypter != nil && req != nil {
				decryptRequestCookies(req, encrypter, wanted)
			}
			resp := next(req)
			if resp == nil || encrypter == nil {
				return resp
			}
			kept := make([]*stdhttp.Cookie, 0, len(resp.Cookies()))
			for _, cookie := range resp.Cookies() {
				if cookie == nil {
					continue
				}
				if len(wanted) > 0 && !wanted[cookie.Name] {
					kept = append(kept, cookie)
					continue
				}
				if strings.HasPrefix(cookie.Value, encryptedPrefix) {
					kept = append(kept, cookie)
					continue
				}
				cipher, err := encrypter.Encrypt(cookie.Value)
				if err != nil {
					continue
				}
				cookie.Value = encryptedPrefix + cipher
				kept = append(kept, cookie)
			}
			return resp.ReplaceCookies(kept)
		}
	}
}

func decryptRequestCookies(req *http.Request, encrypter *encryption.Encrypter, wanted map[string]bool) {
	cookies := req.CookieMap()
	if len(cookies) == 0 {
		return
	}
	for name, value := range cookies {
		if len(wanted) > 0 && !wanted[name] {
			continue
		}
		if !strings.HasPrefix(value, encryptedPrefix) {
			continue
		}
		plain, err := encrypter.Decrypt(strings.TrimPrefix(value, encryptedPrefix))
		if err != nil {
			continue
		}
		req.SetCookie(name, plain)
	}
}
