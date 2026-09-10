# ZATRANO Application Engineering Standard

Canonical engineering language for ZATRANO applications.

Evidence freeze: `github.com/zatrano/framework/v2` **v2.1.0**, `github.com/zatrano/packages` **v1.7.1**, generated scaffolds (`empty` / `web` / `api` / `full`).

This is **not** a Blog architecture, a Laravel port, or Clean Architecture.

## Documents

| File | Role |
|---|---|
| [AGENTS.md](../../AGENTS.md) | AI constitution and reading order (entry) |
| [STANDARD.md](STANDARD.md) | A–Z deterministic language |
| [examples.md](examples.md) | Canonical flow sketches (not demo apps) |
| [gaps.md](gaps.md) | Architectural gap report |
| [conflicts.md](conflicts.md) | Contradictory patterns and the chosen way |
| [completeness-matrix.md](completeness-matrix.md) | Human matrix |
| [completeness.yaml](completeness.yaml) | Machine-readable matrix |
| [enforcement.md](enforcement.md) | Documentation vs tooling vs CI |
| [roadmap.md](roadmap.md) | Implementation phases after review |
| [decisions/](decisions/) | ADRs — proposed canonical choices |
| [laravel-similarity.md](laravel-similarity.md) | Familiar folders vs copied behavior |
| [scans.md](scans.md) | Second/third forensic passes |

## Status

**Phase: forensic extraction.** No public APIs, ORM, generators, or application layouts were changed to produce this standard.

Rules are tagged:

- `IMPLEMENTED` — code, generator, or test already does this
- `ACCEPTED` — ADR accepted; generators must match
- `NOT SUPPORTED` — do not invent an API
- `INCONSISTENT` — two behaviors exist; STANDARD picks one

## One sentence

A ZATRANO application is a kernel consumer: HTTP controllers talk to FormRequests and optional application services, resolve packages with `From(app)`, persist with the first-party ORM, and respond with View or JSON — never with a second architecture.
