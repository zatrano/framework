package middlewares

import (
	"strings"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/packages/apierrors"
	"github.com/zatrano/framework/packages/jwtclaims"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const JWTClaimsKey = "jwt_claims"

// JWTAuth — Bearer belirteci doğrulama ara yazılımı.
// API uç noktaları için session yerine JWT kullanılır.
func JWTAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return apierrors.Send(c, apierrors.Unauthorized("Authorization header eksik"))
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return apierrors.Send(c, apierrors.Unauthorized("Geçersiz Authorization formatı. 'Bearer <token>' bekleniyor"))
		}

		tokenStr := parts[1]
		secret := envconfig.String("JWT_SECRET", "")
		if secret == "" {
			logconfig.Log.Error("JWT_SECRET env değişkeni tanımlı değil")
			return apierrors.Send(c, apierrors.Internal("Sunucu yapılandırma hatası"))
		}

		claims := &jwtclaims.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "beklenmeyen imzalama yöntemi")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			logconfig.Log.Warn("Geçersiz JWT token",
				zap.Error(err),
				zap.String("ip", c.IP()))
			return apierrors.Send(c, apierrors.Unauthorized("Geçersiz veya süresi dolmuş token"))
		}

		c.Locals(JWTClaimsKey, claims)
		c.Locals("userID", claims.UserID)
		return c.Next()
	}
}

// JWTClaimsFromFiber — fiber context'ten JWT iddialarını alır.
func JWTClaimsFromFiber(c fiber.Ctx) *jwtclaims.JWTClaims {
	if claims, ok := c.Locals(JWTClaimsKey).(*jwtclaims.JWTClaims); ok {
		return claims
	}
	return nil
}

// JWTTypeMiddleware — JWT üzerinden kullanıcı tipi kontrolü.
func JWTTypeMiddleware(allowedTypes ...uint) fiber.Handler {
	return func(c fiber.Ctx) error {
		claims := JWTClaimsFromFiber(c)
		if claims == nil {
			return apierrors.Send(c, apierrors.Unauthorized("Token bilgisi alınamadı"))
		}
		for _, t := range allowedTypes {
			if claims.UserTypeID == t {
				return c.Next()
			}
		}
		return apierrors.Send(c, apierrors.Forbidden("Bu işlem için yetkiniz bulunmuyor"))
	}
}
