package auth

// Passwords resolves the password broker from the application container.
func Passwords(app App) *PasswordBroker {
	if app == nil {
		return nil
	}
	raw, err := app.Make("passwords")
	if err != nil {
		return nil
	}
	v, _ := raw.(*PasswordBroker)
	return v
}
