package server

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/core"
	// zatrano:wire:imports:start
	// zatrano:wire:imports:end
)

// registerModules mounts modules that live inside this repository (see `zatrano gen module --wire`).
func registerModules(a *core.App, app *fiber.App) {
	// zatrano:wire:register:start
	// zatrano:wire:register:end
}
