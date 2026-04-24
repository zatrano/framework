package services_test

import (
	"os"
	"testing"

	"github.com/zatrano/framework/models"
	"github.com/zatrano/framework/services"
	"github.com/zatrano/framework/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func setupJWT(t *testing.T) (services.IJWTService, *mocks.MockUserRepository) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key-min-32-chars-long!")
	os.Setenv("JWT_ACCESS_TTL_MINUTES", "15")
	os.Setenv("JWT_REFRESH_TTL_DAYS", "7")
	repo := new(mocks.MockUserRepository)
	return services.NewJWTService(repo), repo
}

func TestGenerateToken_Valid(t *testing.T) {
	svc, _ := setupJWT(t)
	user := &models.User{
		BaseModel:  models.BaseModel{ID: 1},
		Email:      "jwt@example.com",
		UserTypeID: 2,
	}
	token, err := svc.GenerateToken(user)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken_Valid(t *testing.T) {
	svc, _ := setupJWT(t)
	user := &models.User{
		BaseModel:  models.BaseModel{ID: 42},
		Email:      "valid@example.com",
		UserTypeID: 1,
	}
	token, err := svc.GenerateToken(user)
	assert.NoError(t, err)

	claims, err := svc.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), claims.UserID)
	assert.Equal(t, "valid@example.com", claims.Email)
	assert.Equal(t, uint(1), claims.UserTypeID)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	svc, _ := setupJWT(t)
	_, err := svc.ValidateToken("bu.gecersiz.token")
	assert.Error(t, err)
}

func TestRefreshAccessToken_Success(t *testing.T) {
	svc, repo := setupJWT(t)
	user := &models.User{
		BaseModel:  models.BaseModel{ID: 7},
		Email:      "refresh@example.com",
		UserTypeID: 2,
	}
	refreshToken, err := svc.GenerateRefreshToken(user)
	assert.NoError(t, err)

	repo.On("GetUserByID", mock.Anything, uint(7)).Return(user, nil)

	newToken, err := svc.RefreshAccessToken(refreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
}

func TestRefreshAccessToken_WithAccessToken_Fails(t *testing.T) {
	svc, _ := setupJWT(t)
	user := &models.User{BaseModel: models.BaseModel{ID: 8}, Email: "x@x.com"}
	// Erişim token'ı (subject="access") ile yenileme denemesi
	accessToken, _ := svc.GenerateToken(user)
	_, err := svc.RefreshAccessToken(accessToken)
	assert.Error(t, err)
}
