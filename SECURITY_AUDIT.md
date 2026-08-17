# ZATRANO Security Audit — Phase 0

**Module:** `github.com/zatrano/framework`  
**Go:** 1.25.0 · **Version:** 1.2.6  
**Date:** 2026-08-17  
**Scope:** inventory only (no code changes in this phase)

---

## Architecture overview

```text
cmd/zatrano → bootstrap (profiles) → core.Application (ServeHTTP)
  ├─ Kernel middleware: trustedproxy → recover → requestID → security headers → …
  ├─ packages/routing Router
  ├─ packages/http Request/Response
  ├─ Foundation: session, auth, CSRF, CORS, views, DB/ORM
  └─ Optional addons: social, mongo, websocket, process, …
```

Thin kernel (`core/`) + foundation (`bootstrap/foundation`) + opt-in packages. Default HTTP stack uses `net/http` with a custom router (not Fiber). Persistence uses a first-party query builder/ORM (not GORM).

---

## Security-sensitive components

| Area | Location | Notes |
|------|----------|--------|
| HTTP / server | `core/application.go`, `packages/http` | ReadHeaderTimeout only |
| Router | `packages/routing` | Path matching, method override |
| Auth | `packages/auth`, `packages/hashing` | bcrypt; lockout; TOTP; remember-me |
| Authorization | `packages/authorization` | Gates/policies (opt-in) |
| Session / cookies | `packages/session`, `packages/cookie` | File sessions; HttpOnly+Lax |
| CSRF | `packages/middleware/csrf` | Session token; `/api` often excepted |
| CORS | `packages/middleware/cors` | Env-driven; default `*` |
| Trusted proxy | `packages/trustedproxy` | X-Forwarded-* |
| Validation | `packages/validation` | PresenceChecker for unique/exists |
| Views | `packages/view` | `html/template` + `safe` raw HTML |
| ORM / SQL | `packages/orm`, `packages/database/query` | Bindings + Raw APIs |
| Drivers | `packages/database/driver/*` | sqlite/mysql/pgsql/mssql/oracle/mongo |
| Mongo | `packages/mongo` | Document API |
| Crypto | `packages/encryption`, `packages/hashing` | AES-GCM, bcrypt |
| Process | `packages/process`, `packages/console` | `os/exec` |
| Upload / FS | `packages/http/upload.go`, `packages/filesystem` | Multipart + local disks |
| WebSocket | `packages/websocket` | Hijack upgrade |

---

## Attack surface

1. **Public HTTP** — routing, static `public/`, forms, cookies, headers  
2. **Auth flows** — login, reset, verify, MFA, remember-me, social OAuth stub/live  
3. **API surfaces** — CSRF-excepted `/api`, JSON validation  
4. **Persistence** — parameterized queries vs Raw/table interpolation  
5. **Templates** — escaped output vs intentional `safe`  
6. **CLI / process** — `db:setup`, deploy helpers (`os/exec`)  
7. **Proxy trust** — Host / client IP / scheme spoofing  
8. **Optional addons** — websocket, mongo filters, HTTP client / webhooks  

---

## Initial threat model

| Threat | Assets | Likely entry |
|--------|--------|----------------|
| Account takeover | Sessions, reset tokens, remember cookies | Auth/session/CSRF |
| SQLi | DB data integrity/confidentiality | Raw SQL, identifiers, OrderBy |
| XSS | Session cookies (if not HttpOnly paths), stored HTML | `safe`, attributes |
| CSRF | State-changing web routes | Missing/excluded CSRF |
| Privilege escalation | Gates/policies | Missing Authorize |
| SSRF | Server-side HTTP clients / webhooks | User-supplied URLs |
| RCE | Host | `process` + untrusted input |
| DoS | Availability | Body size, regex, unbounded queries |
| Info disclosure | Secrets, schema, stack | APP_DEBUG, Recover bodies |

Assumptions: production should set `APP_DEBUG=false`, strong `APP_KEY`, HTTPS reverse proxy, and enable CSRF on browser-facing routes.

---

## High-risk areas (Phase 1+ backlog)

1. Session cookie / remember-me / XSRF: **Secure** flag policy  
2. Session ID filesystem path sanitization  
3. Password-reset / email-verify token **SHA-1** storage hash → SHA-256  
4. CORS default `*` + credentials combinations  
5. Query `*Raw` and dynamic table/column names  
6. View `safe` / `safeStr` documentation and XSS tests  
7. `os/exec` call sites (`process`, console)  
8. Recover / debug error leakage  
9. WebSocket Origin validation  
10. Upload extension/MIME controls  
11. Trusted proxies misconfiguration (`*`)  
12. Server Read/Write timeouts  

---

## Testing strategy

| Layer | Approach |
|-------|----------|
| Unit | Auth tokens, CSRF, cookie flags, query binding, escape helpers |
| Integration | Router + middleware stack; session login rotation |
| Negative | Injection payloads (SQL, path, XSS, CORS, Host) |
| Race | `go test -race` on session/auth/rate-limit |
| Static | `go vet`, `staticcheck`, `gosec`, `govulncheck` |
| Fuzz | Router paths, query identifiers, cookie parse (bounded CI) |
| Dynamic | Local demo app + OWASP ZAP baseline (non-prod only) |

**Rules:** confirmed issues need reproducers + regression tests; no Fiber/GORM; minimal API breaks; no invented vulns.

---

## Existing CI (pre-audit)

- `.github/workflows/tests.yml`
- `.github/workflows/static-analysis.yml`
- `.github/workflows/coding-style.yml`

Security-specific workflow (`security.yml`) to be added in later phases.
