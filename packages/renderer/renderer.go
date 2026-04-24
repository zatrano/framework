package renderer

import (
	"encoding/base64"
	"net/http"
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

	data["Path"] = strings.TrimSpace(c.Path())
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
