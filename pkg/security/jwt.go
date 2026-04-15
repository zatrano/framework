package security

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"github.com/zatrano/framework/pkg/config"
)

const jwtClaimsLocal = "zatrano.jwt.claims"

// ClaimsKey is the fiber.Locals key for *jwt.MapClaims after JWTMiddleware.
func ClaimsKey() string { return jwtClaimsLocal }

// SignAccessToken builds a signed HS256 access token (for tests and demo flows).
func SignAccessToken(cfg *config.Config, subject string, extra map[string]any) (string, error) {
	secret := strings.TrimSpace(cfg.Security.JWTSecret)
	if secret == "" {
		return "", fmt.Errorf("jwt_secret is empty")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(cfg.Security.JWTExpiry).Unix(),
		"iss": cfg.Security.JWTIssuer,
	}
	for k, v := range extra {
		claims[k] = v
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString([]byte(secret))
}

// JWTMiddleware validates Authorization: Bearer <JWT> (HS256, issuer, expiry).
func JWTMiddleware(cfg *config.Config) fiber.Handler {
	secret := []byte(strings.TrimSpace(cfg.Security.JWTSecret))
	issuer := cfg.Security.JWTIssuer
	return func(c fiber.Ctx) error {
		raw := strings.TrimSpace(strings.TrimPrefix(c.Get("Authorization"), "Bearer "))
		if raw == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "missing bearer token")
		}
		tok, err := jwt.Parse(raw, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return secret, nil
		}, jwt.WithIssuer(issuer), jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !tok.Valid {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid token")
		}
		mc, ok := tok.Claims.(jwt.MapClaims)
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid claims")
		}
		c.Locals(jwtClaimsLocal, mc)
		return c.Next()
	}
}
