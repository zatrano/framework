package handlers

import (
	"net/http"

	"github.com/zatrano/framework/packages/currentuser"
	"github.com/zatrano/framework/packages/renderer"

	"github.com/gofiber/fiber/v3"
)

type PanelHomeHandler struct{}

func NewPanelHomeHandler() *PanelHomeHandler {
	return &PanelHomeHandler{}
}

// HomePage — Panel ana sayfası (admin dışı kullanıcılar için).
func (h *PanelHomeHandler) HomePage(c fiber.Ctx) error {
	cu := currentuser.FromFiber(c)
	name := ""
	if cu.ID > 0 {
		name = cu.Email
	}
	return renderer.Render(c, "panel/home", "layouts/app", fiber.Map{
		"Title":       "Panel",
		"UserEmail":   name,
		"WelcomeText": "github.com/zatrano/framework kullanıcı paneline hoş geldiniz.",
	}, http.StatusOK)
}
