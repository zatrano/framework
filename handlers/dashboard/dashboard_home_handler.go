package handlers

import (
	"net/http"

	"github.com/zatrano/framework/packages/renderer"
	"github.com/zatrano/framework/repositories"

	"github.com/gofiber/fiber/v3"
)

type DashboardHomeHandler struct {
	userRepo    repositories.IUserRepository
	countryRepo repositories.ICountryRepository
	cityRepo    repositories.ICityRepository
	districtRepo repositories.IDistrictRepository
	addressRepo repositories.IAddressRepository
}

func NewDashboardHomeHandler(
	userRepo repositories.IUserRepository,
	countryRepo repositories.ICountryRepository,
	cityRepo repositories.ICityRepository,
	districtRepo repositories.IDistrictRepository,
	addressRepo repositories.IAddressRepository,
) *DashboardHomeHandler {
	return &DashboardHomeHandler{
		userRepo:     userRepo,
		countryRepo:  countryRepo,
		cityRepo:     cityRepo,
		districtRepo: districtRepo,
		addressRepo:  addressRepo,
	}
}

func (h *DashboardHomeHandler) HomePage(c fiber.Ctx) error {
	
	userCount, _ := h.userRepo.GetUserCount(c.Context())
	countryCount, _ := h.countryRepo.GetCountryCount(c.Context())
	cityCount, _ := h.cityRepo.GetCityCount(c.Context())
	districtCount, _ := h.districtRepo.GetDistrictCount(c.Context())
	addressCount, _ := h.addressRepo.GetAddressCount(c.Context())

	return renderer.Render(c, "dashboard/home/home", "layouts/app", fiber.Map{
		"Title":         "Dashboard",
		"UserCount":     int(userCount),
		"CountryCount":  int(countryCount),
		"CityCount":     int(cityCount),
		"DistrictCount": int(districtCount),
		"AddressCount":  int(addressCount),
	}, http.StatusOK)
}
