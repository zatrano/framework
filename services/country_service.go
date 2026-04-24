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

type ICountryService interface {
	GetAllCountries(ctx context.Context, params requests.CountryListParams) (*requests.PaginatedResult, error)
	GetCountryByID(ctx context.Context, id uint) (*models.Country, error)
	CreateCountry(ctx context.Context, req requests.CreateCountryRequest) error
	UpdateCountry(ctx context.Context, id uint, req requests.UpdateCountryRequest) error
	DeleteCountry(ctx context.Context, id uint) error
	GetCountryCount(ctx context.Context) (int64, error)
}

type CountryService struct {
	repo repositories.ICountryRepository
}

func NewCountryService(repo repositories.ICountryRepository) ICountryService {
	return &CountryService{repo: repo}
}

func (s *CountryService) GetAllCountries(ctx context.Context, params requests.CountryListParams) (*requests.PaginatedResult, error) {
	countries, totalCount, err := s.repo.GetAllCountries(ctx, params)
	if err != nil {
		logconfig.Log.Error("Ülkeler alınamadı", zap.Error(err))
		return nil, errors.New("ülkeler getirilirken bir hata oluştu")
	}
	return requests.CreatePaginatedResult(countries, totalCount, params.Page, params.PerPage), nil
}

func (s *CountryService) GetCountryByID(ctx context.Context, id uint) (*models.Country, error) {
	country, err := s.repo.GetCountryByID(ctx, id)
	if err != nil {
		logconfig.Log.Warn("Ülke bulunamadı", zap.Uint("country_id", id), zap.Error(err))
		return nil, errors.New("ülke bulunamadı")
	}
	return country, nil
}

func (s *CountryService) CreateCountry(ctx context.Context, req requests.CreateCountryRequest) error {
	converted := req.BaseCountryRequest.Convert()
	country := &models.Country{
		BaseModel: models.BaseModel{IsActive: converted.IsActive != nil && *converted.IsActive},
		Name:      converted.Name,
	}
	return s.repo.CreateCountry(ctx, country)
}

func (s *CountryService) UpdateCountry(ctx context.Context, id uint, req requests.UpdateCountryRequest) error {
	if _, err := s.repo.GetCountryByID(ctx, id); err != nil {
		return errors.New("ülke bulunamadı")
	}
	converted := req.BaseCountryRequest.Convert()
	updateData := map[string]interface{}{
		"name":      converted.Name,
		"is_active": converted.IsActive != nil && *converted.IsActive,
	}
	return s.repo.UpdateCountry(ctx, id, updateData)
}

func (s *CountryService) DeleteCountry(ctx context.Context, id uint) error {
	return s.repo.DeleteCountry(ctx, id)
}

func (s *CountryService) GetCountryCount(ctx context.Context) (int64, error) {
	return s.repo.GetCountryCount(ctx)
}
