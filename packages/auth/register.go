package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/hashing"
	"github.com/zatrano/framework/packages/http"
)

// UserCreator creates authenticatable users.
type UserCreator interface {
	Create(attrs map[string]any) (Authenticatable, error)
}

// AttributeUpdater updates user attributes by id.
type AttributeUpdater interface {
	UpdateAttributes(id any, attrs map[string]any) error
}

// PasswordUpdater updates a password by email.
type PasswordUpdater interface {
	UpdatePassword(email, hashedPassword string) error
}

// Register creates a user and optionally logs them in.
func (m *Manager) Register(req *http.Request, attrs map[string]any, login ...bool) (Authenticatable, error) {
	if m == nil || m.Guard() == nil {
		return nil, ErrGuardUnavailable
	}
	provider := m.Guard().Provider()
	creator, ok := provider.(UserCreator)
	if !ok || creator == nil {
		return nil, ErrProviderNoRegister
	}

	email := strings.TrimSpace(fmt.Sprint(attrs["email"]))
	if email == "" {
		return nil, ErrEmailRequired
	}
	existing, err := provider.RetrieveByCredentials(map[string]string{"email": email})
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrEmailTaken
	}

	password := strings.TrimSpace(fmt.Sprint(attrs["password"]))
	if password == "" {
		return nil, ErrPasswordRequired
	}
	hashed, err := hashing.Hash(password)
	if err != nil {
		return nil, err
	}
	payload := make(map[string]any, len(attrs))
	for k, v := range attrs {
		payload[k] = v
	}
	payload["password"] = hashed
	delete(payload, "password_confirmation")

	user, err := creator.Create(payload)
	if err != nil {
		return nil, err
	}
	doLogin := true
	if len(login) > 0 {
		doLogin = login[0]
	}
	if doLogin && req != nil {
		if err := m.Login(req, user); err != nil {
			return user, err
		}
	}
	m.dispatch(EventRegistered, RegisteredEvent{Request: req, User: user, Guard: m.Guard().name, At: time.Now().UTC()})
	return user, nil
}

// ChangePassword updates the authenticated user's password.
func (m *Manager) ChangePassword(req *http.Request, current, next string) error {
	user := m.User(req)
	if user == nil {
		return ErrUnauthenticated
	}
	if !hashing.Check(current, user.AuthPassword()) {
		return ErrCurrentPassword
	}
	if strings.TrimSpace(next) == "" {
		return ErrNewPasswordRequired
	}
	hashed, err := hashing.Hash(next)
	if err != nil {
		return err
	}

	provider := m.Guard().Provider()
	updated := false
	if updater, ok := provider.(PasswordUpdater); ok {
		email := EmailForVerification(user)
		if email == "" {
			if generic, ok := user.(*GenericUser); ok {
				email = fmt.Sprint(generic.Get("email"))
			}
		}
		if email != "" {
			if err := updater.UpdatePassword(email, hashed); err != nil {
				return err
			}
			updated = true
		}
	}
	if !updated {
		if attr, ok := provider.(AttributeUpdater); ok {
			if err := attr.UpdateAttributes(user.AuthID(), map[string]any{"password": hashed}); err != nil {
				return err
			}
			updated = true
		}
	}
	if !updated {
		return ErrProviderNoPassword
	}

	m.Guard().clearRememberCookie(req, user)
	ClearPasswordConfirmation(req)
	if m.sessions != nil && req != nil && req.Session() != nil {
		_, _ = m.sessions.DestroyOthersForUser(user.AuthID(), req.Session().ID())
	}
	if generic, ok := user.(*GenericUser); ok && generic.Attributes != nil {
		generic.Attributes["password"] = hashed
	}
	m.dispatch(EventPasswordReset, PasswordResetEvent{Request: req, User: user, Guard: m.Guard().name, At: time.Now().UTC()})
	return nil
}
