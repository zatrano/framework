# Module acquisition v1 — contract

How a **registry-resolved** package identity becomes a Go module dependency. **Not enablement, not a lockfile, not `package:install`.**

```
zatrano.package/v1
       ↓
Registry Resolve
       ├── error → NO PLAN
       └── resolved identity
              ↓
         acquire.FromResult
              ↓
            Plan
              ↓
           Targets
              ↓
          GoGetArg          ← Phase 7 ends (frozen)
              ✕
         Apply / go get     ← Phase 8, not this package
```

Phases 1–7 are frozen. This package stops at `GoGetArg`. Today's `package:install` remains enablement (enable + stubs). `package:enable`'s `go get github.com/zatrano/packages@main` is a wiring convenience, not this protocol.

This package does not run `go get`, write files, or blank-import.

| Layer | Question | Owner |
|-------|----------|--------|
| Resolve | What should be acquired? | `registry` (frozen) |
| Plan | Which `module@version`? | `FromResult` (frozen) |
| GoGetArg | Which concrete argument to `go get`? | `Plan.GoGetArg` / `Targets` (frozen) |
| Apply | How do we actually mutate the module? | Phase 8 — **not started** |

`package:install` **≠** module acquisition. It stays enablement.

## The question

ZATRANO turns a resolved identity into a Go dependency by treating the **Go module** as the acquisition unit and letting **go.mod + go.sum** be the only dependency state.

Catalog **name** is not a versioned artifact. Many official names share `github.com/zatrano/packages`. Acquiring `session` means requiring that **module** once, then enabling `session` in the app. A per-name lock version would invent a second semver beside the module — Phase 5 already forbids that.

## Translation (`zatrano.acquire/v1`)

`FromResult(registry.Result)` builds a `Plan`:

| Registry fact | Plan field | `go get` argument |
|---------------|------------|-------------------|
| `Package.Module` | `module` | left side of `@` |
| tagged `Release.Version` | `selected` | `module@v1.2.0` (`v` prefix normalized) |
| channel `main` | `selected` = `main` | `module@main` |
| `Package.Name` / `Import` | catalog identity (not enablement) | **not** a go.mod path |

`latest` never appears in `Plan.Query` or `Plan.Selected`. Resolve already picked a tag or `main`.

A `Plan` is defined only after a **successful** `Resolve`. `FromResult` does not re-check `framework_min` or re-pick versions. An incompatible query is a Resolve error — there is no plan to translate.

`go get module@main` is allowed as the **input** to the toolchain. Go then writes a **pseudo-version** (`v0.0.0-<timestamp>-<commit>`) into go.mod. That rewritten pin, plus go.sum, is what CI reproduces — not the word `main`.

Heavy packages (`mongo`, `webauthn`, `qr`) have their own module path; the same table applies. Shared-module names produce identical `Query` strings; acquiring `auth` after `session` is a no-op at the module layer.

`Targets([]Plan)` is **module-level normalization**, not Apply and not a second resolver. It deduplicates identical `Query` values per `Module` and orders by module path. Input order does not matter. If two plans share a module but disagree on `Query` (`session@main` vs `auth@v1.0.0`), that is a **conflict**: Targets returns an error. It does not pick `main`, the higher semver, or last-write-wins. Catalog `Name` never becomes a target.

`Plan.Name` / `Plan.Import` are copied identity for traceability. They are **not** enablement: the struct has no Enabled / Imported / stubs / blank-import fields. Enablement remains Enabled ∩ Imported after a later Apply.

## Why there is no `zatrano.lock`

| Need | Already owned by |
|------|------------------|
| Module version pin | `go.mod` (tag or pseudo-version) |
| Zip / go.mod hashes | `go.sum` + checksum database |
| Catalog name → “is this app using session?” | `bootstrap/enabled.go` + blank-import |
| Kernel compatibility | Registry `framework_min` at **Resolve** time |
| Manifest bytes | Optional registry `digest` (metadata, not source) |

