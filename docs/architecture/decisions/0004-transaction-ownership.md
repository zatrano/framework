# ADR-0004 — Services own transactions

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

`orm.Transaction` + `QueryTx` exist. Nested transactions and savepoints do not. Controllers can call Transaction today.

## Problem

Nested calls rollback incorrectly; HTTP layers mix TX with redirects; jobs dispatched inside a TX that fails still run.

## Decision

- Multi-write workflows live in `app/services`.
- That service calls `orm.Transaction`.
- Inside the callback, only `QueryTx` / `QueryOn`.
- Controllers never start transactions.
- Queue/notification dispatch happens **after** a nil return from Transaction.

Single-row `Create`/`update` needs no transaction.

## Why

One owner, no nesting, matches ORM API.

## Rejected alternatives

- Controller TX — violates controller rules, hard to reuse from jobs.
- Database package TX for models — second API.
- Unit of Work type — not in the platform.

## Consequences

`make:service` must become a real workflow host, not `Handle() error`.

## Enforcement

AST: `orm.Transaction` only in `app/services` and `app/console`.
