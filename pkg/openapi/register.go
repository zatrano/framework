package openapi

import (
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
	yaml "go.yaml.in/yaml/v3"

	"github.com/zatrano/framework/pkg/core"
)

// Register exposes merged OpenAPI (static file + framework routes) and Scalar at /docs.
func Register(a *core.App, app *fiber.App) {
	path := strings.TrimSpace(a.Config.OpenAPIPath)
	if path == "" {
		path = "api/openapi.yaml"
	}

	app.Get("/openapi.yaml", func(c fiber.Ctx) error {
		b, err := mergedYAMLBytes(path, a.Log)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		c.Set(fiber.HeaderContentType, "application/yaml")
		return c.Send(b)
	})

	app.Get("/docs", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
		return c.SendString(scalarPage())
	})
}

// MergedYAMLBytes loads base OpenAPI (if present), merges framework routes, returns YAML bytes.
func MergedYAMLBytes(path string) ([]byte, error) {
	return mergedYAMLBytes(path, nil)
}

// mergedYAMLBytes loads the base spec from disk (if present), merges framework routes, returns YAML.
func mergedYAMLBytes(path string, log *zap.Logger) ([]byte, error) {
	loader := openapi3.NewLoader()
	var doc *openapi3.T
	raw, err := os.ReadFile(path)
	if err != nil {
		if log != nil {
			log.Info("openapi: base file not found, using minimal document + framework routes",
				zap.String("path", path),
			)
		}
		doc = minimalDoc()
	} else {
		doc, err = loader.LoadFromData(raw)
		if err != nil {
			return nil, err
		}
	}
	MergeFrameworkRoutes(doc)
	anyDoc, err := doc.MarshalYAML()
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(anyDoc)
}

func minimalDoc() *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "ZATRANO",
			Description: "Merged spec (add your own api/openapi.yaml).",
			Version:     "0.1.0",
		},
	}
}

func scalarPage() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>ZATRANO API</title>
</head>
<body>
  <script
    id="api-reference"
    data-url="/openapi.yaml"
    data-configuration='{"theme": "purple"}'></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.25.0"></script>
</body>
</html>`
}
