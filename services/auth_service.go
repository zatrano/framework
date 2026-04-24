package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/packages/mailqueue"
	"github.com/zatrano/framework/repositories"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type ServiceError string

func (e ServiceError) Error() string { return string(e) }

const (
	ErrInvalidCredentials       ServiceError = "geçersiz kimlik bilgileri"
	ErrUserNotFound             ServiceError = "kullanıcı bulunamadı"
	ErrUserInactive             ServiceError = "kullanıcı aktif değil"
	ErrCurrentPasswordIncorrect ServiceError = "mevcut şifre hatalı"
	ErrPasswordTooShort         ServiceError = "yeni şifre en az 6 karakter olmalıdır"
	ErrPasswordSameAsOld        ServiceError = "yeni şifre mevcut şifre ile aynı olamaz"
	ErrAuthGeneric              ServiceError = "kimlik doğrulaması sırasında bir hata oluştu"
	ErrProfileGeneric           ServiceError = "profil bilgileri alınırken hata"
	ErrUpdatePasswordGeneric    ServiceError = "şifre güncellenirken bir hata oluştu"
	ErrHashingFailed            ServiceError = "yeni şifre oluşturulurken hata"
	ErrDatabaseUpdateFailed     ServiceError = "veritabanı güncellemesi başarısız oldu"
	ErrEmailSendFailed          ServiceError = "e-posta gönderilemedi"
	ErrEmailAlreadyExists       ServiceError = "bu e-posta adresi zaten kayıtlı"
	ErrTokenExpired             ServiceError = "token süresi dolmuş, lütfen tekrar talep edin"
)

const (
	resetTokenTTL        = 1 * time.Hour
	verificationTokenTTL = 24 * time.Hour
)

type IAuthService interface {
	Authenticate(email, password string) (*models.User, error)
	RegisterUser(ctx context.Context, name, email, password string) error
	VerifyEmail(token string) error
	ResendVerificationLink(email string) error

	SendPasswordResetLink(email string) error
	ResetPassword(token, newPassword string) error
	UpdatePassword(ctx context.Context, userID uint, currentPass, newPassword string) error

	GetUserProfile(ctx context.Context, id uint) (*models.User, error)
	UpdateUserInfo(ctx context.Context, userID uint, name, email string) error

	FindOrCreateOAuthUser(providerID, email, name, provider string) (*models.User, error)
}

type AuthService struct {
	repo        repositories.IAuthRepository
	mailService IMailService
}

func NewAuthService(repo repositories.IAuthRepository, mailService IMailService) IAuthService {
	return &AuthService{repo: repo, mailService: mailService}
}

func (s *AuthService) logAuthSuccess(email string, userID uint) {
	logconfig.Log.Info("Kimlik doğrulama başarılı",
		zap.String("email", email),
		zap.Uint("user_id", userID))
}

func (s *AuthService) logDBError(action string, err error, fields ...zap.Field) {
	fields = append(fields, zap.Error(err))
	logconfig.Log.Error(action+" hatası (DB)", fields...)
}

func (s *AuthService) logWarn(action string, fields ...zap.Field) {
	logconfig.Log.Warn(action+" başarısız", fields...)
}

func (s *AuthService) generateToken() string {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		logconfig.Log.Error("Token oluşturulamadı", zap.Error(err))
		return ""
	}
	return hex.EncodeToString(tokenBytes)
}

func (s *AuthService) getUserByEmail(email string) (*models.User, error) {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logWarn("Kullanıcı bulunamadı", zap.String("email", email))
			return nil, ErrUserNotFound
		}
		s.logDBError("Kullanıcı sorgulama", err, zap.String("email", email))
		return nil, ErrAuthGeneric
	}
	return user, nil
}

func (s *AuthService) getUserByID(id uint) (*models.User, error) {
	user, err := s.repo.FindUserByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logWarn("Kullanıcı bulunamadı", zap.Uint("user_id", id))
			return nil, ErrUserNotFound
		}
		s.logDBError("Kullanıcı sorgulama", err, zap.Uint("user_id", id))
		return nil, ErrProfileGeneric
	}
	return user, nil
}

func (s *AuthService) comparePasswords(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}

func (s *AuthService) hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (s *AuthService) Authenticate(email, password string) (*models.User, error) {
	user, err := s.getUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if !user.IsActive {
		s.logWarn("Kullanıcı aktif değil", zap.String("email", email))
		return nil, ErrUserInactive
	}
	if err := s.comparePasswords(user.Password, password); err != nil {
		s.logWarn("Geçersiz parola", zap.String("email", email))
		return nil, ErrInvalidCredentials
	}
	s.logAuthSuccess(email, user.ID)
	return user, nil
}

