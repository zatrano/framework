package providers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/auth"
)

// AuthServiceProvider registers authorization gates and policies.
type AuthServiceProvider struct{}

func (p *AuthServiceProvider) Register(app *core.Application) error {
	return nil
}

func (p *AuthServiceProvider) Boot(app *core.Application) error {
	if mgr := auth.From(app); mgr != nil {
		if g := mgr.Guard(); g != nil {
			if provider, ok := g.Provider().(*auth.DatabaseUserProvider); ok {
				provider.WithHydrate(hydrateAppUser)
			}
		}
	}
	return nil
}

func hydrateAppUser(row map[string]any) auth.Authenticatable {
	if row == nil {
		return nil
	}
	u := &models.User{
		Name:     fmt.Sprint(nullStr(row["name"])),
		Email:    fmt.Sprint(nullStr(row["email"])),
		Password: fmt.Sprint(nullStr(row["password"])),
		Avatar:   fmt.Sprint(nullStr(row["avatar"])),
		IsAdmin:  toBool(row["is_admin"]),
	}
	u.ID = toInt64(row["id"])
	if v := row["remember_token"]; v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
		s := fmt.Sprint(v)
		u.RememberToken = &s
	}
	if t := toTimePtr(row["email_verified_at"]); t != nil {
		u.EmailVerifiedAt = t
	}
	if v := row["two_factor_secret"]; v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
		s := fmt.Sprint(v)
		u.TwoFactorSecret = &s
	}
	if v := row["two_factor_recovery_codes"]; v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
		s := fmt.Sprint(v)
		u.TwoFactorRecoveryCodes = &s
	}
	if t := toTimePtr(row["two_factor_confirmed_at"]); t != nil {
		u.TwoFactorConfirmedAt = t
	}
	if v := row["stripe_customer_id"]; v != nil && fmt.Sprint(v) != "" && fmt.Sprint(v) != "<nil>" {
		s := fmt.Sprint(v)
		u.StripeCustomerID = &s
	}
	return u
}

func nullStr(v any) string {
	if v == nil {
		return ""
	}
	s := fmt.Sprint(v)
	if s == "<nil>" {
		return ""
	}
	return s
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case string:
		parsed, _ := strconv.ParseInt(n, 10, 64)
		return parsed
	default:
		parsed, _ := strconv.ParseInt(fmt.Sprint(v), 10, 64)
		return parsed
	}
}

func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case int64:
		return b != 0
	case int:
		return b != 0
	case float64:
		return b != 0
	case string:
		s := strings.ToLower(strings.TrimSpace(b))
		return s == "1" || s == "true" || s == "t" || s == "yes"
	default:
		s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		return s == "1" || s == "true" || s == "t" || s == "yes"
	}
}

func toTimePtr(v any) *time.Time {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case time.Time:
		if t.IsZero() {
			return nil
		}
		return &t
	case *time.Time:
		return t
	case string:
		s := strings.TrimSpace(t)
		if s == "" || s == "<nil>" {
			return nil
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"} {
			if parsed, err := time.Parse(layout, s); err == nil {
				return &parsed
			}
		}
	}
	return nil
}
