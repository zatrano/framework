package auth

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

// UpdateProfile updates the authenticated user's name/email.
// Changing email clears email_verified_at so verification can run again.
func (m *Manager) UpdateProfile(req *http.Request, name, email string) error {
	user := m.User(req)
	if user == nil {
		return ErrUnauthenticated
	}
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || email == "" {
		return ErrNameEmailRequired
	}

	currentEmail := EmailForVerification(user)
	attrs := map[string]any{"name": name, "email": email}
	if !strings.EqualFold(currentEmail, email) {
		attrs["email_verified_at"] = nil
		existing, err := m.Guard().Provider().RetrieveByCredentials(map[string]string{"email": email})
		if err != nil {
			return err
		}
		if existing != nil && fmt.Sprint(existing.AuthID()) != fmt.Sprint(user.AuthID()) {
			return ErrEmailTaken
		}
	}

	updater, ok := m.Guard().Provider().(AttributeUpdater)
	if !ok {
		return ErrProviderNoProfile
	}
	if err := updater.UpdateAttributes(user.AuthID(), attrs); err != nil {
		return err
	}
	if generic, ok := user.(*GenericUser); ok && generic.Attributes != nil {
		for k, v := range attrs {
			generic.Attributes[k] = v
		}
	}
	return nil
}
