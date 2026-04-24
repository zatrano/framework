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

type IAddressService interface {
	GetAllAddresses(ctx context.Context, params requests.AddressListParams) (*requests.PaginatedResult, error)
	GetAddressByID(ctx context.Context, id uint) (*models.Address, error)
	CreateAddress(ctx context.Context, req requests.CreateAddressRequest) error
	UpdateAddress(ctx context.Context, id uint, req requests.UpdateAddressRequest) error
	DeleteAddress(ctx context.Context, id uint) error
	GetAddressCount(ctx context.Context) (int64, error)
}

type AddressService struct {
	repo repositories.IAddressRepository
}

func NewAddressService(repo repositories.IAddressRepository) IAddressService {
	return &AddressService{repo: repo}
}

func (s *AddressService) GetAllAddresses(ctx context.Context, params requests.AddressListParams) (*requests.PaginatedResult, error) {
	list, total, err := s.repo.GetAllAddresses(ctx, params)
	if err != nil {
		logconfig.Log.Error("Adresler alınamadı", zap.Error(err))
		return nil, errors.New("adresler getirilirken bir hata oluştu")
	}
	return requests.CreatePaginatedResult(list, total, params.Page, params.PerPage), nil
}

func (s *AddressService) GetAddressByID(ctx context.Context, id uint) (*models.Address, error) {
	m, err := s.repo.GetAddressByID(ctx, id)
	if err != nil {
		logconfig.Log.Warn("Adres bulunamadı", zap.Uint("id", id), zap.Error(err))
		return nil, errors.New("adres bulunamadı")
	}
	return m, nil
}

func (s *AddressService) CreateAddress(ctx context.Context, req requests.CreateAddressRequest) error {
	c := req.BaseAddressRequest.Convert()
	if c.CountryID == nil || c.CityID == nil || c.DistrictID == nil {
		return errors.New("ülke, şehir ve ilçe seçilmelidir")
	}
	m := &models.Address{
		BaseModel:  models.BaseModel{IsActive: c.IsActive != nil && *c.IsActive},
		CountryID:  *c.CountryID,
		CityID:     *c.CityID,
		DistrictID: *c.DistrictID,
		Detail:     c.Detail,
	}
	return s.repo.CreateAddress(ctx, m)
}

func (s *AddressService) UpdateAddress(ctx context.Context, id uint, req requests.UpdateAddressRequest) error {
	if _, err := s.repo.GetAddressByID(ctx, id); err != nil {
		return errors.New("adres bulunamadı")
	}
	c := req.BaseAddressRequest.Convert()
	if c.CountryID == nil || c.CityID == nil || c.DistrictID == nil {
		return errors.New("ülke, şehir ve ilçe seçilmelidir")
	}
	data := map[string]interface{}{
		"country_id":  *c.CountryID,
		"city_id":     *c.CityID,
		"district_id": *c.DistrictID,
		"detail":      c.Detail,
		"is_active":   c.IsActive != nil && *c.IsActive,
	}
	return s.repo.UpdateAddress(ctx, id, data)
}

func (s *AddressService) DeleteAddress(ctx context.Context, id uint) error {
	return s.repo.DeleteAddress(ctx, id)
}

func (s *AddressService) GetAddressCount(ctx context.Context) (int64, error) {
	return s.repo.GetAddressCount(ctx)
}
