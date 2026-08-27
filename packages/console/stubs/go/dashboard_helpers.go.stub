package web

import (
	"fmt"
	"strconv"

	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/auth"
	. "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/localization"
)

func dashboardLang(app *core.Application, key string, replace ...map[string]string) string {
	if tr := localization.From(app); tr != nil {
		return tr.Get(key, replace...)
	}
	return key
}

func dashboardCurrentUser(req *Request, app *core.Application) *models.User {
	raw := auth.From(app).User(req)
	if raw == nil {
		return nil
	}
	if u, ok := raw.(*models.User); ok {
		return u
	}
	return nil
}

func dashboardUserIDString(u *models.User) string {
	if u == nil {
		return ""
	}
	return fmt.Sprintf("%d", u.ID)
}

func dashboardParseID(req *Request, key string) (int64, error) {
	raw := req.Route(key)
	if raw == "" {
		raw = req.Input(key)
	}
	return strconv.ParseInt(raw, 10, 64)
}
