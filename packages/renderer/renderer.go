package renderer

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/zatrano/framework/packages/currentuser"
	"github.com/zatrano/framework/packages/flashmessages"
	"github.com/zatrano/framework/packages/formflash"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

const (
	CsrfTokenKey        = "CsrfToken"
	FlashSuccessKeyView = "Success"
	FlashErrorKeyView   = "Error"
	OldInputKey         = "Old"
	ValidationErrorsKey = "ValidationErrors"
	// LayoutApp tam sayfa kabuğu; HX-Request ile yalnızca gömülü şablon döner.
	LayoutApp = "layouts/app"
	// LayoutWebsite — genel site kabuğu (layouts/website.html).
	LayoutWebsite = "layouts/website"
	// LayoutAuth — oturum açma / kayıt kabuğu.
	LayoutAuth = "layouts/auth"
)

var (
	htmxShellMu      sync.RWMutex
	htmxShellLayouts = make(map[string]struct{})
)

func init() {
	RegisterHtmxShellLayout(LayoutApp)
	RegisterHtmxShellLayout(LayoutWebsite)
	RegisterHtmxShellLayout(LayoutAuth)
}

// RegisterHtmxShellLayout, verilen layout için HX-Request geldiğinde gömülü şablon (kabuk olmadan) dönmeyi açar.
// Örn. yeni "layouts/userpanel" eklendiğinde init veya paket init içinde çağırın.
func RegisterHtmxShellLayout(layout string) {
	layout = strings.TrimSpace(layout)
	if layout == "" {
		return
	}
	htmxShellMu.Lock()
	htmxShellLayouts[layout] = struct{}{}
	htmxShellMu.Unlock()
}

func layoutSupportsHtmxPartial(layout string) bool {
	htmxShellMu.RLock()
	defer htmxShellMu.RUnlock()
	_, ok := htmxShellLayouts[layout]
	return ok
}

// IsHtmxRequest, tarayıcıdan gelen HTMX isteğini (HX-Request: true) tanır.
func IsHtmxRequest(c fiber.Ctx) bool {
	return strings.EqualFold(strings.TrimSpace(c.Get("HX-Request")), "true")
}

func normMenuPath(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 1 {
		s = strings.TrimRight(s, "/")
	}
	return s
}

func pathFromOriginalURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Path != "" {
		return u.Path
	}
	if strings.HasPrefix(raw, "/") {
		if i := strings.IndexAny(raw, "?#"); i >= 0 {
			return raw[:i]
		}
		return raw
	}
	return ""
}

func isDashboardAnaSayfaPath(p string) bool {
	p = normMenuPath(p)
	return p == "/dashboard" || p == "/dashboard/home"
}

// navbarPath, kenar çubuğunda aktif menü eşlemesi için tutarlı yol döner.
// Dashboard grup kökünde c.Path() bazen "/" kalır; URI PathOriginal, FullPath ve OriginalURL ile tamamlanır.
// Ana sayfa rotaları (/dashboard, /dashboard/, /dashboard/home) tek kök olarak /dashboard normalize edilir.
func navbarPath(c fiber.Ctx) string {
	reqPath := string(c.Request().URI().PathOriginal())
	if i := strings.IndexByte(reqPath, '?'); i >= 0 {
		reqPath = reqPath[:i]
	}
	reqPath = normMenuPath(strings.TrimSpace(reqPath))

	pathRaw := strings.TrimSpace(c.Path())
	fp := normMenuPath(c.FullPath())
	orig := normMenuPath(pathFromOriginalURL(c.OriginalURL()))

	p := pathRaw
	if (p == "/" || p == "") && reqPath != "" && strings.HasPrefix(reqPath, "/dashboard") {
		p = reqPath
	}
	if (p == "/" || p == "") && fp != "" && strings.HasPrefix(fp, "/dashboard") {
		p = fp
	}
	if p == "" && fp != "" {
		p = fp
	}
	if (p == "" || p == "/") && strings.HasPrefix(orig, "/dashboard") {
		p = orig
	}
	p = normMenuPath(p)

	if isDashboardAnaSayfaPath(p) || isDashboardAnaSayfaPath(fp) || isDashboardAnaSayfaPath(orig) || isDashboardAnaSayfaPath(reqPath) ||
		(normMenuPath(pathRaw) == "/" && (isDashboardAnaSayfaPath(fp) || isDashboardAnaSayfaPath(orig) || isDashboardAnaSayfaPath(reqPath))) {
		return "/dashboard"
	}
	return p
}

