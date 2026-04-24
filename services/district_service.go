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

type IDistrictService interface {
	GetAllDistricts(ctx context.Context, params requests.DistrictListParams) (*requests.PaginatedResult, error)
	GetDistrictByID(ctx context.Context, id uint) (*models.District, error)
	CreateDistrict(ctx context.Context, req requests.CreateDistrictRequest) error
	UpdateDistrict(ctx context.Context, id uint, req requests.UpdateDistrictRequest) error
	DeleteDistrict(ctx context.Context, id uint) error
	GetDistrictCount(ctx context.Context) (int64, error)
}

type DistrictService struct {
	repo repositories.IDistrictRepository
}

func NewDistrictService(repo repositories.IDistrictRepository) IDistrictService {
	return &DistrictService{repo: repo}
}

func (s *DistrictService) GetAllDistricts(ctx context.Context, params requests.DistrictListParams) (*requests.PaginatedResult, error) {
	districts, totalCount, err := s.repo.GetAllDistricts(ctx, params)
	if err != nil {
		logconfig.Log.Error("İlçeler alınamadı", zap.Error(err))
		return nil, errors.New("ilçeler getirilirken bir hata oluştu")
	}
	return requests.CreatePaginatedResult(districts, totalCount, params.Page, params.PerPage), nil
}

func (s *DistrictService) GetDistrictByID(ctx context.Context, id uint) (*models.District, error) {
	district, err := s.repo.GetDistrictByID(ctx, id)
	if err != nil {
		logconfig.Log.Warn("İlçe bulunamadı", zap.Uint("district_id", id), zap.Error(err))
		return nil, errors.New("ilçe bulunamadı")
	}
	return district, nil
}

func (s *DistrictService) CreateDistrict(ctx context.Context, req requests.CreateDistrictRequest) error {
	converted := req.BaseDistrictRequest.Convert()
	if converted.CountryID == nil {
		return errors.New("ülke seçilmelidir")
	}
	if converted.CityID == nil {
		return errors.New("şehir seçilmelidir")
	}
	district := &models.District{
		BaseModel:  models.BaseModel{IsActive: converted.IsActive != nil && *converted.IsActive},
		CountryID: *converted.CountryID,
		CityID:     *converted.CityID,
		Name:       converted.Name,
	}
	return s.repo.CreateDistrict(ctx, district)
}

func (s *DistrictService) UpdateDistrict(ctx context.Context, id uint, req requests.UpdateDistrictRequest) error {
	if _, err := s.repo.GetDistrictByID(ctx, id); err != nil {
		return errors.New("ilçe bulunamadı")
	}
	converted := req.BaseDistrictRequest.Convert()
	if converted.CountryID == nil {
		return errors.New("ülke seçilmelidir")
	}
	if converted.CityID == nil {
		return errors.New("şehir seçilmelidir")
	}
	updateData := map[string]interface{}{
		"country_id": *converted.CountryID,
		"city_id":    *converted.CityID,
		"name":       converted.Name,
		"is_active":  converted.IsActive != nil && *converted.IsActive,
	}
	return s.repo.UpdateDistrict(ctx, id, updateData)
}

func (s *DistrictService) DeleteDistrict(ctx context.Context, id uint) error {
	return s.repo.DeleteDistrict(ctx, id)
}

func (s *DistrictService) GetDistrictCount(ctx context.Context) (int64, error) {
	return s.repo.GetDistrictCount(ctx)
}
