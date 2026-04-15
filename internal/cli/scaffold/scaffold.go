package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Options for generating a new application.
type Options struct {
	Dir         string
	AppName     string
	Module      string
	ZatranoPath string // if set, go.mod gets a replace directive for local ZATRANO development
}

type fileT struct {
	rel  string
	tmpl string
}

// Run writes the starter layout to Dir.
func Run(opts Options) error {
	if strings.TrimSpace(opts.Dir) == "" {
		return fmt.Errorf("output directory is required")
	}
	if strings.TrimSpace(opts.AppName) == "" {
		return fmt.Errorf("app name is required")
	}
	if strings.TrimSpace(opts.Module) == "" {
		return fmt.Errorf("module path is required (e.g. github.com/acme/myapp)")
	}
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return err
	}

	rep := strings.TrimSpace(opts.ZatranoPath)
	if rep != "" {
		rep = filepath.ToSlash(rep)
		if strings.ContainsAny(rep, " \t") {
			rep = `"` + rep + `"`
		}
	}
	data := map[string]any{
		"Module":      opts.Module,
		"AppName":     opts.AppName,
		"ReplacePath": rep,
	}

	files := []fileT{
		{"go.mod", tplGoMod},
		{filepath.Join("internal", "routes", "register.go"), tplRoutesRegister},
		{filepath.Join("cmd", opts.AppName, "main.go"), tplMain},
		{filepath.Join("config", "examples", "dev.yaml"), tplDevYAML},
		{filepath.Join("locales", "en.json"), tplLocalesEn},
		{filepath.Join("locales", "tr.json"), tplLocalesTr},
		{filepath.Join("api", "openapi.yaml"), tplOpenAPI},
		{filepath.Join("migrations", "000001_init.up.sql"), tplMigrationUp},
		{filepath.Join("migrations", "000001_init.down.sql"), tplMigrationDown},
		{filepath.Join("db", "seeds", ".gitkeep"), ""},
		{"README.md", tplReadme},
	}

	for _, f := range files {
		out := filepath.Join(opts.Dir, f.rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if f.tmpl == "" {
			if err := os.WriteFile(out, []byte{}, 0o644); err != nil {
				return err
			}
			continue
		}
		t, err := template.New(f.rel).Parse(f.tmpl)
		if err != nil {
			return fmt.Errorf("parse %s: %w", f.rel, err)
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			return fmt.Errorf("render %s: %w", f.rel, err)
		}
		if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
			return err
		}
	}

	return nil
}

const tplGoMod = `module {{.Module}}

go 1.25.0

require github.com/zatrano/framework v0.0.0

{{- if .ReplacePath}}

replace github.com/zatrano/framework => {{.ReplacePath}}
{{- end}}
`

const tplMain = `package main

import (
	"log"

	"{{.Module}}/internal/routes"
	"github.com/zatrano/framework/pkg/zatrano"
)

func main() {
	if err := zatrano.Start(zatrano.StartOptions{
		RegisterRoutes: routes.Register,
	}); err != nil {
		log.Fatal(err)
	}
}
`

const tplRoutesRegister = `package routes

import (
	"github.com/gofiber/fiber/v3"

	"github.com/zatrano/framework/pkg/core"
	// zatrano:wire:imports:start
	// zatrano:wire:imports:end
)

// Register mounts application modules (updated by zatrano gen module / gen crud).
func Register(a *core.App, app *fiber.App) {
	// zatrano:wire:register:start
	// zatrano:wire:register:end
}
`

const tplDevYAML = `env: dev
app_name: {{.AppName}}

http_addr: ":8080"
http_read_timeout: 30s

# http:
#   cors_enabled: true
#   cors_allow_origins: ["http://localhost:5173"]

# i18n:
#   enabled: true
#   default_locale: en
#   supported_locales: [en, tr]
#   locales_dir: locales

database_url: ""
database_required: false

redis_url: ""
redis_required: false

log_level: info
log_development: true

migrations_dir: migrations
seeds_dir: db/seeds
openapi_path: api/openapi.yaml

security:
  session_enabled: true
  csrf_enabled: true
  csrf_skip_prefixes:
    - /api/
  jwt_secret: "change-me-in-dev-only"
  jwt_issuer: zatrano
  jwt_expiry: 60m
  cookie_secure: false
  demo_token_endpoint: true

static_path: public
static_url_prefix: /static
`

