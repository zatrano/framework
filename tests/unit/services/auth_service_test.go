package services_test

import (
	"context"
	"testing"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/services"
	"github.com/zatrano/framework/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
	"golang.org/x/crypto/bcrypt"
)

func hashedPass(t *testing.T, plain string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	assert.NoError(t, err)
	return string(b)
}

func TestAuthenticate_Success(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	user := &models.User{
		BaseModel: models.BaseModel{ID: 1, IsActive: true},
		Email:     "test@example.com",
		Password:  hashedPass(t, "secret123"),
	}
	repo.On("FindUserByEmail", "test@example.com").Return(user, nil)

	svc := services.NewAuthService(repo, mail)
	result, err := svc.Authenticate("test@example.com", "secret123")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, uint(1), result.ID)
	repo.AssertExpectations(t)
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	user := &models.User{
		BaseModel: models.BaseModel{ID: 1, IsActive: true},
		Email:     "test@example.com",
		Password:  hashedPass(t, "secret123"),
	}
	repo.On("FindUserByEmail", "test@example.com").Return(user, nil)

	svc := services.NewAuthService(repo, mail)
	result, err := svc.Authenticate("test@example.com", "yanlis")

	assert.Error(t, err)
	assert.Equal(t, services.ErrInvalidCredentials, err)
	assert.Nil(t, result)
}

func TestAuthenticate_UserNotFound(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	repo.On("FindUserByEmail", "yok@example.com").
		Return(nil, gorm.ErrRecordNotFound)

	svc := services.NewAuthService(repo, mail)
	result, err := svc.Authenticate("yok@example.com", "herhangi")

	assert.Error(t, err)
	assert.Equal(t, services.ErrUserNotFound, err)
	assert.Nil(t, result)
}

func TestAuthenticate_InactiveUser(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	user := &models.User{
		BaseModel: models.BaseModel{ID: 2, IsActive: false},
		Email:     "pasif@example.com",
		Password:  hashedPass(t, "secret123"),
	}
	repo.On("FindUserByEmail", "pasif@example.com").Return(user, nil)

	svc := services.NewAuthService(repo, mail)
	result, err := svc.Authenticate("pasif@example.com", "secret123")

	assert.Error(t, err)
	assert.Equal(t, services.ErrUserInactive, err)
	assert.Nil(t, result)
}

func TestRegisterUser_EmailAlreadyExists(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	existing := &models.User{BaseModel: models.BaseModel{ID: 1}, Email: "var@example.com"}
	repo.On("FindUserByEmail", "var@example.com").Return(existing, nil)

	svc := services.NewAuthService(repo, mail)
	err := svc.RegisterUser(context.Background(), "Test", "var@example.com", "pass1234")

	assert.Error(t, err)
	assert.Equal(t, services.ErrEmailAlreadyExists, err)
}

func TestRegisterUser_Success(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	repo.On("FindUserByEmail", "yeni@example.com").
		Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).
		Return(nil)

	svc := services.NewAuthService(repo, mail)
	err := svc.RegisterUser(context.Background(), "Yeni", "yeni@example.com", "sifre1234")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetUserProfile_Success(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	user := &models.User{
		BaseModel: models.BaseModel{ID: 5},
		Email:     "profil@example.com",
	}
	repo.On("FindUserByID", uint(5)).Return(user, nil)

	svc := services.NewAuthService(repo, mail)
	result, err := svc.GetUserProfile(context.Background(), 5)

	assert.NoError(t, err)
	assert.Equal(t, "profil@example.com", result.Email)
}

func TestUpdatePassword_CorrectFlow(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	user := &models.User{
		BaseModel: models.BaseModel{ID: 3, IsActive: true},
		Password:  hashedPass(t, "eskisifre"),
	}
	repo.On("FindUserByID", uint(3)).Return(user, nil)
	repo.On("UpdateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(nil)

	svc := services.NewAuthService(repo, mail)
	err := svc.UpdatePassword(context.Background(), 3, "eskisifre", "yenisifre123")

	assert.NoError(t, err)
}

func TestUpdatePassword_SamePassword(t *testing.T) {
	repo := new(mocks.MockAuthRepository)
	mail := new(mocks.MockMailService)

	user := &models.User{
		BaseModel: models.BaseModel{ID: 3, IsActive: true},
		Password:  hashedPass(t, "ayni123"),
	}
	repo.On("FindUserByID", uint(3)).Return(user, nil)

	svc := services.NewAuthService(repo, mail)
	err := svc.UpdatePassword(context.Background(), 3, "ayni123", "ayni123")

	assert.Error(t, err)
	assert.Equal(t, services.ErrPasswordSameAsOld, err)
}
