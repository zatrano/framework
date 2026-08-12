package social

import (
	"fmt"
	"strings"
)

// PersistResult is the outcome of linking a social identity to an app user.
type PersistResult struct {
	UserID  int64
	Created bool
}

// Persistence stores social identities against application users.
// Apps implement this (typically with ORM) or use the make:auth stubs.
type Persistence interface {
	FindUserIDByProvider(provider, providerID string) (userID int64, err error)
	FindUserIDByEmail(email string) (userID int64, err error)
	CreateUser(name, email, avatar string, emailVerified bool) (userID int64, err error)
	SyncUser(userID int64, name, avatar string, emailVerified bool) error
	UpsertAccount(userID int64, social *User) error
}

// Persist finds or creates an app user for the social identity, then upserts the provider link.
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
	emailVerified := true // trusted providers return verified emails when email is present

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
	} else if err := store.SyncUser(userID, name, avatar, emailVerified); err != nil {
		return nil, err
	}

	if err := store.UpsertAccount(userID, socialUser); err != nil {
		return nil, err
	}
	return &PersistResult{UserID: userID, Created: created}, nil
}