const tplLocalesEn = `{
  "app": {
    "welcome": "Welcome to {{.AppName}}"
  },
  "validation": {
    "required": "This field is required",
    "email": "Must be a valid email address",
    "min": "Must be at least {{"{{.Param}}"}} characters",
    "max": "Must be at most {{"{{.Param}}"}} characters",
    "gte": "Must be greater than or equal to {{"{{.Param}}"}}",
    "lte": "Must be less than or equal to {{"{{.Param}}"}}",
    "len": "Must be exactly {{"{{.Param}}"}} characters",
    "url": "Must be a valid URL",
    "uuid": "Must be a valid UUID",
    "oneof": "Must be one of: {{"{{.Param}}"}}",
    "numeric": "Must be numeric",
    "alpha": "Must contain only letters",
    "alphanum": "Must contain only letters and numbers"
  }
}
`

const tplLocalesTr = `{
  "app": {
    "welcome": "{{.AppName}} uygulamasına hoş geldiniz"
  },
  "validation": {
    "required": "Bu alan zorunludur",
    "email": "Geçerli bir e-posta adresi olmalıdır",
    "min": "En az {{"{{.Param}}"}} karakter olmalıdır",
    "max": "En fazla {{"{{.Param}}"}} karakter olmalıdır",
    "gte": "{{"{{.Param}}"}} değerinden büyük veya eşit olmalıdır",
    "lte": "{{"{{.Param}}"}} değerinden küçük veya eşit olmalıdır",
    "len": "Tam olarak {{"{{.Param}}"}} karakter olmalıdır",
    "url": "Geçerli bir URL olmalıdır",
    "uuid": "Geçerli bir UUID olmalıdır",
    "oneof": "Şunlardan biri olmalıdır: {{"{{.Param}}"}}",
    "numeric": "Sayısal bir değer olmalıdır",
    "alpha": "Sadece harf içermelidir",
    "alphanum": "Sadece harf ve rakam içermelidir"
  }
}
`

const tplOpenAPI = `openapi: 3.0.3
info:
  title: {{.AppName}} API
  version: 0.1.0
paths:
  /api/v1/public/ping:
    get:
      summary: Ping
      responses:
        "200":
          description: OK
`

const tplMigrationUp = `-- {{.AppName}} initial migration
CREATE TABLE IF NOT EXISTS app_hello (
    id bigserial PRIMARY KEY,
    message text NOT NULL DEFAULT 'hello from {{.AppName}}',
    created_at timestamptz NOT NULL DEFAULT now()
);

`

const tplMigrationDown = `DROP TABLE IF EXISTS app_hello;
`

const tplReadme = `# {{.AppName}}

Generated by [ZATRANO](https://github.com/zatrano/framework).

## Run

` + "```bash" + `
cp config/examples/dev.yaml config/dev.yaml
go mod tidy
go run ./cmd/{{.AppName}}
` + "```" + `

Set ` + "`DATABASE_URL`" + ` and ` + "`REDIS_URL`" + ` when you enable Postgres/Redis.

## Modules

` + "```bash" + `
zatrano gen module my_feature
go fmt ./...
` + "```" + `

This updates ` + "`internal/routes/register.go`" + ` and runs ` + "`go fmt`" + ` on it. Use ` + "`--skip-wire`" + ` to only generate files under ` + "`modules/`" + `, then ` + "`zatrano gen wire <name>`" + ` when ready.

## Checks (optional)

` + "```bash" + `
zatrano verify
` + "```" + `

Runs ` + "`go vet`" + `, ` + "`go test`" + `, and merged OpenAPI validation (install ` + "`zatrano`" + ` on PATH first).

## Migrations & seeds

` + "```bash" + `
go install github.com/zatrano/framework/cmd/zatrano@latest
zatrano db migrate
zatrano db seed   # after adding .sql files under db/seeds/
` + "```" + `
`
