package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CRUD generates REST-style handler stubs, RegisterCRUD, and form request structs under modules/<name>/.
func CRUD(moduleRoot, baseDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid resource name %q", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)
	base := filepath.Join(baseDir, name)
	seg := strings.ReplaceAll(name, "_", "-")
	plural := seg + "s"
	route := "/api/v1/" + plural
	reqImport := modPath + "/" + filepath.ToSlash(filepath.Join(baseDir, name, "requests"))

	files := map[string]string{
		"crud_handlers.go":                       tmplCRUDHandlers(name, pascal, plural, reqImport),
		"crud_register.go":                       tmplCRUDRegister(name, pascal, route, modPath),
		filepath.Join("requests", "create_"+name+".go"): tmplCRUDCreateRequest(pascal),
		filepath.Join("requests", "update_"+name+".go"): tmplCRUDUpdateRequest(pascal),
	}
	var written []string
	for fn, body := range files {
		path := filepath.Join(base, fn)
		written = append(written, path)
		if dryRun {
			continue
		}
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, err
		}
	}
	return written, nil
}

func tmplCRUDHandlers(pkg, pascal, plural, reqImport string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/gofiber/fiber/v3"

	"%[4]s"
	"github.com/zatrano/framework/pkg/repository"
	"github.com/zatrano/framework/pkg/zatrano"
)

// %[2]sCRUDHandler is a starter REST layer for %[1]s (replace with DTOs and service calls).
type %[2]sCRUDHandler struct{}

// New%[2]sCRUDHandler constructs the handler.
func New%[2]sCRUDHandler() *%[2]sCRUDHandler {
	return &%[2]sCRUDHandler{}
}

// List returns a placeholder collection.
func (h *%[2]sCRUDHandler) List(c fiber.Ctx) error {
	// Example eager loading to prevent N+1 queries:
	// scopes := repository.Scopes(repository.PreloadAll())
	return c.JSON(fiber.Map{"items": []any{}, "resource": %[3]q})
}

// Show returns one item by ID from the path.
func (h *%[2]sCRUDHandler) Show(c fiber.Ctx) error {
	id := c.Params("id")
	// Example specific preload:
	// scopes := repository.Scopes(repository.Preload("Profile"))
	return c.JSON(fiber.Map{"id": id, "resource": %[3]q})
}

// Create accepts JSON body with validation.
func (h *%[2]sCRUDHandler) Create(c fiber.Ctx) error {
	req, err := zatrano.Validate[requests.Create%[2]sRequest](c)
	if err != nil {
		return err
	}
	_ = req // TODO: pass to service layer
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"created": true, "resource": %[3]q})
}

// Update replaces a resource by ID with validation.
func (h *%[2]sCRUDHandler) Update(c fiber.Ctx) error {
	id := c.Params("id")
	req, err := zatrano.Validate[requests.Update%[2]sRequest](c)
	if err != nil {
		return err
	}
	_ = req // TODO: pass to service layer
	return c.JSON(fiber.Map{"id": id, "updated": true, "resource": %[3]q})
}

// Destroy removes a resource by ID.
func (h *%[2]sCRUDHandler) Destroy(c fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{"id": id, "deleted": true, "resource": %[3]q})
}
`, pkg, pascal, plural, reqImport)
}

func tmplCRUDRegister(pkg, pascal, route, moduleImport string) string {
	return fmt.Sprintf(`package %[1]s

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/core"
)

// RegisterCRUD mounts REST routes for %[1]s.
// Wire with: zatrano gen crud %[1]s --wire (import %[4]s/modules/%[1]s).
func RegisterCRUD(a *core.App, app *fiber.App) {
	_ = a // reserved for DB/service injection
	h := New%[2]sCRUDHandler()
	g := app.Group("%[3]s")
	g.Get("/", h.List)
	g.Get("/:id", h.Show)
	g.Post("/", h.Create)
	g.Put("/:id", h.Update)
	g.Delete("/:id", h.Destroy)
}
`, pkg, pascal, route, moduleImport)
}

func tmplCRUDCreateRequest(pascal string) string {
	return fmt.Sprintf(`package requests

// Create%[1]sRequest is the form request for creating a %[1]s.
// Add your fields and validation tags below.
//
// Struct tags use go-playground/validator syntax.
// See: https://pkg.go.dev/github.com/go-playground/validator/v10
//
// Usage in handlers:
//
//	req, err := zatrano.Validate[requests.Create%[1]sRequest](c)
//	if err != nil { return err }
type Create%[1]sRequest struct {
	Name  string `+"`"+`json:"name"  validate:"required,min=2,max=255"`+"`"+`
	Email string `+"`"+`json:"email" validate:"required,email"`+"`"+`
}
`, pascal)
}

func tmplCRUDUpdateRequest(pascal string) string {
	return fmt.Sprintf(`package requests

// Update%[1]sRequest is the form request for updating a %[1]s.
// Add your fields and validation tags below.
//
// Usage in handlers:
//
//	req, err := zatrano.Validate[requests.Update%[1]sRequest](c)
//	if err != nil { return err }
type Update%[1]sRequest struct {
	Name  string `+"`"+`json:"name"  validate:"omitempty,min=2,max=255"`+"`"+`
	Email string `+"`"+`json:"email" validate:"omitempty,email"`+"`"+`
}
`, pascal)
}

