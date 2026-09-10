# ADR-0003 — Repositories are optional

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

`make:repository` writes a concrete struct (`All`/`Find`/`Create` wrapping orm). No interfaces. ORM is already the abstraction over SQL.

## Problem

Teams may invent `BlogRepository` interfaces “for testing” on every model.

## Decision

Default persistence API is `orm.Query[T]()` / `Find` / `Create`. Repositories are optional seams for repeated queries or fakes. No per-model interface requirement.

## Why

ORM is generic and testable with a test database. Extra interfaces are not in the generator and are not safer.

## Rejected alternatives

- Repository mandatory — extra files, no benefit.
- Direct `database/query` in controllers — skips model events and Fillable.

## Consequences

Controllers/services may import `packages/orm`. That is allowed.

## Enforcement

Doctor: no rule *requires* repositories. APP-REP-001 **errors** on exported `*Repository` interfaces and `BaseRepository` / `GenericRepository` / `*RepositoryFactory`. Concrete structs PASS.