A second lockfile would duplicate pins, drift from `go mod tidy`, and imply per-package versions on a shared module. Manifest digest and module zip hashes are **different integrity domains**; mixing them in one ZATRANO file would hide that.

Revisit a sidecar file only if a proven gap appears that go.mod/go.sum/enabled.go cannot express (for example a non-Go artifact). Do not add one “in case”.

`go get` is later **acquisition** (Phase 8). `go mod tidy` is **module/import graph rearrangement**. It is not an acquisition manifest and not a lockfile.

## Phase 8 entry (not started)

Phase 8 has **not** started. When it does, the first task is **not** coding `go get`. Order:

1. Apply SPEC
2. Mutation boundary
3. `go get` invocation contract
4. go.mod / go.sum failure semantics
5. Concurrency semantics
6. Partial-apply semantics
7. Rollback / recovery semantics
8. **Then** implementation

That contract must exist before any `exec` or filesystem mutation. It is the hand-off that keeps phases 1–6 and Phase 7's pure translation intact.

| Topic | Why it is not Phase 7 |
|-------|------------------------|
| Mutation boundary | What Apply may write (go.mod / go.sum only?) vs what it must not (enablement, stubs, blank-imports) |
| `go get` invocation | Process model, working directory, arguments (`Targets` / `GoGetArg`), environment |
| go.mod / go.sum failures | Toolchain errors vs ZATRANO errors; incomplete writes |
| Concurrency | Two Apply calls on the same module |
| Partial application | N of M `Targets` applied when one fails |
| Rollback | Restore previous go.mod/go.sum pair vs leave toolchain output |

`package:install` keeps today's enablement meaning **even at the start of Phase 8**. Binding real module acquisition to that command is evaluated only after the Apply contract is written.

`tidy` must not be assumed as “pin what we acquired.”

## Invariants (frozen)

| Property | Lock |
|----------|------|
| Same `Result` → same `Plan` | `FromResult` is a pure function |
| Same module, different catalog names, same pin → one target | `Targets` keys on `Module` |
| Same module, different pins → error | Targets **deduplicates**; it does not pick a winner |
| Target order is deterministic | lexicographic module path; input order ignored |
| `latest` never appears on a Plan | `moduleQuery` rejects it; Resolve already chose tag or `main` |
| Unresolved → no Plan | Resolve error short-circuits; `FromResult` requires name + module + version-or-`main` |
| `FromResult` does not Resolve | no `Index` methods; no `framework_min` re-check |
| `FromResult` does not touch the filesystem | no `os` / `os/exec`; Apply is a later phase |
| Plan is not enablement | field freeze; no Enabled / Imported / stubs |

Architecture tests reject a second resolution implementation, Apply/`go get`, and enablement fields on `Plan`.

## Phase freeze

**Phase 7 is closed.** Export surface: `FromResult`, `Targets`, `Plan.GoGetArg`. Next is Phase 8: Apply **contract first**, then process execution.

| Phase | Status |
|-------|--------|
| 1 Architecture | Frozen |
| 2 Package contract | Frozen |
| 3 Official packages | Frozen |
| 4 `zatrano.package/v1` | Frozen |
| 5 Registry | Frozen |
| 6 Registry CLI consumer | Frozen |
| 7 Acquisition Plan | **Frozen** |
| 8 Module Acquisition Apply | **Not started** — SPEC first, then implementation |

Today's `package:install` remains enablement. It is not module acquisition. Phase 8 must not overwrite that meaning at start.

## Deferred (Phase 8)

Apply contract (mutation boundary, `go get` invocation, go.mod/go.sum failures, concurrency, partial apply, rollback), then implementation. A new CLI command, private GOPROXY, offline, GOPROXY as a ZATRANO HTTP registry. Do not fold any of this into `package:install` at Phase 8 start. Do not treat `go mod tidy` as an acquisition lockfile.
