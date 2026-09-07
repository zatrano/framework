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
           Targets          ← Phase 7 ends (frozen)
              ↓
     unique acquisition units
              ✕
         Apply / go get     ← Phase 8, not this package
```

Phases 1–7 are frozen. This package stops at `Targets`. Today's `package:install` remains enablement (enable + stubs). `package:enable`'s `go get github.com/zatrano/packages@main` is a wiring convenience, not this protocol.

This package does not run `go get`, write files, or blank-import.

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

## Verification (later Apply — Phase 8)

Phase 8 starts with an **Apply contract**, not by rewriting `package:install`. Enablement (Enabled ∩ Imported + stubs) stays a separate operation.

1. `Plan` came from `registry.Resolve` (not from a CLI-local picker). `Targets` only listed unique modules; it did not re-resolve.
2. Mutate go.mod / go.sum using `Plan.Query` / `Targets` output. Policy for `go get` vs `go mod edit` belongs in that contract.
3. Do **not** code `go get` → `go mod tidy` as the install mechanism. `tidy` rearranges the module graph from source imports; it is not “pin the packages we just acquired.”
4. Trust Go's `go.sum` + sumdb for source.
5. Optional: `registry.VerifyDigest` on manifest bytes — metadata only.
6. Enablement is a **following** step, not part of Apply.

Failure/rollback of Apply is “leave go.mod/go.sum as Go left them or restore the previous pair.” That policy is for the Apply implementation, not this translation contract.

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

**Phase 7 is closed.** This package stops at `Targets`. Next is Phase 8: Apply contract, then process execution.

| Phase | Status |
|-------|--------|
| 1 Architecture | Frozen |
| 2 Package contract | Frozen |
| 3 Official packages | Frozen |
| 4 `zatrano.package/v1` | Frozen |
| 5 Registry | Frozen |
| 6 Registry CLI consumer | Frozen |
| 7 Acquisition Plan (`FromResult` / `Targets`) | **Frozen** |
| 8 Module Acquisition Apply | Not started — contract first |

Today's `package:install` remains enablement. Phase 8 must not overwrite that meaning.

## Deferred (Phase 8)

`Apply` (`go get`, `go mod edit`, go.sum mutation), process execution, failure/rollback, upgrade/downgrade UX, private GOPROXY, offline, a new CLI command, GOPROXY as a ZATRANO HTTP registry. Do not fold any of this into `package:install`. Do not assume `go mod tidy` pins acquired modules.
