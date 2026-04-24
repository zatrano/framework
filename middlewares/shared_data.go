package middlewares

import (
	"github.com/zatrano/framework/services"

	"github.com/gofiber/fiber/v3"
)

// SharedDataMiddleware, tüm isteklerde (auth, panel, dashboard, website, API HTML vb.)
// şablonlara giden ortak veriyi Locals’ta üretir.
// DefinitionValues: yalnızca burada set edilir; handler render verisine eklemez (renderer prepareRenderData Locals’tan birleştirir).
// MenuPages: website menü iskeleti (şimdilik boş; layout kullanımına bırakılır).
func SharedDataMiddleware(definitionService services.IDefinitionService) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("MenuPages", []interface{}{})
		c.Locals("DefinitionValues", definitionService.GetMap(c.Context()))
		return c.Next()
	}
}
