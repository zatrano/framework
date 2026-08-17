# Security Policy

## Supported Versions

| Version | Supported |
| ------- | --------- |
| 1.2.x   | ✅ |
| 1.1.x   | ✅ (security fixes backported when practical) |
| < 1.1   | ❌ |

## Reporting a Vulnerability

Email **Serhan KARAKOÇ** at [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com).

Do **not** open a public GitHub issue for security vulnerabilities.

Please include:

* ZATRANO version / commit
* Affected package and API
* Reproduction steps (PoC)
* Impact assessment

We aim to acknowledge reports within a few business days and ship fixes promptly.

## Security philosophy

* Prefer **secure defaults** that still allow local HTTP development
* User-controlled values must not become SQL identifiers, Mongo operators, or filesystem paths
* Sensitive cookies: **HttpOnly** + **SameSite=Lax**; **Secure** when HTTPS or `SESSION_SECURE`/`COOKIE_SECURE=true`
* CSRF on browser form posts by default; JSON APIs may opt out with `csrf.Except` explicitly
* WebSocket upgrades use **same-origin** Origin checks by default
* Cryptographic tokens use `crypto/rand`; password hashing uses **bcrypt**
* Production: `APP_DEBUG=false`, strong `APP_KEY`, reverse proxy TLS, explicit CORS origins, never `TRUSTED_PROXIES=*` without `TRUST_PROXIES_ALLOW_STAR=true`

## Secure configuration (production checklist)

```env
APP_ENV=production
APP_DEBUG=false
APP_KEY=<32+ byte random>
APP_URL=https://example.com
SESSION_SECURE=true
CORS_ALLOWED_ORIGINS=https://example.com
CORS_ALLOW_CREDENTIALS=true
TRUSTED_PROXIES=10.0.0.0/8   # never * in production without TRUST_PROXIES_ALLOW_STAR=true
COOKIE_SECURE=true
MAX_UPLOAD_BYTES=33554432
```

## Recommendations

| Area | Guidance |
|------|----------|
| Authentication | Enable lockout; use HTTPS; rotate session after login |
| Sessions | File permissions `0600`; consider Redis for multi-node |
| CSRF | Keep CSRF on HTML forms; do not blanket-except browser cookie routes |
| CORS | Explicit origins; never `*` with credentials |
| WebSocket | Default same-origin; use `AllowAnyOrigin` only in controlled environments |
| Database | Never pass request input to `WhereRaw` / `OrderByRaw` / table names |
| Uploads | Validate MIME/extensions in the app; framework blocks path escape |
| Secrets | Env or secret manager; never commit `.env` |
| Reverse proxy | Terminate TLS; set trusted proxies narrowly; pass `X-Forwarded-Proto` |

## Security testing

See [docs/security/testing.md](docs/security/testing.md) and `SECURITY_AUDIT.md` / `SECURITY_REPORT.md`.
