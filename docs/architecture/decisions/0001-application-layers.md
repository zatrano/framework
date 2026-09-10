# ADR-0001 — Application layers

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

ZATRANO generates controllers, optional services, optional repositories, FormRequests, models. It does not generate UseCases, Actions, Entities, or DTOs. `packages/bus` exists as an optional command bus.

## Problem

Engineers can still introduce Clean Architecture / CQRS folders and claim compliance.

## Decision

Canonical stack:

```text
Controller → FormRequest (writes) → optional Service → orm.* / pkg.From(app)
```

Rejected as application types: UseCase, Action, DomainService, DTO, Entity (DDD), ValueObject, Handler (as HTTP type).

`bus.Dispatch` is opt-in, never the default CRUD path.

## Why

Matches generators, `From(app)`, and kernel freeze. Minimum layers that the platform actually owns.

## Rejected alternatives

- Mandatory service for every controller method — contradicts simple HomeController and CRUD.
- Mandatory repository — generator is optional concrete wrapper.
- Bus-first — optional package, reflection-ish `any` commands.

## Consequences

AI must not create `app/domain`. Simple CRUD may skip services.

## Enforcement

Doctor directory/package/type checks (APP-LAY-001/002/003), including `interactors/` and `app/application/`. `app/core` names are SEMANTIC (phase 3.5). No `make:usecase`.