func pageTitleHeader(layout string, data fiber.Map) string {
	t, _ := data["Title"].(string)
	t = strings.TrimSpace(t)
	if layout == LayoutWebsite {
		if t != "" {
			return t + " | ZATRANO"
		}
		return "ZATRANO"
	}
	if t != "" {
		return "ZATRANO | " + t
	}
	return "ZATRANO"
}

// v3: fiber.Ctx artık somut tip (arayüz değil)
func prepareRenderData(c fiber.Ctx, data fiber.Map) fiber.Map {
	renderData := make(fiber.Map)

	// Fiber v3 CSRF: token context'te; eski kod Locals("csrf") bekliyordu.
	tok := csrf.TokenFromContext(c)
	if tok == "" {
		if v, ok := c.Locals("csrf").(string); ok {
			tok = v
		}
	}
	renderData[CsrfTokenKey] = tok

	flashData, _ := flashmessages.GetFlashMessages(c)
	if flashData.Success != "" {
		renderData[FlashSuccessKeyView] = flashData.Success
	}

	formData, err := formflash.GetData(c)
	if err != nil || formData == nil {
		formData = make(map[string]interface{})
	}
	renderData[OldInputKey] = formData
	renderData["FormData"] = formData
	renderData["Data"] = formData

	validationErrors, err := formflash.GetValidationErrors(c)
	if err != nil || validationErrors == nil {
		validationErrors = make(map[string]string)
	}
	renderData[ValidationErrorsKey] = validationErrors
	renderData["Errors"] = validationErrors

	// v3: currentuser artık Locals'tan alınır (SetUserContext kaldırıldı)
	currentUser := currentuser.FromFiber(c)
	if currentUser.ID != 0 {
		renderData["User"] = currentUser
	}

	var handlerError string
	if data == nil {
		data = fiber.Map{}
	}
	if errVal, ok := data[FlashErrorKeyView]; ok {
		if errStr, okStr := errVal.(string); okStr {
			handlerError = errStr
		}
	}

	data["Path"] = navbarPath(c)
	if menuPages := c.Locals("MenuPages"); menuPages != nil {
		data["MenuPages"] = menuPages
	}
	// Yalnızca SharedDataMiddleware (Locals). Handler asla DefinitionValues göndermez.
	if v := c.Locals("DefinitionValues"); v != nil {
		data["DefinitionValues"] = v
	}

	renderData["Upper"] = func(s string) string { return strings.ToUpper(s) }
	renderData["upper"] = func(s string) string { return strings.ToUpper(s) }

	for key, value := range data {
		renderData[key] = value
	}

	var finalError string
	if handlerError != "" {
		finalError = handlerError
	} else if flashData.Error != "" {
		finalError = flashData.Error
	}
	if finalError != "" {
		renderData[FlashErrorKeyView] = finalError
	} else {
		delete(renderData, FlashErrorKeyView)
	}

	if _, ok := renderData["TurnstileSiteKey"]; !ok {
		if v := c.Locals("TurnstileSiteKey"); v != nil {
			renderData["TurnstileSiteKey"] = v
		}
	}

	return renderData
}

func Render(c fiber.Ctx, template string, layout string, data fiber.Map, statusCode ...int) error {
	status := http.StatusOK
	if len(statusCode) > 0 {
		status = statusCode[0]
	}
	finalData := prepareRenderData(c, data)
	layoutToUse := layout
	origLayout := layout
	if layoutSupportsHtmxPartial(origLayout) && IsHtmxRequest(c) {
		layoutToUse = ""
		// Özel HTTP başlığında ham UTF-8 tarayıcıda bozulabiliyor; başlığı yalnızca Base64 taşıyoruz.
		c.Set("X-Page-Title-B64", base64.StdEncoding.EncodeToString([]byte(pageTitleHeader(origLayout, finalData))))
	}
	if layoutToUse == "" {
		return c.Status(status).Render(template, finalData)
	}
	return c.Status(status).Render(template, finalData, layoutToUse)
}
