# ZATRANO Application Engineering Standard

Canonical engineering language for ZATRANO applications.

Evidence freeze: `github.com/zatrano/framework/v2` **v2.1.0**, `github.com/zatrano/packages` **v1.7.1**, generated scaffolds (`empty` / `web` / `api` / `full`).

This is **not** a Blog architecture, a Laravel port, or Clean Architecture.

## Documents

| File | Role |
|---|---|
| [AGENTS.md](../../AGENTS.md) | AI constitution and reading order (entry) |
| [STANDARD.md](STANDARD.md) | A–Z deterministic language |
| [examples.md](examples.md) | Short sketches; **golden.md wins** on conflict |
| [golden.md](golden.md) | Phase 2 golden scenarios — exact files, verbs, tests |
| [rules.md](rules.md) | Phase 3 machine-enforced rule catalog |
| [rules.yaml](rules.yaml) | Machine-readable catalog |
| [phase3.md](phase3.md) | Phase 3 enforcement report |
| [phase3.5.md](phase3.5.md) | Phase 3.5 adversarial verification — enforcement boundary |
| [gaps.md](gaps.md) | Architectural gap report |
| [conflicts.md](conflicts.md) | Contradictory patterns and the chosen way |
| [completeness-matrix.md](completeness-matrix.md) | Human matrix |
| [completeness.yaml](completeness.yaml) | Machine-readable matrix |
| [enforcement.md](enforcement.md) | Documentation vs tooling vs CI |
| [roadmap.md](roadmap.md) | Implementation phases after review |
| [decisions/](decisions/) | ADRs 0001–0010 — Accepted |
| [laravel-similarity.md](laravel-similarity.md) | Familiar folders vs copied behavior |
| [scans.md](scans.md) | Second/third forensic passes |

## Status

**Phase: 3.5 complete (adversarial verification of `zatrano doctor`).** Kernel, contracts, ORM, and public API remain frozen. ADRs 0001–0010 Accepted. Enforcement boundary: [phase3.5.md](phase3.5.md).

Rules are tagged:

- `IMPLEMENTED` — code, generator, or test already does this
- `ACCEPTED` — ADR accepted; generators must match
- `NOT SUPPORTED` — do not invent an API
- `INCONSISTENT` — two behaviors exist; STANDARD picks one

## One sentence

A ZATRANO application is a kernel consumer: HTTP controllers talk to FormRequests and optional application services, resolve packages with `From(app)`, persist with the first-party ORM, and respond with View or JSON — never with a second architecture.
