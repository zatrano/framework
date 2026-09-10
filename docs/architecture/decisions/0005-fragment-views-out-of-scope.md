# ADR-0005 — HTML fragments are not a ZATRANO application API

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Some product language treats partial-swap HTML as the default frontend. Framework and packages contain **zero** fragment APIs, middleware, or view helpers for that model. Web rendering is `http.View` + Go templates + Redirect + flash.

## Problem

Agents will invent fragment controllers, swap-header branches, and parallel JSON/HTML methods.

## Decision

HTML fragment / partial-swap clients are **NOT SUPPORTED**. Canonical web is full-page views and redirects. Applications may attach their own scripts in `public/js` as raw assets, but that is **outside** the engineering standard (no fragment conventions, no official helpers).

If an official fragment addon is later adopted, it requires a new ADR that supersedes this one.

## Why

Repository evidence. Do not invent architecture.

## Rejected alternatives

- Pretend fragments are canonical “where applicable” — creates a second undocumented web stack.
- Ban extra scripts in `public/` — unnecessary; only the **platform API** is out of scope.

## Consequences

STANDARD §S ignores fragment partials as a first-class response type.

## Enforcement

Documentation; later doctor if `app/http` grows fragment helper packages without an addon.
