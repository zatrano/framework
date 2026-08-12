package auth

import (
	"time"

	"github.com/zatrano/framework/packages/http"
)

const (
	EventAttempting             = "auth.attempting"
	EventFailed                 = "auth.failed"
	EventLogin                  = "auth.login"
	EventLogout                 = "auth.logout"
	EventRegistered             = "auth.registered"
	EventVerified               = "auth.verified"
	EventPasswordReset          = "auth.password_reset"
	EventCurrentDeviceLogout    = "auth.current_device_logout"
	EventOtherDeviceLogout      = "auth.other_device_logout"
	EventLockout                = "auth.lockout"
	EventTwoFactorChallenged    = "auth.two_factor_challenged"
	EventTwoFactorAuthenticated = "auth.two_factor_authenticated"
)

// Dispatcher receives optional authentication lifecycle events.
type Dispatcher interface {
	Dispatch(name string, event any) error
}

// AuthEvent is the common payload for authentication lifecycle events.
type AuthEvent struct {
	Request     *http.Request
	User        Authenticatable
	Credentials map[string]string
	Guard       string
	At          time.Time
}

type AttemptingEvent = AuthEvent
type FailedEvent = AuthEvent
type LoginEvent = AuthEvent
type LogoutEvent = AuthEvent
type RegisteredEvent = AuthEvent
type VerifiedEvent = AuthEvent
type PasswordResetEvent = AuthEvent
type CurrentDeviceLogoutEvent = AuthEvent
type OtherDeviceLogoutEvent = AuthEvent
type LockoutEvent = AuthEvent
type TwoFactorChallengedEvent = AuthEvent
type TwoFactorAuthenticatedEvent = AuthEvent
