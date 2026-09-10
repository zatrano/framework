# ADR-0007 — Controller generator must match presentation profile

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Web `HomeController` returns `http.View`. API home returns `http.JSON`. `make:controller` always emits JSON, including `package web`.

## Problem

New web controllers look like API controllers. Agents copy JSON.

## Decision

`make:controller` without `--api` emits a View (or Redirect) stub. `--api` emits JSON. `--admin` follows web unless later specified.

Until implemented, humans/AI **overwrite** the stub; the JSON web stub is a known non-canonical artifact.

## Why

`zatrano new --web|--api|--full` presentation contract.

## Rejected alternatives

- Always JSON — breaks web scaffold.
- Always View — breaks API.

## Consequences

Generator change in implementation phase (no kernel ABI).

## Enforcement

Template tests after the generator fix.
