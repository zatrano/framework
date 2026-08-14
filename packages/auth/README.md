# Auth

Session authentication, remember-me, password reset, email verification, lockout, TOTP 2FA, and multi-device logout.

## Resolve

```go
auth.From(app)
auth.Passwords(app)
```

## Guards

Config-driven named session guards use per-guard session keys (`login_{guard}_id`):

```go
auth.Middleware(auth.From(app))           // default guard
auth.Middleware(auth.From(app), "web", "api")
auth.VerifyEmailMiddleware(auth.From(app))
```

Personal access tokens live in `packages/apitoken` (middleware), not as an auth guard driver.

## Lockout & 2FA

Lockout counters use the app cache when available (`SetLockoutCache`).

```go
ok, err := auth.From(app).ChallengeTwoFactor(req, code, true) // remember this device
```

Config: `auth.two_factor.issuer`, `auth.two_factor.remember_device_days`, `auth.lockout.*`.

## Scaffold

```bash
zatrano make:auth
```

Wire `RegisterAuthWeb(app)` / `RegisterAuthAPI(app)` from your routes. Scaffold flashes use `auth.*` locale keys (`APP_LOCALE` + `lang/{locale}/auth.json`). Package errors such as `auth.ErrEmailTaken` return those keys from `Error()`.
