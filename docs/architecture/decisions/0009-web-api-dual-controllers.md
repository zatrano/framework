# ADR-0009 — Dual presentation is two controllers, not two services

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Golden scenarios need both HTML and JSON for the same resource (Post, Product, Order). STANDARD §G already forbids mixing `http.View` and `http.JSON` in one controller method. Generated `make:auth` nevertheless reuses `web.AuthController` for `/api/v1/auth` and branches with `WantsJSON()`.

## Problem

Independent agents can invent `PostWebService` / `PostApiService`, a shared controller with `WantsJSON()`, or a DTO layer “for parity.”

## Decision

For **application resources** (User admin CRUD, Post, Category, Product, Order, File):

```text
app/http/controllers/web/{resource}_controller.go
app/http/controllers/api/{resource}_controller.go
        ↓
same FormRequests · same Policy abilities · same optional Service
```

- Web methods return `http.View` or `Redirect`.
- API methods return `http.JSON`.
- Do **not** create `WebService` / `ApiService` / `WebUseCase` / `ApiUseCase`.
- Share an application service **only** when STANDARD §H requires a service (multi-write, transaction, reuse). Simple CRUD duplicates thin controllers that both call ORM.

**Exception (do not copy for resources):** `make:auth` generated `AuthController` may serve both web and API via `WantsJSON()`. That is the authentication generator’s stub, not the resource CRUD architecture. Do not redesign `make:auth` in this report.

## Why

Matches scaffold packages (`controllers/web` vs `controllers/api`), ADR-0007 presentation, and ADR-0001 (no extra layers for transport).

## Rejected alternatives

- One controller, `WantsJSON()` on every resource method — second architecture; breaks View-only web tests.
- Transport-specific services — duplicates business rules.
- Shared “presenter” / Resource required for JSON — ADR-0008 already made `http.JSON` the default.

## Consequences

Golden Post CRUD always has two controller files when both transports exist. API omits `Create`/`Edit` view methods.

## Enforcement

Doctor: `http.View` and `http.JSON` in the same method is a violation except `auth_controller.go` / `social_auth_controller.go` and types `AuthController` / `SocialAuthController`.
