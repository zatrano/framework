package web

import "github.com/zatrano/framework/packages/http"

// HomeController serves the welcome page.
type HomeController struct{}

// Index renders the welcome view.
func (c *HomeController) Index(req *http.Request) *http.Response {
	return http.View("welcome", map[string]any{})
}
