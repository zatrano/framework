# ADR-0008 — Default API body is http.JSON

- **Status:** Accepted
- **Date:** 2026-09-10

## Context

API home uses `http.JSON`. `packages/jsonapi` and `make:resource` exist as optional transformers.

## Problem

Agents may wrap every endpoint in JSON:API documents or Resource classes.

## Decision

Default: `http.JSON` of maps or structs (models with json tags, or explicit maps). `jsonapi` and `app/http/resources` are opt-in when the product requires JSON:API or reused transformers.

## Why

Generated API apps do not enable jsonapi. Keep one default.

## Rejected alternatives

- JSON:API mandatory — extra package and shape not in templates.
- Resource class mandatory — factory command is not the API scaffold.

## Consequences

Filtering example returns `{data, meta}` maps unless jsonapi is enabled.

## Enforcement

Documentation only unless jsonapi is in EnabledAddons (then allow Document types).
