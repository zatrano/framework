package mocks

import (
	"context"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/requests"

	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) GetAllUsers(ctx context.Context, params requests.UserListParams) ([]models.User, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) CreateUser(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

func (m *MockUserRepository) UpdateUser(ctx context.Context, id uint, data map[string]interface{}) error {
	return m.Called(ctx, id, data).Error(0)
}

func (m *MockUserRepository) DeleteUser(ctx context.Context, id uint) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockUserRepository) GetUserCount(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}
