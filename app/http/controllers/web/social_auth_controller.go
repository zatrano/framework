package web

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/flash"
	"github.com/zatrano/framework/packages/hashing"
	. "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/localization"
	"github.com/zatrano/framework/packages/orm"
	"github.com/zatrano/framework/packages/social"
)

const (
	sessionOAuthStatePrefix = "oauth_state_"
	pathLogin               = "/auth/login"
)

// SocialAuthController handles Google/GitHub OAuth login/register.
type SocialAuthController struct {
	App *core.Application
}

func (c *SocialAuthController) lang(key string, replace ...map[string]string) string {
	if tr := localization.From(c.App); tr != nil {
		return tr.Get(key, replace...)
	}
	return key
}

func (c *SocialAuthController) providerLabel(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "google":
		return c.lang("auth.provider_google")
	case "github":
		return c.lang("auth.provider_github")
	}
	if p == "" {
		return provider
	}
	r, size := utf8.DecodeRuneInString(p)
	if r == utf8.RuneError {
		return provider
	}
	return string(unicode.ToUpper(r)) + p[size:]
}

func (c *SocialAuthController) providerReplace(provider string) map[string]string {
	return map[string]string{"provider": c.providerLabel(provider)}
}

// GoogleRedirect starts the Google OAuth flow.
func (c *SocialAuthController) GoogleRedirect(req *Request) *Response {
	return c.redirect(req, "google")
}

// GoogleCallback completes Google OAuth and signs the user in.
func (c *SocialAuthController) GoogleCallback(req *Request) *Response {
	return c.callback(req, "google")
}

// GitHubRedirect starts the GitHub OAuth flow.
func (c *SocialAuthController) GitHubRedirect(req *Request) *Response {
	return c.redirect(req, "github")
}

// GitHubCallback completes GitHub OAuth and signs the user in.
func (c *SocialAuthController) GitHubCallback(req *Request) *Response {
	return c.callback(req, "github")
}

func (c *SocialAuthController) redirect(req *Request, provider string) *Response {
	repl := c.providerReplace(provider)
	if c.App == nil || social.From(c.App) == nil {
		return flash.WithError(req, c.lang("auth.social_not_configured", repl), pathLogin)
	}
	url, state, err := social.From(c.App).Redirect(provider)
	if err != nil {
		return flash.WithError(req, c.lang("auth.social_start_failed", repl), pathLogin)
	}
	if sess := req.Session(); sess != nil {
		sess.Put(sessionOAuthStatePrefix+provider, state)
	}
	return Redirect(url)
}

func (c *SocialAuthController) callback(req *Request, provider string) *Response {
	repl := c.providerReplace(provider)
	if errMsg := strings.TrimSpace(req.Query("error")); errMsg != "" {
		desc := strings.TrimSpace(req.Query("error_description"))
		if desc == "" {
			desc = c.lang("auth.social_cancelled", repl)
		}
		return flash.WithError(req, desc, pathLogin)
	}

	state := strings.TrimSpace(req.Query("state"))
	code := strings.TrimSpace(req.Query("code"))
	if state == "" || code == "" {
		return flash.WithError(req, c.lang("auth.social_invalid_response", repl), pathLogin)
	}

	expected := ""
	if sess := req.Session(); sess != nil {
		if v, ok := sess.Pull(sessionOAuthStatePrefix + provider).(string); ok {
			expected = strings.TrimSpace(v)
		}
	}
	if expected == "" || expected != state {
		return flash.WithError(req, c.lang("auth.social_state_failed", repl), pathLogin)
	}
	_ = social.From(c.App).ValidateState(state)

	socialUser, err := social.From(c.App).User(provider, code)
	if err != nil || socialUser == nil || strings.TrimSpace(socialUser.Email) == "" {
		return flash.WithError(req, c.lang("auth.social_load_failed", repl), pathLogin)
	}

	result, err := social.Persist(&ormSocialPersistence{}, socialUser)
	if err != nil || result == nil || result.UserID == 0 {
		return flash.WithError(req, c.lang("auth.social_link_failed"), pathLogin)
	}

	user, err := orm.Find[models.User](result.UserID)
	if err != nil || user == nil {
		return flash.WithError(req, c.lang("auth.social_user_load_failed"), pathLogin)
	}

	if err := auth.From(c.App).Login(req, user); err != nil {
		return flash.WithError(req, c.lang("auth.social_login_failed"), pathLogin)
	}

	msg := c.lang("auth.social_signed_in", repl)
	if result.Created {
		msg = c.lang("auth.social_account_created", repl)
	}
	return flash.WithSuccess(req, msg, auth.PullIntendedURL(req, "/"))
}

