# ADR-0005 — HTMX is not a ZATRANO application API

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

Some product language treats HTMX as the default frontend. Framework and packages contain **zero** HTMX APIs, middleware, or view helpers. Web rendering is `http.View` + Go templates + Redirect + flash.

## Problem

Agents will invent fragment controllers, `HX-Request` branches, and parallel JSON/HTML methods.

## Decision

HTMX is **NOT SUPPORTED**. Canonical web is full-page views and redirects. Applications may attach HTMX in `public/js` as raw assets, but that is **outside** the engineering standard (no fragment conventions, no official helpers).

If an official HTMX addon is later adopted, it requires a new ADR that supersedes this one.

## Why

Repository evidence. Do not invent architecture.

## Rejected alternatives

- Pretend HTMX is canonical “where applicable” — creates a second undocumented web stack.
- Ban HTMX in `public/` — unnecessary; only the **platform API** is out of scope.

## Consequences

STANDARD §S ignores fragment partials as a first-class response type.

## Enforcement

Documentation; later doctor if `app/http` grows `hx` helper packages without an addon.
