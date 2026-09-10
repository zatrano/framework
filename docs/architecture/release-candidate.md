# ZATRANO Release Candidate

Phase 5 conditions closure. Architecture is not redesigned. This report records whether Framework `v2.2.0` and Packages `v1.7.2` are ready to tag.

Inspected 2026-09-10. Repository state is authoritative. Phase 5 audit history in [phase5.md](phase5.md) is not rewritten.

## Source State

Framework:

```text
commit:          (annotated tag v2.2.0 on this release commit)
branch:          main
previous public tag: v2.1.0 (fc9047d, immutable)
candidate version:   v2.2.0
module:          github.com/zatrano/framework/v2
Go:              go1.25.0 windows/amd64
CGO_ENABLED:     0
```

Packages:

```text
commit:          a2c0c66a992b486086f79a66cac1d3d3dba13553
branch:          main
previous public tag: v1.7.1 (immutable)
candidate version:   v1.7.2
module:          github.com/zatrano/packages
require:         github.com/zatrano/framework/v2 v2.0.28
```

Local packages working tree remains dirty with unrelated WIP. That WIP is **not** in `a2c0c66` and is **not** part of tag `v1.7.2`. Verification of packages used a clean git worktree of `a2c0c66`.

## Version determination

Canonical Framework identity is the `VERSION` file (runtime fallback `console/version.go` `currentRelease`). Both are `2.2.0`. Historical tag `v2.1.0` already exists and must not be moved.

Post-`v2.1.0` on main is application-architecture enforcement (`zatrano doctor`), generator STANDARD alignment, Phase 4 freeze, Phase 4.5 docs, and Phase 5 audit. Kernel ABI, `contracts`, and ORM public API are unchanged. Under the existing pre-3.0 policy, that is a **MINOR** (`2.2.0`), not a patch (the surface is more than a bugfix) and not a major (no kernel/contract break).

Packages `v1.7.1` still fail-opens `unique`/`exists` when no checker is bound or the rule is malformed. HEAD `288bb2f` (included in `a2c0c66`) fail-closes those paths. Same correction family as `v1.7.1` (fail-closed Redis). **PATCH** `v1.7.2`. The module pin stays `framework v2.0.28`; fail-closed does not need newer kernel APIs.

## P5-H1

```text
Status: CLOSED
```

`VERSION`, `currentRelease`, README badge, CHANGELOG `## 2.2.0 - 2026-09-10`, PACKAGES.md current pair, first-time enablement pin, and current-version tests now agree on **2.2.0** / packages **v1.7.2**. Historical `v2.1.0` / `v1.7.1` headings and the frozen-invariants baseline sentence remain. Public tag `v2.1.0` is not reused.

## P5-H2

```text
Status: CLOSED
```

Published `v1.7.1` `checkPresence`: `checker == nil` → `return true`; malformed `table,column` → `return true` (fail-open). Checker `err` already returned false.

HEAD / candidate `v1.7.2` `checkPresence`: no checker → `false`; missing table/column → `false`; checker `err` → `false`; empty table/column does not call the checker.

Tests: `validation/presence_test.go` (unique/exists existing, absent, database error, checker unavailable) and `validation/form_request_test.go` (`ValidateForm` unique unavailable/error/existing/absent; exists existing/absent). `go test ./validation` PASS on `a2c0c66`.

## P5-H3

```text
Status: CLOSED
```

Unrelated WIP left untouched in the packages working tree. Not committed. Not tagged.

| Path | Classification | What |
|---|---|---|
| `ai/publish.go`, `billing/publish.go`, `mongo/publish.go`, `oauth/publish.go`, `social/publish.go`, `webauthn/publish.go` | UNRELATED WIP | Untracked config blob constants |
| `ai/provider.go`, `billing/provider.go`, `mongo/provider.go`, `oauth/provider.go`, `social/provider.go`, `webauthn/provider.go` | UNRELATED WIP | `ConfigFiles` pointing at those blobs |
| `bootutil/cli.go` | UNRELATED WIP | Removes `ConsoleStubsDir` / `goModReplace` |
| `auth/stubs.go` | UNRELATED WIP | Embed-only stubs; drops `ConsoleStubsDir` fallback |

None of this is fail-closed validation, Phase 5, or required by `v1.7.2`. Production `a2c0c66` still has `ConsoleStubsDir`. Including it would ship an unfinished stub/config experiment.

## Verification

```text
framework go test:           PASS  (`go test ./... -timeout 30m`)
framework go vet:            PASS
framework doctor:            PASS (framework root: 0 errors, 1 expected `no app/` warning)
framework doctor --strict:   PASS on generated empty/web/api/full consumers
                             framework root --strict exits 1 on the `no app/` warning — not an architecture failure
framework golden:            PASS (`tests/compatibility`)
framework generators:        PASS (`console` new empty/web/api/full + add:* tests)

packages go test:            PASS (`a2c0c66` worktree `go test ./...`)
packages go vet:             PASS (`a2c0c66` worktree)

race:                        NOT EXECUTABLE — ENVIRONMENTAL LIMITATION
                             (`go: -race requires cgo`; CGO_ENABLED=0; gcc unavailable)
CI RACE COVERAGE:            CONFIGURED
                             framework `.github/workflows/security.yml` job `go-test-race`
                             packages `.github/workflows/security.yml` job `go-test-race`
```

Generated consumers (`zatrano new` empty / `--web` / `--api` / `--full` with `--replace`): `go test ./...`, `go vet ./...`, `zatrano doctor`, `zatrano doctor --strict` all PASS.

Security: no new kernel/auth surface in this RC. Packages change is fail-closed validation (hardening). CI gosec remains configured. No security regression found in this preparation.

## Release Tree

Packages tag `v1.7.2` = commit `a2c0c66` only. Unrelated WIP is excluded.

Framework tag `v2.2.0` = this release commit only (version metadata, CHANGELOG, current-docs pins, this report). No architecture/code redesign.

## Compatibility

```text
packages@v1.7.2 requires github.com/zatrano/framework/v2 v2.0.28 (minimum)
packages@v1.7.2 is compatible with framework v2.2.0
framework v2.2.0 does not import github.com/zatrano/packages
framework first-time enablement pin: github.com/zatrano/packages@v1.7.2
packages version.current fallback remains 2.0.28 (packages module identity, not a kernel bump)
```

The packages `replace` to `../framework` is unchanged from `v1.7.1` (local development; ignored by the module proxy for consumers).

## Remaining Conditions

```text
No outstanding Phase 5 release conditions.
```

Push and GitHub Release creation are **not** part of this candidate. They remain operator steps after review.

## Recommendation

```text
RELEASE READY
```
