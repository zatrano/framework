# ZATRANO Application Engineering Standard

Canonical engineering language for ZATRANO applications.

Evidence freeze: `github.com/zatrano/framework/v2` **v2.2.0**, `github.com/zatrano/packages` **v1.7.2**, generated scaffolds (`empty` / `web` / `api` / `full`).

This is **not** a Blog architecture and not a port of another platform.

## Documents

| File | Role |
|---|---|
| [AGENTS.md](../../AGENTS.md) | AI constitution and reading order (entry) |
| [STANDARD.md](STANDARD.md) | A–Z deterministic language |
| [examples.md](examples.md) | Short sketches; **golden.md wins** on conflict |
| [golden.md](golden.md) | Golden scenarios — exact files, verbs, tests |
| [rules.md](rules.md) | Machine-enforced rule catalog |
| [rules.yaml](rules.yaml) | Machine-readable catalog |
| [doctor-report.md](doctor-report.md) | Doctor enforcement report |
| [doctor-boundary.md](doctor-boundary.md) | Adversarial verification — enforcement boundary |
| [freeze-report.md](freeze-report.md) | STANDARD freeze report |
| [unique-exists-runtime.md](unique-exists-runtime.md) | Fail-closed unique/exists runtime |
| [platform-audit.md](platform-audit.md) | Platform conformance and release audit |
| [release-candidate.md](release-candidate.md) | Release-candidate conditions |
| [no-second-way.md](no-second-way.md) | Ambiguity audit — one path per concern |
| [gaps.md](gaps.md) | Architectural gap report |
| [conflicts.md](conflicts.md) | Contradictory patterns and the chosen way |
| [completeness-matrix.md](completeness-matrix.md) | Human matrix |
| [completeness.yaml](completeness.yaml) | Machine-readable matrix |
| [enforcement.md](enforcement.md) | Documentation vs tooling vs CI |
| [roadmap.md](roadmap.md) | Implementation status after review |
| [decisions/](decisions/) | ADRs 0001–0011 — Accepted |
| [familiar-names.md](familiar-names.md) | Familiar folders vs copied behavior |
| [scans.md](scans.md) | Second/third forensic passes |

## Status

**FROZEN (Application Engineering Standard, ADR-0011).** Kernel, contracts, ORM, and public API remain frozen. ADRs 0001–0011 Accepted (ADR-0010 amended fail-closed). Spec: [STANDARD.md](STANDARD.md). Boundary: [doctor-boundary.md](doctor-boundary.md). Audit: [no-second-way.md](no-second-way.md). Runtime unique/exists: [unique-exists-runtime.md](unique-exists-runtime.md). Release audit: [platform-audit.md](platform-audit.md).

Rules are tagged:

- `IMPLEMENTED` — code, generator, or test already does this
- `ACCEPTED` — ADR accepted; generators must match
- `NOT SUPPORTED` — do not invent an API
- `INCONSISTENT` — two behaviors exist; STANDARD picks one

## One sentence

A ZATRANO application is a kernel consumer: HTTP controllers talk to FormRequests and optional application services, resolve packages with `From(app)`, persist with the first-party ORM, and respond with View or JSON — never with a second architecture.
