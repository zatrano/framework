package services

import (
	"context"
	"errors"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/repositories"
	"github.com/zatrano/framework/requests"
	"golang.org/x/crypto/bcrypt"

	"go.uber.org/zap"
)

type IUserService interface {
	GetAllUsers(ctx context.Context, params requests.UserListParams) (*requests.PaginatedResult, error)
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	CreateUser(ctx context.Context, req requests.CreateUserRequest) error
	UpdateUser(ctx context.Context, id uint, req requests.UpdateUserRequest) error
	DeleteUser(ctx context.Context, id uint) error
	GetUserCount(ctx context.Context) (int64, error)
}

type UserService struct {
	repo repositories.IUserRepository
}

func NewUserService(repo repositories.IUserRepository) IUserService {
	return &UserService{repo: repo}
}

func (s *UserService) hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (s *UserService) GetAllUsers(ctx context.Context, params requests.UserListParams) (*requests.PaginatedResult, error) {
	users, totalCount, err := s.repo.GetAllUsers(ctx, params)
	if err != nil {
		logconfig.Log.Error("Kullanıcılar alınamadı", zap.Error(err))
		return nil, errors.New("kullanıcılar getirilirken bir hata oluştu")
	}
	return requests.CreatePaginatedResult(users, totalCount, params.Page, params.PerPage), nil
}

func (s *UserService) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		logconfig.Log.Warn("Kullanıcı bulunamadı", zap.Uint("user_id", id), zap.Error(err))
		return nil, errors.New("kullanıcı bulunamadı")
	}
	return user, nil
}

func (s *UserService) CreateUser(ctx context.Context, req requests.CreateUserRequest) error {
	converted := req.BaseUserRequest.Convert()

	if converted.UserTypeID == nil {
		return errors.New("kullanıcı tipi seçilmelidir")
	}

	user := &models.User{
		BaseModel: models.BaseModel{
			IsActive: converted.IsActive != nil && *converted.IsActive,
		},
		Name:              converted.Name,
		Email:             converted.Email,
		Password:          req.Password,
		UserTypeID:        *converted.UserTypeID,
		ResetToken:        converted.ResetToken,
		EmailVerified:     converted.EmailVerified != nil && *converted.EmailVerified,
		VerificationToken: converted.VerificationToken,
		Provider:          converted.Provider,
		ProviderID:        converted.ProviderID,
	}

	if user.Password == "" {
		return errors.New("şifre alanı boş olamaz")
	}

	hashedPass, err := s.hashPassword(user.Password)
	if err != nil {
		logconfig.Log.Error("Şifre oluşturulamadı", zap.Error(err))
		return errors.New("şifre oluşturulurken hata oluştu")
	}
	user.Password = hashedPass

	return s.repo.CreateUser(ctx, user)
}

func (s *UserService) UpdateUser(ctx context.Context, id uint, req requests.UpdateUserRequest) error {
	_, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return errors.New("kullanıcı bulunamadı")
	}

	converted := req.BaseUserRequest.Convert()

	if converted.UserTypeID == nil {
		return errors.New("kullanıcı tipi seçilmelidir")
	}

	updateData := map[string]interface{}{
		"name":               converted.Name,
		"email":              converted.Email,
		"is_active":          converted.IsActive != nil && *converted.IsActive,
		"user_type_id":       *converted.UserTypeID,
		"reset_token":        converted.ResetToken,
		"verification_token": converted.VerificationToken,
		"provider":           converted.Provider,
		"provider_id":        converted.ProviderID,
	}

	if converted.EmailVerified != nil {
		updateData["email_verified"] = *converted.EmailVerified
	}

	if req.Password != "" {
		hashed, err := s.hashPassword(req.Password)
		if err != nil {
			return errors.New("şifre oluşturulurken hata oluştu")
		}
		updateData["password"] = hashed
	}

	return s.repo.UpdateUser(ctx, id, updateData)
}

func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *UserService) GetUserCount(ctx context.Context) (int64, error) {
	return s.repo.GetUserCount(ctx)
}
