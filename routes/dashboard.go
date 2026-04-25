package routes

import (
	"github.com/zatrano/framework/app"
	handlers "github.com/zatrano/framework/handlers/dashboard"
	"github.com/zatrano/framework/middlewares"

	"github.com/gofiber/fiber/v3"
)

func registerDashboardRoutes(fiberApp *fiber.App, c *app.Container) {
	dashboardGroup := fiberApp.Group("/dashboard")
	dashboardGroup.Use(
		middlewares.SessionMiddleware(),
		middlewares.AuthMiddleware(c.Auth),
		middlewares.UserTypeMiddleware(1),
	)

	dashboardHomeHandler := handlers.NewDashboardHomeHandler(
		c.UserRepo, c.CountryRepo, c.CityRepo, c.DistrictRepo, c.AddressRepo,
	)
	dashboardGroup.Get("/", dashboardHomeHandler.HomePage)

	userTypeHandler := handlers.NewDashboardUserTypeHandler(c.UserType)
	dashboardGroup.Get("/user-types", userTypeHandler.ListUserTypes)
	dashboardGroup.Get("/user-types/create", userTypeHandler.ShowCreateUserType)
	dashboardGroup.Post("/user-types/create", userTypeHandler.CreateUserType)
	dashboardGroup.Get("/user-types/update/:id", userTypeHandler.ShowUpdateUserType)
	dashboardGroup.Post("/user-types/update/:id", userTypeHandler.UpdateUserType)
	dashboardGroup.Delete("/user-types/delete/:id", userTypeHandler.DeleteUserType)

	countryHandler := handlers.NewDashboardCountryHandler(c.Country)
	dashboardGroup.Get("/countries", countryHandler.ListCountries)
	dashboardGroup.Get("/countries/create", countryHandler.ShowCreateCountry)
	dashboardGroup.Post("/countries/create", countryHandler.CreateCountry)
	dashboardGroup.Get("/countries/update/:id", countryHandler.ShowUpdateCountry)
	dashboardGroup.Post("/countries/update/:id", countryHandler.UpdateCountry)
	dashboardGroup.Delete("/countries/delete/:id", countryHandler.DeleteCountry)

	cityHandler := handlers.NewDashboardCityHandler(c.City, c.Country)
	dashboardGroup.Get("/cities", cityHandler.ListCities)
	dashboardGroup.Get("/cities/create", cityHandler.ShowCreateCity)
	dashboardGroup.Post("/cities/create", cityHandler.CreateCity)
	dashboardGroup.Get("/cities/update/:id", cityHandler.ShowUpdateCity)
	dashboardGroup.Post("/cities/update/:id", cityHandler.UpdateCity)
	dashboardGroup.Delete("/cities/delete/:id", cityHandler.DeleteCity)

	districtHandler := handlers.NewDashboardDistrictHandler(c.District, c.City, c.Country)
	dashboardGroup.Get("/districts", districtHandler.ListDistricts)
	dashboardGroup.Get("/districts/create", districtHandler.ShowCreateDistrict)
	dashboardGroup.Post("/districts/create", districtHandler.CreateDistrict)
	dashboardGroup.Get("/districts/update/:id", districtHandler.ShowUpdateDistrict)
	dashboardGroup.Post("/districts/update/:id", districtHandler.UpdateDistrict)
	dashboardGroup.Delete("/districts/delete/:id", districtHandler.DeleteDistrict)

	addressHandler := handlers.NewDashboardAddressHandler(c.Address, c.District, c.City, c.Country)
	dashboardGroup.Get("/addresses", addressHandler.ListAddresses)
	dashboardGroup.Get("/addresses/create", addressHandler.ShowCreateAddress)
	dashboardGroup.Post("/addresses/create", addressHandler.CreateAddress)
	dashboardGroup.Get("/addresses/update/:id", addressHandler.ShowUpdateAddress)
	dashboardGroup.Post("/addresses/update/:id", addressHandler.UpdateAddress)
	dashboardGroup.Delete("/addresses/delete/:id", addressHandler.DeleteAddress)

	contactMsgHandler := handlers.NewDashboardContactMessageHandler(c.Contact)
	dashboardGroup.Get("/contact-messages", contactMsgHandler.List)
	dashboardGroup.Get("/contact-messages/:id", contactMsgHandler.Show)

	userHandler := handlers.NewDashboardUserHandler(c.User, c.UserType)
	dashboardGroup.Get("/users", userHandler.ListUsers)
	dashboardGroup.Get("/users/create", userHandler.ShowCreateUser)
	dashboardGroup.Post("/users/create", userHandler.CreateUser)
	dashboardGroup.Get("/users/update/:id", userHandler.ShowUpdateUser)
	dashboardGroup.Post("/users/update/:id", userHandler.UpdateUser)
	dashboardGroup.Delete("/users/delete/:id", userHandler.DeleteUser)

	definitionHandler := handlers.NewDashboardDefinitionHandler(c.Definition)
	dashboardGroup.Get("/definitions", definitionHandler.ListDefinitions)
	dashboardGroup.Get("/definitions/create", definitionHandler.ShowCreateDefinition)
	dashboardGroup.Post("/definitions/create", definitionHandler.CreateDefinition)
	dashboardGroup.Get("/definitions/update/:id", definitionHandler.ShowUpdateDefinition)
	dashboardGroup.Post("/definitions/update/:id", definitionHandler.UpdateDefinition)
	dashboardGroup.Delete("/definitions/delete/:id", definitionHandler.DeleteDefinition)
}
