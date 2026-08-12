package api

import (
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/version"
)

// HomeController serves API root information.
type HomeController struct{}

// Index returns framework name and version.
func (c *HomeController) Index(req *http.Request) *http.Response {
	return http.JSON(map[string]any{
		"name":    "ZATRANO",
		"version": version.Get(),
	})
}
