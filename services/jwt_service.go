package services

import (
	"context"
	"errors"
	"time"

	"github.com/zatrano/framework/configs/envconfig"
	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/jwtclaims"
	"github.com/zatrano/framework/packages/jwtrevoke"
	"github.com/zatrano/framework/repositories"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type IJWTService interface {
	GenerateToken(user *models.User) (string, error)
	GenerateRefreshToken(user *models.User) (string, error)
	ValidateToken(tokenStr string) (*jwtclaims.JWTClaims, error)
	RefreshAccessToken(refreshToken string) (string, error)
	RevokeToken(tokenStr string) error
	IsTokenRevoked(tokenStr string) (bool, error)
}

type JWTService struct {
	userRepo   repositories.IUserRepository
	secret     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTService(userRepo repositories.IUserRepository) IJWTService {
	return &JWTService{
		userRepo:   userRepo,
		secret:     envconfig.String("JWT_SECRET", "change-me-in-production"),
		accessTTL:  time.Duration(envconfig.Int("JWT_ACCESS_TTL_MINUTES", 15)) * time.Minute,
		refreshTTL: time.Duration(envconfig.Int("JWT_REFRESH_TTL_DAYS", 7)) * 24 * time.Hour,
	}
}

func (s *JWTService) GenerateToken(user *models.User) (string, error) {
	claims := jwtclaims.JWTClaims{
		UserID:     user.ID,
		Email:      user.Email,
		UserTypeID: user.UserTypeID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    envconfig.String("APP_NAME", "github.com/zatrano/framework"),
			Subject:   "access",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) GenerateRefreshToken(user *models.User) (string, error) {
	claims := jwtclaims.JWTClaims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    envconfig.String("APP_NAME", "github.com/zatrano/framework"),
			Subject:   "refresh",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) ValidateToken(tokenStr string) (*jwtclaims.JWTClaims, error) {
	claims := &jwtclaims.JWTClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("beklenmeyen imzalama yöntemi")
		}
		return []byte(s.secret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("geçersiz token")
	}
	return claims, nil
}

func (s *JWTService) RefreshAccessToken(refreshToken string) (string, error) {
	revoked, err := s.IsTokenRevoked(refreshToken)
	if err != nil {
		return "", errors.New("token durumu doğrulanamadı")
	}
	if revoked {
		return "", errors.New("refresh token geçersiz")
	}

	claims, err := s.ValidateToken(refreshToken)
	if err != nil {
		return "", errors.New("geçersiz refresh token")
	}
	if claims.Subject != "refresh" {
		return "", errors.New("bu token bir refresh token değil")
	}
	user, err := s.userRepo.GetUserByID(context.Background(), claims.UserID)
	if err != nil {
		return "", errors.New("kullanıcı bulunamadı")
	}
	return s.GenerateToken(user)
}

func (s *JWTService) RevokeToken(tokenStr string) error {
	claims, err := s.ValidateToken(tokenStr)
	if err != nil {
		return err
	}
	expiresAt := claims.ExpiresAt.Time
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	if err := jwtrevoke.RevokeToken(context.Background(), tokenStr, ttl); err != nil {
		logconfig.Log.Error("JWT revoke kaydı başarısız", zap.Error(err))
		return err
	}
	return nil
}

func (s *JWTService) IsTokenRevoked(tokenStr string) (bool, error) {
	return jwtrevoke.IsRevoked(context.Background(), tokenStr)
}
