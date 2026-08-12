package foundation

import (
	"fmt"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/authorization"
	"github.com/zatrano/framework/packages/flash"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/localization"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/session"
	"github.com/zatrano/framework/packages/validation"
	"github.com/zatrano/framework/packages/view"
)

type httpBridge struct {
	app *core.Application
}

func installHTTPBridge(app *core.Application) {
	app.SetHTTPBridge(&httpBridge{app: app})
}

func (b *httpBridge) Middleware() []routing.MiddlewareFunc {
	return []routing.MiddlewareFunc{
		b.sessionMiddleware(),
		b.localeMiddleware(),
	}
}

func (b *httpBridge) Finalize(w stdhttp.ResponseWriter, req *http.Request, resp *http.Response) *http.Response {
	app := b.app
	if resp == nil {
		resp = http.Abort(204)
	}

	engine := view.From(app)
	if resp.ViewName() != "" && engine != nil {
		data := resp.ViewData()
		if data == nil {
			data = map[string]any{}
		}
		if sess := req.Session(); sess != nil {
			if token, ok := sess.Get("_csrf_token").(string); ok {
				data["_token"] = token
			}
			data["flash"] = flash.All(req)
			data["old"] = flash.OldInput(req)
			data["errors"] = validation.ErrorsFromSession(req)
			data["errorBags"] = validation.ErrorBagsFromSession(req)
		} else {
			data["old"] = map[string]string{}
			data["errors"] = validation.NewMessageBag(nil)
			data["errorBags"] = map[string]any{}
		}
		authMgr := auth.From(app)
		authenticated := authMgr != nil && authMgr.Check(req)
		data["auth"] = authenticated
		data["guest"] = !authenticated
		if tr := localization.From(app); tr != nil {
			data["locale"] = tr.GetLocale()
			langPath := app.BasePath("lang")
			data["langPublished"] = localization.Published(langPath)
			data["locales"] = localization.Options(langPath, tr.GetLocale())
		}
		var user authorization.Authenticatable
		if authenticated {
			if u := authMgr.User(req); u != nil {
				if a, ok := any(u).(authorization.Authenticatable); ok {
					user = a
					data["user"] = user
				}
			}
		}
		if gate := authorization.From(app); gate != nil {
			data["__can"] = func(ability string, args ...any) bool {
				return gate.Allows(user, ability, args...)
			}
		}
		html, err := engine.Render(resp.ViewName(), data)
		if err != nil {
			if app.IsDebug() {
				resp = http.HTML(fmt.Sprintf("<h1>View Error</h1><pre>%v</pre>", err)).Status(500)
			} else {
				resp = http.Abort(500, "View rendering failed")
			}
		} else {
			resp.SetContent([]byte(html), "text/html; charset=utf-8")
		}
	}

	if bag, ok := req.Session().(*session.Bag); ok && bag != nil {
		if sess := session.From(app); sess != nil {
			_ = sess.Save(bag)
			stdhttp.SetCookie(w, &stdhttp.Cookie{
				Name:     sess.CookieName(),
				Value:    bag.ID(),
				Path:     "/",
				HttpOnly: true,
				SameSite: stdhttp.SameSiteLaxMode,
				MaxAge:   int(time.Hour.Seconds() * 2),
			})
		}
	}
	return resp
}

func (b *httpBridge) sessionMiddleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			sess := session.From(b.app)
			if sess == nil {
				return next(req)
			}
			id := req.Cookie(sess.CookieName())
			bag, err := sess.Start(id)
			if err == nil {
				req.SetSession(bag)
			}
			return next(req)
		}
	}
}

func (b *httpBridge) localeMiddleware() routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			tr := localization.From(b.app)
			if tr != nil {
				langPath := b.app.BasePath("lang")
				locale := ""
				if sess := req.Session(); sess != nil {
					if raw, ok := sess.Get("locale").(string); ok {
						locale = strings.TrimSpace(strings.ToLower(raw))
					}
				}
				if locale == "" {
					locale = negotiateLocale(req.Header("Accept-Language"), localization.Available(langPath))
				}
				if locale != "" && localization.HasLocale(langPath, locale) {
					tr.SetLocale(locale)
					_ = tr.Load(locale)
				}
				if engine := view.From(b.app); engine != nil {
					engine.Share("locale", tr.GetLocale())
				}
			}
			return next(req)
		}
	}
}

func negotiateLocale(header string, available []string) string {
	header = strings.TrimSpace(header)
	if header == "" || len(available) == 0 {
		return ""
	}
	allowed := map[string]string{}
	for _, code := range available {
		code = strings.ToLower(strings.TrimSpace(code))
		allowed[code] = code
		if i := strings.IndexByte(code, '-'); i > 0 {
			allowed[code[:i]] = code
		}
	}
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.Split(part, ";")[0])
		tag = strings.ToLower(tag)
		if tag == "" || tag == "*" {
			continue
		}
		if code, ok := allowed[tag]; ok {
			return code
		}
		if i := strings.IndexByte(tag, '-'); i > 0 {
			if code, ok := allowed[tag[:i]]; ok {
				return code
			}
		}
	}
	return ""
}
