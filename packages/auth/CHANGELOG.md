# Changelog

## 1.3.0

- User-facing errors return localization keys (`auth.email_taken`, `auth.lockout`, …)
- `make:auth` controller stub translates flashes via `localization` + `lang/tr|en/auth.json`
- Built-in `localization/defaults/{en,tr}/auth.json` catalogs

## 1.2.0

- Cache-backed lockout via `SetLockoutCache` (boot wires `cache.From(app)`)
- 2FA remember-this-device cookie (`ChallengeTwoFactor(..., true)` / config `remember_device_days`)
- Challenge preserves original login `remember` flag; challenge attempt lockout
- Configurable TOTP issuer (`auth.two_factor.issuer`)

## 1.1.0

- Per-guard session keys (`login_{guard}_id`) with legacy `auth_user_id` read for web
- Guard-aware `Middleware` / `GuestMiddleware` / `VerifyEmailMiddleware`
- Password broker: `ErrUserNotFound` / `ErrResetThrottled`, silent `SendResetLink`, configurable throttle
- `MarkEmailAsVerified` persists and dispatches `auth.verified`
- Config: `passwords`, `lockout`; boot honors expire/throttle/lockout
- Password rehash on successful login when hash cost changes
- Scaffold stubs use `auth.From` / `mail.From` / `social.From`; `make:auth` prefers `packages/console/stubs`

## 1.0.0

- Initial session auth, remember-me, broker, 2FA, lockout, intended URL, confirm password
