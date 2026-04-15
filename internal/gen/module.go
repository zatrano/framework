package gen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Module writes handler/service/repository/register stubs under baseDir/<name>.
// moduleRoot must contain go.mod (used for import paths in generated code).
func Module(moduleRoot, baseDir, rawName string, dryRun bool) ([]string, error) {
	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("invalid module name %q (use letters, digits, _ or -)", rawName)
	}
	modPath, err := ModuleImportPath(moduleRoot)
	if err != nil {
		return nil, err
	}
	pascal := snakeToPascal(name)
	base := filepath.Join(baseDir, name)
	files := map[string]string{
		"repository.go": tmplRepository(name, pascal),
		"service.go":    tmplService(name, pascal),
		"handler.go":    tmplHandler(name, pascal),
		"register.go":   tmplRegister(name, pascal, modPath),
	}
	var written []string
	for fn, body := range files {
		path := filepath.Join(base, fn)
		written = append(written, path)
		if dryRun {
			continue
		}
		if err := os.MkdirAll(base, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			return nil, err
		}
	}
	return written, nil
}

// PackageName returns the snake_case package/directory name for gen module/crud.
func PackageName(raw string) string {
	return normalizeName(raw)
}

func normalizeName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, "-", "_")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '_', unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "_")
}

func snakeToPascal(s string) string {
	parts := strings.Split(s, "_")
	var out strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		r := []rune(p)
		out.WriteRune(unicode.ToUpper(r[0]))
		if len(r) > 1 {
			out.WriteString(string(r[1:]))
		}
	}
	return out.String()
}

func tmplRepository(pkg, pascal string) string {
	return fmt.Sprintf(`package %s

import (
	"context"

	"gorm.io/gorm"
)

// %[2]sRepository is the data access layer.
type %[2]sRepository struct {
	db *gorm.DB
}

// New%[2]sRepository constructs a repository.
func New%[2]sRepository(db *gorm.DB) *%[2]sRepository {
	return &%[2]sRepository{db: db}
}

// Ping verifies database connectivity for this module.
func (r *%[2]sRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
`, pkg, pascal)
}

func tmplService(pkg, pascal string) string {
	return fmt.Sprintf(`package %s

import (
	"context"
)

// %[2]sService contains business logic.
type %[2]sService struct {
	repo *%[2]sRepository
}

// New%[2]sService constructs a service.
func New%[2]sService(repo *%[2]sRepository) *%[2]sService {
	return &%[2]sService{repo: repo}
}

// Health is a placeholder — replace with real workflows.
func (s *%[2]sService) Health(ctx context.Context) error {
	return s.repo.Ping(ctx)
}
`, pkg, pascal)
}

func tmplHandler(pkg, pascal string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/gofiber/fiber/v3"
)

// %[2]sHandler exposes HTTP handlers.
type %[2]sHandler struct {
	svc *%[2]sService
}

// New%[2]sHandler constructs a handler.
func New%[2]sHandler(svc *%[2]sService) *%[2]sHandler {
	return &%[2]sHandler{svc: svc}
}

// List is a starter collection endpoint.
func (h *%[2]sHandler) List(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"items": []any{},
		"meta":  fiber.Map{"module": %[1]q},
	})
}
`, pkg, pascal)
}

func tmplRegister(pkg, pascal, moduleImport string) string {
	seg := strings.ReplaceAll(pkg, "_", "-")
	route := "/api/v1/" + seg + "s"
	return fmt.Sprintf(`package %[1]s

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/core"
)

// Register mounts HTTP routes for this module.
// Wire with: zatrano gen module %[1]s --wire (or add %[1]s.Register to your routes file).
// Import path: %[4]s/modules/%[1]s
func Register(a *core.App, app *fiber.App) {
	if a.DB == nil {
		return
	}
	repo := New%[2]sRepository(a.DB)
	svc := New%[2]sService(repo)
	h := New%[2]sHandler(svc)

	g := app.Group("%[3]s")
	g.Get("/", h.List)
}
`, pkg, pascal, route, moduleImport)
}
