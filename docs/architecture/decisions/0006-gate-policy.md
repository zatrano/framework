# ADR-0006 — Authorization is Gate and Policy

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

`packages/authorization` implements Gate, Policy, before/after callbacks. Auth boot binds the Gate. Dashboard stubs include role/permission **files** that are not the package API.

## Problem

Controllers may check `role == "admin"` or skip checks after validation.

## Decision

- Resource authorization: `make:policy` + Gate `Authorize` **before** mutating and before leaking private representations.
- FormRequest `Authorize` is coarse HTTP access.
- Dashboard RBAC stubs are UI/persistence experiments, **not** the application authorization model.
- There is no first-party roles/permissions service API to call.

## Why

Only Gate/Policy is a real package surface with tests.

## Rejected alternatives

- Dashboard role package in the app — not shipped.
- Middleware-only abilities without policies — does not scale to ownership.

## Consequences

Golden “authorization” domain uses Policy, not role tables, unless a future package ADR ships RBAC.

## Enforcement

SEMANTIC / review: `role ==` in controllers is non-compliant. Doctor does **not** scan role strings (false-positive risk). Gate/Policy remains the API.
