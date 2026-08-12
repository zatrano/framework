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
		return fmt.Errorf("unauthenticated")
	}
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" || email == "" {
		return fmt.Errorf("name and email are required")
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
			return fmt.Errorf("email already taken")
		}
	}

	updater, ok := m.Guard().Provider().(AttributeUpdater)
	if !ok {
		return fmt.Errorf("user provider does not support profile updates")
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
