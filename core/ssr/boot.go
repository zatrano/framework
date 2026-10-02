package ssr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zatrano/canvas"
	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel/dirs"
)

func boot(app contracts.App) error {
	engine := New(dirs.TemplatesDir(app))
	engine.EnableCache(!app.IsDebug())
	engine.Share("appName", app.Config().GetString("app.name", "ZATRANO"))
	wireTranslator(engine, app)
	wireAssets(engine, app)
	engine.SetEnvironment(app.Environment())
	app.Container().Instance(ContainerKey, engine)
	installHTTPBridge(app)
	return nil
}

type translatorAPI interface {
	GetLocale() string
	Get(key string, replace ...map[string]string) string
	GetFor(locale, key string, replace ...map[string]string) string
	Choice(key string, number int, replace ...map[string]string) string
}

type assetsAPI interface {
	URL(path string) string
}

func wireTranslator(engine *canvas.Engine, app contracts.App) {
	raw, err := app.Make("translator")
	if err != nil {
		return
	}
	tr, ok := raw.(translatorAPI)
	if !ok || tr == nil {
		return
	}
	engine.Share("locale", tr.GetLocale())
	engine.AddFunc("trans", func(localeOrKey string, args ...any) string {
		locale := ""
		key := localeOrKey
		var replace map[string]string
		if len(args) == 0 {
			return tr.Get(key)
		}
		if s, ok := args[0].(string); ok {
			locale = localeOrKey
			key = s
			if len(args) > 1 {
				replace = coerceStringMap(args[1])
			}
			return tr.GetFor(locale, key, replace)
		}
		replace = coerceStringMap(args[0])
		return tr.Get(key, replace)
	})
	engine.AddFunc("dict", func(pairs ...any) map[string]any {
		out := map[string]any{}
		for i := 0; i+1 < len(pairs); i += 2 {
			out[fmt.Sprint(pairs[i])] = pairs[i+1]
		}
		return out
	})
	engine.AddFunc("choice", func(key string, number any) string {
		n := 0
		switch v := number.(type) {
		case int:
			n = v
		case int64:
			n = int(v)
		case float64:
			n = int(v)
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				n = parsed
			}
		default:
			n, _ = strconv.Atoi(fmt.Sprint(number))
		}
		return tr.Choice(key, n)
	})
}

func wireAssets(engine *canvas.Engine, app contracts.App) {
	raw, err := app.Make("assets")
	if err != nil {
		return
	}
	a, ok := raw.(assetsAPI)
	if !ok || a == nil {
		return
	}
	engine.AddFunc("vite", func(path string) string {
		return a.URL(path)
	})
	engine.AddFunc("mix", func(path string) string {
		return a.URL(path)
	})
}

func coerceStringMap(v any) map[string]string {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]string); ok {
		return m
	}
	if m, ok := v.(map[string]any); ok {
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = fmt.Sprint(val)
		}
		return out
	}
	return nil
}