type ormSocialPersistence struct{}

func (p *ormSocialPersistence) FindUserIDByProvider(provider, providerID string) (int64, error) {
	uid := models.ProviderUIDFor(strings.ToLower(provider), providerID)
	rows, err := orm.Query[models.SocialAccount]().Where("provider_uid", uid).Limit(1).Get()
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].UserID, nil
}

func (p *ormSocialPersistence) FindUserIDByEmail(email string) (int64, error) {
	rows, err := orm.Query[models.User]().Where("email", email).Limit(1).Get()
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return rows[0].ID, nil
}

func (p *ormSocialPersistence) CreateUser(name, email, avatar string, emailVerified bool) (int64, error) {
	plain, err := randomOAuthPassword(24)
	if err != nil {
		return 0, err
	}
	hashed, err := hashing.Hash(plain)
	if err != nil {
		return 0, err
	}
	// Canonical profile photo lives on User — never read SocialAccount.Avatar for UI.
	user := &models.User{
		Name:     name,
		Email:    email,
		Password: hashed,
		Avatar:   avatar,
	}
	if emailVerified {
		now := time.Now().UTC()
		user.EmailVerifiedAt = &now
	}
	if err := orm.Save(user); err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (p *ormSocialPersistence) SyncUser(userID int64, name, avatar string, emailVerified bool) error {
	user, err := orm.Find[models.User](userID)
	if err != nil || user == nil {
		return fmt.Errorf("user %d not found", userID)
	}
	changed := false
	if name != "" && strings.TrimSpace(user.Name) == "" {
		user.Name = name
		changed = true
	}
	// Copy provider avatar onto the auth user (system of record for display).
	if avatar != "" && user.Avatar != avatar {
		user.Avatar = avatar
		changed = true
	}
	if emailVerified && !user.HasVerifiedEmail() {
		user.MarkEmailAsVerified()
		changed = true
	}
	if !changed {
		return nil
	}
	return orm.Save(user)
}

func (p *ormSocialPersistence) UpsertAccount(userID int64, socialUser *social.User) error {
	provider := strings.ToLower(strings.TrimSpace(socialUser.Provider))
	providerID := strings.TrimSpace(socialUser.ID)
	uid := models.ProviderUIDFor(provider, providerID)

	name := strings.TrimSpace(socialUser.Name)
	email := strings.ToLower(strings.TrimSpace(socialUser.Email))
	avatar := strings.TrimSpace(socialUser.Avatar)
	token := strings.TrimSpace(socialUser.Token)

	rows, err := orm.Query[models.SocialAccount]().Where("provider_uid", uid).Limit(1).Get()
	if err != nil {
		return err
	}

	var account *models.SocialAccount
	if len(rows) > 0 {
		account = &rows[0]
	} else {
		account = &models.SocialAccount{
			UserID:      userID,
			Provider:    provider,
			ProviderID:  providerID,
			ProviderUID: uid,
		}
	}
	account.UserID = userID
	account.Provider = provider
	account.ProviderID = providerID
	account.ProviderUID = uid
	if name != "" {
		account.Name = &name
	}
	if email != "" {
		account.Email = &email
	}
	// Keep a provider snapshot on the link row; display still uses User.Avatar only.
	if avatar != "" {
		account.Avatar = &avatar
	}
	if token != "" {
		account.AccessToken = &token
	}
	return orm.Save(account)
}

func randomOAuthPassword(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
