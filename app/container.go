package app

import (
	"github.com/zatrano/framework/repositories"
	"github.com/zatrano/framework/services"
)

// Container — uygulama başlarken bir kez oluşturulan tüm servis ve repository instance'larını tutar.
// İşleyiciler bu container'dan beslenir; hiçbir New*() çağrısı işleyici veya rota içinde yapılmaz.
type Container struct {
	// ── Altyapı servisleri ────────────────────────────────────────────────────
	Mail services.IMailService
	JWT  services.IJWTService

	// ── Kimlik Doğrulama ───────────────────────────────────────────────────────
	Auth services.IAuthService

	// ── Domain servisleri ─────────────────────────────────────────────────────
	Address    services.IAddressService
	City       services.ICityService
	Contact    services.IContactService
	Country    services.ICountryService
	Definition services.IDefinitionService
	District   services.IDistrictService
	User       services.IUserService
	UserType   services.IUserTypeService

	// ── Repository'ler (doğrudan repo gerektiren işleyiciler için) ────────────
	UserRepo     repositories.IUserRepository
	CountryRepo  repositories.ICountryRepository
	CityRepo     repositories.ICityRepository
	DistrictRepo repositories.IDistrictRepository
	AddressRepo  repositories.IAddressRepository
}

// Build — tüm bağımlılıkları wire eder ve Container döner.
// Sıra önemlidir: önce repo'lar, sonra onlara bağlı servisler oluşturulur.
func Build() *Container {
	// ── Repositories ──────────────────────────────────────────────────────────
	authRepo := repositories.NewAuthRepository()
	addressRepo := repositories.NewAddressRepository()
	cityRepo := repositories.NewCityRepository()
	contactRepo := repositories.NewContactRepository()
	countryRepo := repositories.NewCountryRepository()
	defRepo := repositories.NewDefinitionRepository()
	districtRepo := repositories.NewDistrictRepository()
	userRepo := repositories.NewUserRepository()
	userTypeRepo := repositories.NewUserTypeRepository()

	// ── Altyapı servisleri ────────────────────────────────────────────────────
	mail := services.NewMailService()
	jwt := services.NewJWTService(userRepo)

	// ── Domain servisleri ─────────────────────────────────────────────────────
	return &Container{
		Mail: mail,
		JWT:  jwt,

		Auth: services.NewAuthService(authRepo, mail),

		Address:    services.NewAddressService(addressRepo),
		City:       services.NewCityService(cityRepo),
		Contact:    services.NewContactService(contactRepo),
		Country:    services.NewCountryService(countryRepo),
		Definition: services.NewDefinitionService(defRepo),
		District:   services.NewDistrictService(districtRepo),
		User:       services.NewUserService(userRepo),
		UserType:   services.NewUserTypeService(userTypeRepo),

		UserRepo:     userRepo,
		CountryRepo:  countryRepo,
		CityRepo:     cityRepo,
		DistrictRepo: districtRepo,
		AddressRepo:  addressRepo,
	}
}
