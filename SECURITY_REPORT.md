# ZATRANO Security Report

**Date:** 2026-08-17  
**Framework version:** 1.2.7 (this audit release)  
**Scope:** Phases 0–25 (focused remediation of confirmed issues; not an exhaustive pentest of every addon)

## Executive Summary

ZATRANO’s core stack (auth bcrypt, CSRF, parameterized WHERE values, `html/template` escaping) is generally sound. This audit confirmed and fixed several **real** weaknesses around SQL identifier injection, session filesystem path handling, password-reset token hashing, Mongo equality-filter operator injection, and HTTP server timeouts / session cookie Secure policy.

**Do not interpret this report as “100% secure.”** Optional addons (WebSocket Origin, `process`/`os/exec`, Raw SQL APIs, view `safe`) remain trusted-developer surfaces.

### Counts

| Severity | Confirmed | Fixed in this pass |
|----------|-----------|--------------------|
| Critical | 2 | 2 |
| High | 4 | 4 |
| Medium | 5 | 5 |
| Low | 3 | 2 (+ verified CORS) |
| Informational | several | documented |

Regression tests added for SQLi identifiers, session path traversal, Mongo operators, CORS credentials+wildcard, filesystem/static path escape, upload StoreAs, WebSocket Origin, trusted proxy star.

---

## Findings

### SEC-001 — Session ID path traversal (Critical)

* **Component:** `packages/session`
* **Description:** Cookie session IDs were joined into filesystem paths without validation (`filepath.Join(dir, id)`), allowing `../` style IDs.
* **Attack:** Attacker sets session cookie to `../other-file` to read/write outside the session directory (depending on OS permissions).
* **Fix:** Accept only 32-char hex IDs; `underDir` check on Start/Save/Destroy.
* **Regression:** `TestSessionPathTraversalRejected`
* **Status:** Fixed

### SEC-002 — SQL injection via OrderBy / identifiers (High)

* **Component:** `packages/database/query`
* **Description:** `OrderBy`, `GroupBy`, `Having`, `Where` (column/operator), `Join`, `WhereIn` interpolated untrusted identifiers/operators.
* **Attack:** `OrderBy("id; DROP TABLE…")` or hostile operators in `Where`.
* **Fix:** `sanitizeIdentifier` / `sanitizeOperator` / direction whitelist; builder `err` surfaced on `Get`.
* **Regression:** `TestSQLInjectionOrderByRejected`, `TestSQLInjectionWhereOperatorRejected`, `FuzzSanitizeIdentifier`
* **Status:** Fixed  
* **Note:** `WhereRaw` / `OrderByRaw` / `SelectRaw` remain trusted APIs by design.

### SEC-003 — Mongo operator injection (High)

* **Component:** `packages/mongo`
* **Description:** Filters with `$ne` / `$where` could alter query semantics on the real driver; hostile filters could become empty ⇒ match-all.
* **Fix:** `sanitizeEqualityFilter` rejects `$` keys; hostile filters return no documents.
* **Regression:** `TestMongoOperatorInjectionRejected`
* **Status:** Fixed

### SEC-004 — Password reset token stored as SHA-1 (Medium)

* **Component:** `packages/auth` `hashToken`
* **Description:** High-entropy tokens hashed with SHA-1 at rest.
* **Fix:** SHA-256 (invalidates outstanding reset tokens — acceptable).
* **Status:** Fixed  
* **Also:** `EmailHash` for verification links moved to SHA-256 (existing links invalidate).

### SEC-005 — Session cookie Secure flag (Medium)

* **Component:** `bootstrap/foundation/http_bridge.go`
* **Description:** Session cookie lacked `Secure`.
* **Fix:** `Secure: req.Secure() || SESSION_SECURE`
* **Status:** Fixed (localhost HTTP still works unless `SESSION_SECURE=true`)

### SEC-006 — Missing Read/Write timeouts (Medium)

* **Component:** `core/application.go`
* **Fix:** `ReadTimeout`/`WriteTimeout` 60s, `IdleTimeout` 120s
* **Status:** Fixed

### SEC-007 — CORS `*` + credentials (Low / already mitigated)

* **Component:** `packages/middleware/cors.go`
* **Description:** Credentials header already suppressed when origin is `*`.
* **Regression:** `TestCORSCredentialsNotWithWildcard`
* **Status:** Verified safe; test added

### SEC-008 — View `safe` / Raw SQL / process (Informational)

Documented residual risks for trusted developers. Not removed (would break DX).

