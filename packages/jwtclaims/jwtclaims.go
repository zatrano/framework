// Package jwtclaims, JWT token iddialarını services ve middleware katmanları arasında paylaşır
// (import döngüsünü önlemek için middlewares paketinin dışında tutulur).
package jwtclaims

import "github.com/golang-jwt/jwt/v5"

// JWTClaims — JWT token'ındaki iddialar yapısı.
type JWTClaims struct {
	UserID     uint   `json:"user_id"`
	Email      string `json:"email"`
	UserTypeID uint   `json:"user_type_id"`
	jwt.RegisteredClaims
}