func (s *AuthService) RegisterUser(ctx context.Context, name, email, password string) error {
	existing, err := s.repo.FindUserByEmail(email)
	if err == nil && existing != nil {
		return ErrEmailAlreadyExists
	}

	verificationToken := s.generateToken()
	if verificationToken == "" {
		return errors.New("token oluşturulamadı")
	}

	hashedPassword, err := s.hashPassword(password)
	if err != nil {
		s.logDBError("Şifre hashleme", err, zap.String("email", email))
		return ErrHashingFailed
	}

	expiresAt := time.Now().Add(verificationTokenTTL)
	user := &models.User{
		Name:                  name,
		Email:                 email,
		Password:              hashedPassword,
		UserTypeID:            2,
		EmailVerified:         false,
		VerificationToken:     verificationToken,
		VerificationExpiresAt: &expiresAt,
		BaseModel:             models.BaseModel{IsActive: true},
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		s.logDBError("Kullanıcı oluşturma", err, zap.String("email", email))
		return ErrDatabaseUpdateFailed
	}

	// ← ASENKRON: mail kuyruğu üzerinden gönder, işleyiciyi bloklamaz
	s.enqueueVerificationEmail(user.Email, verificationToken)

	logconfig.Log.Info("Kullanıcı kaydı başarılı",
		zap.String("email", email), zap.Uint("user_id", user.ID))
	return nil
}

// enqueueVerificationEmail — doğrulama mailini asenkron kuyruğa ekler.
func (s *AuthService) enqueueVerificationEmail(email, token string) {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		logconfig.Log.Warn("APP_BASE_URL tanımlı değil, doğrulama maili gönderilemedi")
		return
	}
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "github.com/zatrano/framework"
	}
	link := fmt.Sprintf("%s/auth/verify-email?token=%s", baseURL, token)

	mailqueue.Send(email, "E-posta Doğrulama", "verification", map[string]interface{}{
		"Link":        link,
		"SiteName":    siteName,
		"ExpiryHours": int(verificationTokenTTL.Hours()),
	})
}

// enqueuePasswordResetEmail — şifre sıfırlama mailini asenkron kuyruğa ekler.
func (s *AuthService) enqueuePasswordResetEmail(email, token string) {
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		return
	}
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "github.com/zatrano/framework"
	}
	link := fmt.Sprintf("%s/auth/reset-password?token=%s", baseURL, token)

	mailqueue.Send(email, "Şifre Sıfırlama", "password_reset", map[string]interface{}{
		"Link":        link,
		"SiteName":    siteName,
		"ExpiryHours": int(resetTokenTTL.Hours()),
	})
}

func (s *AuthService) VerifyEmail(token string) error {
	user, err := s.repo.FindUserByVerificationToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return ErrAuthGeneric
	}
	if user.VerificationExpiresAt != nil && time.Now().After(*user.VerificationExpiresAt) {
		s.logWarn("Verification token süresi dolmuş", zap.Uint("user_id", user.ID))
		return ErrTokenExpired
	}
	user.EmailVerified = true
	user.VerificationToken = ""
	user.VerificationExpiresAt = nil
	if err := s.repo.UpdateUser(context.Background(), user); err != nil {
		s.logDBError("Email doğrulama", err, zap.Uint("user_id", user.ID))
		return ErrDatabaseUpdateFailed
	}
	logconfig.Log.Info("Email doğrulandı", zap.Uint("user_id", user.ID))
	return nil
}

func (s *AuthService) ResendVerificationLink(email string) error {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return ErrAuthGeneric
	}
	if user.EmailVerified {
		return nil
	}
	verificationToken := s.generateToken()
	if verificationToken == "" {
		return errors.New("token oluşturulamadı")
	}
	expiresAt := time.Now().Add(verificationTokenTTL)
	user.VerificationToken = verificationToken
	user.VerificationExpiresAt = &expiresAt
	if err := s.repo.UpdateUser(context.Background(), user); err != nil {
		s.logDBError("Verification token güncelleme", err, zap.String("email", email))
		return ErrDatabaseUpdateFailed
	}
	s.enqueueVerificationEmail(user.Email, verificationToken)
	logconfig.Log.Info("Doğrulama linki yeniden gönderildi (async)", zap.String("email", email))
	return nil
}

