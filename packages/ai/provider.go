package ai

import (
	"fmt"

	"github.com/zatrano/framework/bootstrap/addons"
	appconfig "github.com/zatrano/framework/config"
	"github.com/zatrano/framework/core"
	pkgconfig "github.com/zatrano/framework/packages/config"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "ai",
		Key:         "ai",
		Description: "AI chat providers",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the AI addon (registration only; driver logic is unchanged).
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	pkgconfig.LoadIfAbsent(app.Config(), "ai", appconfig.AI())
	mgr := New()
	var logFn LogFn
	if lg := app.Logger(); lg != nil {
		logFn = lg.Infof
	}
	cfg := map[string]any{}
	if raw := app.Config().Get("ai"); raw != nil {
		if m, ok := raw.(map[string]any); ok {
			cfg = m
		}
	}
	if cfg["api_key"] == nil || fmt.Sprint(cfg["api_key"]) == "" {
		cfg["api_key"] = app.Config().GetString("ai.api_key", env.Get("AI_API_KEY", env.Get("OPENAI_API_KEY", "")))
	}
	if cfg["driver"] == nil || fmt.Sprint(cfg["driver"]) == "" {
		cfg["driver"] = app.Config().GetString("ai.driver", env.Get("AI_DRIVER", ""))
	}
	if cfg["default"] == nil || fmt.Sprint(cfg["default"]) == "" {
		cfg["default"] = app.Config().GetString("ai.default", env.Get("AI_DEFAULT", env.Get("AI_DRIVER", "")))
	}
	if cfg["base_url"] == nil || fmt.Sprint(cfg["base_url"]) == "" {
		cfg["base_url"] = app.Config().GetString("ai.base_url", env.Get("AI_BASE_URL", ""))
	}
	if cfg["model"] == nil || fmt.Sprint(cfg["model"]) == "" {
		cfg["model"] = app.Config().GetString("ai.model", env.Get("AI_MODEL", ""))
	}
	if cfg["timeout"] == nil {
		cfg["timeout"] = app.Config().GetInt("ai.timeout", env.GetInt("AI_TIMEOUT", 30))
	}
	if cfg["temperature"] == nil || fmt.Sprint(cfg["temperature"]) == "" {
		cfg["temperature"] = app.Config().GetString("ai.temperature", env.Get("AI_TEMPERATURE", ""))
	}
	if cfg["max_tokens"] == nil {
		cfg["max_tokens"] = app.Config().GetInt("ai.max_tokens", env.GetInt("AI_MAX_TOKENS", 0))
	}
	if cfg["providers"] == nil {
		if raw := app.Config().Get("ai.providers"); raw != nil {
			cfg["providers"] = raw
		}
	}
	if cfg["profiles"] == nil {
		if raw := app.Config().Get("ai.profiles"); raw != nil {
			cfg["profiles"] = raw
		}
	}
	if err := mgr.BootConfig(cfg, logFn); err != nil {
		return err
	}
	app.Container().Instance("ai", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error {
	if app == nil || app.Router() == nil {
		return nil
	}
	mgr := From(app)
	if mgr == nil {
		return nil
	}
	app.Router().Post("/demo/ai/chat", DemoChatHandler(mgr)).As("demo.ai.chat")
	return nil
}
