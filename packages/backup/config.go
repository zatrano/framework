package backup

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/database"
)

// ConfigGetter is satisfied by *config.Repository.
type ConfigGetter interface {
	GetString(key string, fallback ...string) string
	Get(key string, fallback ...any) any
}

// BasePather is satisfied by *kernel.Application.
type BasePather interface {
	BasePath(parts ...string) string
}

// ConfigFromApp builds a backup Config for the named (or default) database connection.
func ConfigFromApp(app BasePather, cfg ConfigGetter, connection string) (Config, error) {
	if cfg == nil {
		return Config{}, fmt.Errorf("backup: config unavailable")
	}
	defaultName := database.NormalizeDriverName(cfg.GetString("database.default", "sqlite"))
	name := strings.TrimSpace(connection)
	if name == "" {
		name = defaultName
	} else {
		if !validBackupConnectionName(name) {
			return Config{}, fmt.Errorf("backup: invalid connection name")
		}
		name = database.NormalizeDriverName(name)
	}

	prefix := "database.connections." + name + "."
	driver := database.NormalizeDriverName(cfg.GetString(prefix+"driver", name))
	if driver == "" {
		driver = name
	}

	out := Config{
		Driver:   driver,
		Host:     cfg.GetString(prefix + "host"),
		Port:     cfg.GetString(prefix + "port"),
		Database: cfg.GetString(prefix + "database"),
		Username: cfg.GetString(prefix + "username"),
		Password: cfg.GetString(prefix + "password"),
		Charset:  cfg.GetString(prefix + "charset"),
		SSLMode:  cfg.GetString(prefix + "sslmode"),
		Service:  cfg.GetString(prefix + "service"),
		URI:      cfg.GetString(prefix + "uri"),
		Dir:      app.BasePath("storage", "backups"),
		BasePath: app.BasePath(),
	}
	if out.Database == "" && driver == "sqlite" {
		out.Database = "database/database.sqlite"
	}
	if out.Driver == "" {
		return Config{}, fmt.Errorf("backup: connection %q not configured", name)
	}
	return out, nil
}

// ManagerFromApp resolves Config and returns a Manager for the connection.
func ManagerFromApp(app *kernel.Application, connection string) (*Manager, error) {
	if app == nil {
		return nil, fmt.Errorf("backup: application unavailable")
	}
	cfg, err := ConfigFromApp(app, app.Config(), connection)
	if err != nil {
		return nil, err
	}
	return NewManager(cfg), nil
}

func validBackupConnectionName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}