func (s *AuthService) SendPasswordResetLink(email string) error {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return ErrAuthGeneric
	}
	resetToken := s.generateToken()
	if resetToken == "" {
		return errors.New("token oluşturulamadı")
	}
	resetExpiresAt := time.Now().Add(resetTokenTTL)
	user.ResetToken = resetToken
	user.ResetTokenExpiresAt = &resetExpiresAt
	if err := s.repo.UpdateUser(context.Background(), user); err != nil {
		s.logDBError("Reset token güncelleme", err, zap.String("email", email))
		return ErrDatabaseUpdateFailed
	}
	s.enqueuePasswordResetEmail(user.Email, resetToken)
	logconfig.Log.Info("Şifre sıfırlama linki gönderildi (async)", zap.String("email", email))
	return nil
}

func (s *AuthService) ResetPassword(token, newPassword string) error {
	user, err := s.repo.FindUserByResetToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return ErrAuthGeneric
	}
	if user.ResetTokenExpiresAt != nil && time.Now().After(*user.ResetTokenExpiresAt) {
		s.logWarn("Reset token süresi dolmuş", zap.Uint("user_id", user.ID))
		return ErrTokenExpired
	}
	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		s.logDBError("Şifre hashleme", err, zap.Uint("user_id", user.ID))
		return ErrHashingFailed
	}
	user.Password = hashedPassword
	user.ResetToken = ""
	user.ResetTokenExpiresAt = nil
	if err := s.repo.UpdateUser(context.Background(), user); err != nil {
		s.logDBError("Şifre sıfırlama", err, zap.Uint("user_id", user.ID))
		return ErrDatabaseUpdateFailed
	}
	logconfig.Log.Info("Şifre sıfırlandı", zap.Uint("user_id", user.ID))
	return nil
}

func (s *AuthService) UpdatePassword(ctx context.Context, userID uint, currentPass, newPassword string) error {
	user, err := s.getUserByID(userID)
	if err != nil {
		return err
	}
	if user.Password == "" {
		if user.Provider == "" {
			return errors.New("provider bilgisi olmayan kullanıcı için şifre boş olamaz")
		}
		if len(newPassword) < 6 {
			return ErrPasswordTooShort
		}
		hashedPassword, err := s.hashPassword(newPassword)
		if err != nil {
			return ErrHashingFailed
		}
		user.Password = hashedPassword
		if err := s.repo.UpdateUser(ctx, user); err != nil {
			return ErrDatabaseUpdateFailed
		}
		return nil
	}
	if err := s.comparePasswords(user.Password, currentPass); err != nil {
		return ErrCurrentPasswordIncorrect
	}
	if len(newPassword) < 6 {
		return ErrPasswordTooShort
	}
	if currentPass == newPassword {
		return ErrPasswordSameAsOld
	}
	hashedPassword, err := s.hashPassword(newPassword)
	if err != nil {
		return ErrHashingFailed
	}
	user.Password = hashedPassword
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		return ErrDatabaseUpdateFailed
	}
	logconfig.Log.Info("Parola güncellendi", zap.Uint("user_id", userID))
	return nil
}

func (s *AuthService) GetUserProfile(ctx context.Context, id uint) (*models.User, error) {
	return s.getUserByID(id)
}

func (s *AuthService) UpdateUserInfo(ctx context.Context, userID uint, name, email string) error {
	user, err := s.getUserByID(userID)
	if err != nil {
		return err
	}
	if user.Email != email {
		existing, err := s.repo.FindUserByEmail(email)
		if err == nil && existing != nil && existing.ID != userID {
			return ErrEmailAlreadyExists
		}
	}
	user.Name = name
	user.Email = email
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logDBError("Kullanıcı bilgileri güncelleme", err, zap.Uint("user_id", userID))
		return ErrDatabaseUpdateFailed
	}
	logconfig.Log.Info("Kullanıcı bilgileri güncellendi", zap.Uint("user_id", userID))
	return nil
}

func (s *AuthService) FindOrCreateOAuthUser(providerID, email, name, provider string) (*models.User, error) {
	existing, err := s.repo.FindByProviderAndID(provider, providerID)
	if err == nil {
		return existing, nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if byEmail, e2 := s.repo.FindUserByEmail(email); e2 == nil {
			byEmail.Provider = provider
			byEmail.ProviderID = providerID
			byEmail.EmailVerified = true
			if err := s.repo.UpdateUser(context.Background(), byEmail); err != nil {
				return nil, err
			}
			return byEmail, nil
		}
	}
	u := &models.User{
		Name:          name,
		Email:         email,
		UserTypeID:    2,
		EmailVerified: true,
		BaseModel:     models.BaseModel{IsActive: true},
		Provider:      provider,
		ProviderID:    providerID,
	}
	if err := s.repo.CreateUser(context.Background(), u); err != nil {
		return nil, err
	}
	logconfig.Log.Info("OAuth kullanıcısı oluşturuldu",
		zap.String("provider", provider), zap.String("email", email))
	return u, nil
}

var _ IAuthService = (*AuthService)(nil)