### SEC-009 — Static / LocalDisk path traversal (Critical)

* **Component:** `core/application.go`, `packages/filesystem`, `packages/safepath`
* **Fix:** `safepath.Resolve` / `Under` on static `public/` and all LocalDisk I/O
* **Regression:** `TestPathTraversalRejected`, `TestResolveRejectsTraversal`
* **Status:** Fixed

### SEC-010 — Upload StoreAs traversal + client max raise (High)

* **Component:** `packages/http/upload.go`
* **Fix:** basename + Resolve; `X-Max-Upload` can only lower cap
* **Regression:** `TestStoreAsPathTraversalRejected`
* **Status:** Fixed

### SEC-011 — WebSocket missing Origin check (High)

* **Component:** `packages/websocket`
* **Fix:** default `SameOrigin`; `UpgradeWithCheckOrigin` / `AllowAnyOrigin`
* **Regression:** `TestSameOriginRejectsCrossSite`
* **Status:** Fixed

### SEC-012 — TRUSTED_PROXIES=* in production (Medium)

* **Component:** `packages/trustedproxy`
* **Fix:** ignore `*` unless `TRUST_PROXIES_ALLOW_STAR=true` or non-production `APP_ENV`
* **Regression:** `TestFromEnvIgnoresStarInProduction`
* **Status:** Fixed

### SEC-013 — CSRF default /api except (Medium)

* **Component:** `packages/middleware/csrf`
* **Fix:** `Middleware` protects all paths; scaffold keeps explicit `Except("/api")`
* **Status:** Fixed (secure default)

### SEC-014 — Cookie Secure defaults (Low)

* **Component:** cookie jar, XSRF cookie
* **Fix:** `COOKIE_SECURE`/`SESSION_SECURE`; XSRF `Secure` when request is HTTPS
* **Status:** Fixed

---

## Tool Results

| Tool | Status |
|------|--------|
| `go vet` | Covered in CI (`static-analysis.yml` + `security.yml`) |
| `staticcheck` | Wired in `security.yml` |
| `govulncheck` | Wired in `security.yml` |
| `gosec` | Wired in `security.yml` |
| `go test -race` | Session/auth/ratelimit/middleware in `security.yml` |
| Fuzz | `FuzzSanitizeIdentifier` smoke 15s in CI |
| OWASP ZAP | Manual via `tests/securitydemo` + docs |

Full scanner output should be captured from CI runs after merge; local gosec install may require Go ≥ toolchain requested by the gosec module.

---

## Security Coverage

| Area | Covered |
|------|---------|
| Router | Partial (existing tests; fuzz identifier focus) |
| HTTP timeouts | Yes |
| Auth hashing / reset hash | Yes |
| Authorization / IDOR | Partial (framework provides gates; app-level) |
| Sessions | Path + cookie Secure |
| CSRF | Existing + demo |
| ORM / SQL identifiers | Yes |
| Mongo | Operator filter |
| Templates | Escape default; `safe` documented |
| CORS | Credentials+wildcard |
| File upload | Path + size cap |
| Static files | safepath |
| WebSocket Origin | SameOrigin default |
| Trusted proxies | production `*` ignored |
| Config / crypto | Inventory + token hash |
| Concurrency | Race job in CI |

---

## Remaining Risks

1. `WhereRaw` / `OrderByRaw` / dynamic table names — developer responsibility (trusted SQL APIs)
2. View `safe` / `{!! !!}` raw HTML — intentional XSS surface when misused
3. `packages/process` / console `os/exec` — never feed untrusted input (documented)
4. Upload MIME/extension policy still app-level (path traversal fixed)
5. CSRF often excepted for `/api` in the app scaffold — use Bearer/tokens for cookie-auth APIs
6. Full ZAP/active scan not automated in CI (demo documented)
7. Explicit `AllowAnyOrigin` / `TRUST_PROXIES_ALLOW_STAR=true` re-open risks by design

---

## Reproduce

```bash
go test ./packages/database/query/ ./packages/session/ ./packages/middleware/ ./packages/auth/ ./packages/filesystem/ ./packages/safepath/ ./packages/websocket/ ./packages/trustedproxy/ ./packages/http/ -count=1
go -C packages/mongo test -count=1
go test ./packages/database/query/ -run=^$ -fuzz=FuzzSanitizeIdentifier -fuzztime=15s
go run ./tests/securitydemo   # then ZAP baseline against :18080
```

See `docs/security/testing.md`.
