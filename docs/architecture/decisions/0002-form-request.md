# ADR-0002 — FormRequest is the write-input type

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

`validation.FormRequest` + `ValidateForm` + `ResponseFor` exist. Auth stubs and `ValidateRequest` allow inline rule maps. No Store/Update/Index naming is generated.

## Problem

Mutating endpoints can skip Authorize, skip flash/422 mapping, and scatter rules.

## Decision

When `validation` is enabled:

- POST/PUT/PATCH application endpoints MUST use a FormRequest in `app/http/requests`.
- Naming: `{Resource}StoreRequest`, `{Resource}UpdateRequest`, `{Resource}IndexRequest`, `{Action}Request`.
- Destroy with path id only: no request type.
- `PrepareForValidation` is the normalization site.

## Why

Single place for authorize + rules + messages + HTTP error mapping.

## Rejected alternatives

- Always inline `Make` — duplicates ResponseFor behavior.
- Separate DTO package — not generated, doubles types.

## Consequences

`make:auth` writes FormRequest types under `app/http/requests` and the auth controller calls `ValidateForm`. Dashboard stubs may still use inline `validation.Make` (remaining gap).

## Enforcement

Heuristic: Store/Update controller methods call `ValidateForm` when validation enabled.
