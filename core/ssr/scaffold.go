package ssr

import (
	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/ssr/starter"
)

// WriteStarterTemplates writes templates/layouts/app.html and templates/web/welcome.html when missing.
func WriteStarterTemplates(app contracts.App) error {
	return starter.Write(app)
}
