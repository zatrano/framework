package services

import (
	"context"
	"errors"

	"github.com/zatrano/framework/configs/logconfig"
	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/repositories"
	"github.com/zatrano/framework/requests"

	"go.uber.org/zap"
)

type IUserTypeService interface {
	GetAllUserTypes(ctx context.Context, params requests.UserTypeListParams) (*requests.PaginatedResult, error)
	GetUserTypeByID(ctx context.Context, id uint) (*models.UserType, error)
	CreateUserType(ctx context.Context, req requests.UserTypeRequest) error
	UpdateUserType(ctx context.Context, id uint, req requests.UserTypeRequest) error
	DeleteUserType(ctx context.Context, id uint) error
}

type UserTypeService struct {
	repo repositories.IUserTypeRepository
}

func NewUserTypeService(repo repositories.IUserTypeRepository) IUserTypeService {
	return &UserTypeService{repo: repo}
}

func (s *UserTypeService) GetAllUserTypes(ctx context.Context, params requests.UserTypeListParams) (*requests.PaginatedResult, error) {
	userTypes, totalCount, err := s.repo.GetAllUserTypes(ctx, params)
	if err != nil {
		return nil, err
	}

	return requests.CreatePaginatedResult(userTypes, totalCount, params.Page, params.PerPage), nil
}

func (s *UserTypeService) GetUserTypeByID(ctx context.Context, id uint) (*models.UserType, error) {
	userType, err := s.repo.GetUserTypeByID(ctx, id)
	if err != nil {
		logconfig.Log.Warn("Kullanıcı Tipi bulunamadı", zap.Uint("user_type_id", id), zap.Error(err))
		return nil, errors.New("kullanıcı tipi bulunamadı")
	}
	return userType, nil
}

func (s *UserTypeService) CreateUserType(ctx context.Context, req requests.UserTypeRequest) error {
	converted := req.BaseUserTypeRequest.Convert()

	userType := &models.UserType{
		BaseModel: models.BaseModel{IsActive: false},
		Name:      converted.Name,
	}

	if converted.IsActive != nil {
		userType.BaseModel.IsActive = *converted.IsActive
	}

	return s.repo.CreateUserType(ctx, userType)
}

func (s *UserTypeService) UpdateUserType(ctx context.Context, id uint, req requests.UserTypeRequest) error {
	_, err := s.repo.GetUserTypeByID(ctx, id)
	if err != nil {
		return errors.New("kullanıcı tipi bulunamadı")
	}

	converted := req.BaseUserTypeRequest.Convert()

	updateData := map[string]interface{}{
		"name":      converted.Name,
		"is_active": converted.IsActive != nil && *converted.IsActive,
	}

	return s.repo.UpdateUserType(ctx, id, updateData)
}

func (s *UserTypeService) DeleteUserType(ctx context.Context, id uint) error {
	return s.repo.DeleteUserType(ctx, id)
}
