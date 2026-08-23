package social

import (
	"fmt"
	"strconv"
	"strings"
)

// PersistResult is the outcome of linking a social identity to an app user.
type PersistResult struct {
	UserID  int64
	Created bool
}

// Persistence stores social identities against application users.
// Apps implement this (typically with ORM) or use the make:auth stubs.
//
// Avatar contract: CreateUser/SyncUser must write the provider picture onto the
// authenticatable user (canonical profile photo). UpsertAccount may also store
// a snapshot on the social_accounts row, but applications must not use that
// field for display — only the user avatar.
type Persistence interface {
	FindUserIDByProvider(provider, providerID string) (userID int64, err error)
	FindUserIDByEmail(email string) (userID int64, err error)
	CreateUser(name, email, avatar string, emailVerified bool) (userID int64, err error)
	SyncUser(userID int64, name, avatar string, emailVerified bool) error
	UpsertAccount(userID int64, social *User) error
}

// Persist finds or creates an app user for the social identity, then upserts the provider link.
//
// Security rules:
//   - email_verified_at / CreateUser(..., verified) follows socialUser.EmailVerified only
//     (never forced true).
//   - Linking an existing account by email requires socialUser.EmailVerified == true;
//     otherwise Persist fails closed (account-takeover prevention).
func Persist(store Persistence, socialUser *User) (*PersistResult, error) {
	if store == nil {
		return nil, fmt.Errorf("social: persistence is nil")
	}
	if socialUser == nil {
		return nil, fmt.Errorf("social: user is nil")
	}
	provider := strings.ToLower(strings.TrimSpace(socialUser.Provider))
	providerID := strings.TrimSpace(socialUser.ID)
	email := strings.ToLower(strings.TrimSpace(socialUser.Email))
	if provider == "" || providerID == "" {
		return nil, fmt.Errorf("social: provider and provider id are required")
	}
	if email == "" {
		return nil, fmt.Errorf("social: email is required")
	}

	name := strings.TrimSpace(socialUser.Name)
	if name == "" {
		name = strings.TrimSpace(socialUser.Nickname)
	}
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	avatar := strings.TrimSpace(socialUser.Avatar)
	emailVerified := socialUser.EmailVerified

	if userID, err := store.FindUserIDByProvider(provider, providerID); err != nil {
		return nil, err
	} else if userID > 0 {
		if err := store.SyncUser(userID, name, avatar, emailVerified); err != nil {
			return nil, err
		}
		if err := store.UpsertAccount(userID, socialUser); err != nil {
			return nil, err
		}
		return &PersistResult{UserID: userID, Created: false}, nil
	}

	created := false
	userID, err := store.FindUserIDByEmail(email)
	if err != nil {
		return nil, err
	}
	if userID == 0 {
		userID, err = store.CreateUser(name, email, avatar, emailVerified)
		if err != nil {
			return nil, err
		}
		created = true
	} else {
		if !emailVerified {
			return nil, fmt.Errorf("social: cannot link to existing account: email is not verified by provider")
		}
		if err := store.SyncUser(userID, name, avatar, emailVerified); err != nil {
			return nil, err
		}
	}

	if err := store.UpsertAccount(userID, socialUser); err != nil {
		return nil, err
	}
	return &PersistResult{UserID: userID, Created: created}, nil
}

// parseEmailVerified interprets provider claim values. present=false means the claim was absent.
func parseEmailVerified(v any) (verified bool, present bool) {
	if v == nil {
		return false, false
	}
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		if s == "" || s == "<nil>" {
			return false, false
		}
		if s == "true" || s == "1" {
			return true, true
		}
		if s == "false" || s == "0" {
			return false, true
		}
		return false, true
	case float64:
		return t != 0, true
	case int:
		return t != 0, true
	case jsonNumber:
		n, err := strconv.ParseFloat(string(t), 64)
		if err != nil {
			return false, true
		}
		return n != 0, true
	default:
		s := strings.ToLower(strings.TrimSpace(fmt.Sprint(v)))
		if s == "" || s == "<nil>" {
			return false, false
		}
		if s == "true" || s == "1" {
			return true, true
		}
		if s == "false" || s == "0" {
			return false, true
		}
		return false, true
	}
}

// jsonNumber avoids importing encoding/json in every call site for json.Number.
type jsonNumber string
