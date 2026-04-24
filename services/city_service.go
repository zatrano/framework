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

type ICityService interface {
	GetAllCities(ctx context.Context, params requests.CityListParams) (*requests.PaginatedResult, error)
	GetCityByID(ctx context.Context, id uint) (*models.City, error)
	CreateCity(ctx context.Context, req requests.CreateCityRequest) error
	UpdateCity(ctx context.Context, id uint, req requests.UpdateCityRequest) error
	DeleteCity(ctx context.Context, id uint) error
	GetCityCount(ctx context.Context) (int64, error)
}

type CityService struct {
	repo repositories.ICityRepository
}

func NewCityService(repo repositories.ICityRepository) ICityService {
	return &CityService{repo: repo}
}

func (s *CityService) GetAllCities(ctx context.Context, params requests.CityListParams) (*requests.PaginatedResult, error) {
	cities, totalCount, err := s.repo.GetAllCities(ctx, params)
	if err != nil {
		logconfig.Log.Error("Şehirler alınamadı", zap.Error(err))
		return nil, errors.New("şehirler getirilirken bir hata oluştu")
	}
	return requests.CreatePaginatedResult(cities, totalCount, params.Page, params.PerPage), nil
}

func (s *CityService) GetCityByID(ctx context.Context, id uint) (*models.City, error) {
	city, err := s.repo.GetCityByID(ctx, id)
	if err != nil {
		logconfig.Log.Warn("Şehir bulunamadı", zap.Uint("city_id", id), zap.Error(err))
		return nil, errors.New("şehir bulunamadı")
	}
	return city, nil
}

func (s *CityService) CreateCity(ctx context.Context, req requests.CreateCityRequest) error {
	converted := req.BaseCityRequest.Convert()
	if converted.CountryID == nil {
		return errors.New("ülke seçilmelidir")
	}
	city := &models.City{
		BaseModel: models.BaseModel{IsActive: converted.IsActive != nil && *converted.IsActive},
		CountryID: *converted.CountryID,
		Name:      converted.Name,
	}
	return s.repo.CreateCity(ctx, city)
}

func (s *CityService) UpdateCity(ctx context.Context, id uint, req requests.UpdateCityRequest) error {
	if _, err := s.repo.GetCityByID(ctx, id); err != nil {
		return errors.New("şehir bulunamadı")
	}
	converted := req.BaseCityRequest.Convert()
	if converted.CountryID == nil {
		return errors.New("ülke seçilmelidir")
	}
	updateData := map[string]interface{}{
		"country_id": *converted.CountryID,
		"name":       converted.Name,
		"is_active":  converted.IsActive != nil && *converted.IsActive,
	}
	return s.repo.UpdateCity(ctx, id, updateData)
}

func (s *CityService) DeleteCity(ctx context.Context, id uint) error {
	return s.repo.DeleteCity(ctx, id)
}

func (s *CityService) GetCityCount(ctx context.Context) (int64, error) {
	return s.repo.GetCityCount(ctx)
}
