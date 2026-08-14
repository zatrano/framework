package auth

import "errors"

// User-facing errors use localization keys as Error() text (e.g. auth.email_taken).
// Controllers should pass err.Error() through localization.Translator.Get.
var (
	ErrEmailTaken          = errors.New("auth.email_taken")
	ErrEmailRequired       = errors.New("auth.email_required")
	ErrPasswordRequired    = errors.New("auth.password_required")
	ErrNewPasswordRequired = errors.New("auth.new_password_required")
	ErrNameEmailRequired   = errors.New("auth.name_email_required")
	ErrUnauthenticated     = errors.New("auth.unauthenticated")
	ErrCurrentPassword     = errors.New("auth.password")
	ErrResetTokenInvalid   = errors.New("auth.reset_token_invalid")
	ErrGuardUnavailable    = errors.New("auth.guard_unavailable")
	ErrProviderNoRegister  = errors.New("auth.provider_no_register")
	ErrProviderNoPassword  = errors.New("auth.provider_no_password")
	ErrProviderNoProfile   = errors.New("auth.provider_no_profile")
)

// MessageKey returns err.Error() for catalog lookup (nil-safe).
func MessageKey(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
