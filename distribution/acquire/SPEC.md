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
          GoGetArg          ← Acquisition plan ends (frozen)
              ✕
         Invoke / Runner    ← Apply contract process boundary
              ✕
         Execute / go get   ← Apply contract execution
              ✕
         Inspect            ← Apply contract go.mod / go.sum
              ✕
         per-root lock      ← Apply contract concurrency
              ✕
         ExecuteTargets     ← Apply contract partial apply
              ✕
         RecoverFiles       ← Apply contract best-effort file recovery
              ✕
         integration tests  ← Apply contract completion evidence
```

Architecture, package, registry, and acquisition-plan contracts are frozen. Acquisition plan stops at `GoGetArg`. Today's `package:install` remains enablement (enable + stubs). `package:enable`'s `go get github.com/zatrano/packages@main` is a wiring convenience, not this protocol.

Acquisition plan (`plan.go`) does not run `go get`, write files, or blank-import. `Invoke` takes a `Runner`; tests inject a fake. `Execute` binds `ExecRunner` and serializes mutation per module root. `Inspect` reads go.mod / go.sum, does not mutate them, and does not take the mutation lock.

| Layer | Question | Owner |
|-------|----------|--------|
| Resolve | What should be acquired? | `registry` (frozen) |
| Plan | Which `module@version`? | `FromResult` (frozen) |
| GoGetArg | Which concrete argument to `go get`? | `Plan.GoGetArg` / `Targets` (frozen) |
| Invoke | How is the process called? | `Invoke` / `Runner` |
| Execute | Did `go get` run, and what did it return? | `Execute` / `ExecRunner` |
| Inspect | What did Go write into go.mod / go.sum? | `Inspect` |
| Lock | Exclusive mutation per module root | `lockMutation` |
| Apply | How do we report mutation / partial / rollback? | `ExecuteTargets` / `ApplyResult`; `RecoverFiles` (files only, not guaranteed) |

`package:install` **≠** module acquisition. It stays enablement.

## The question

ZATRANO turns a resolved identity into a Go dependency by treating the **Go module** as the acquisition unit and letting **go.mod + go.sum** be the only dependency state.

Catalog **name** is not a versioned artifact. Many official names share `github.com/zatrano/packages`. Acquiring `session` means requiring that **module** once, then enabling `session` in the app. A per-name lock version would invent a second semver beside the module — the registry contract already forbids that.

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

`go get` is later **acquisition** (Apply). `go mod tidy` is **module/import graph rearrangement**. It is not an acquisition manifest and not a lockfile.

## Apply contract freeze

Apply contract: [`APPLY.md`](APPLY.md). Implementation steps 1–8 are complete and frozen. A new CLI command is a later specification, not a remaining Apply-contract gate. Do not recode it here. Acquisition/enablement: [`ORCHESTRATION.md`](ORCHESTRATION.md) — SPEC accepted; A complete / FROZEN; B (`package:acquire`) implemented; C (`--enable`) COMPLETE / FROZEN at `v2.0.27`.

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
| `FromResult` does not touch the filesystem | no `os` / `os/exec`; Apply is a later specification |
| Plan is not enablement | field freeze; no Enabled / Imported / stubs |

Architecture tests reject a second resolution implementation, enablement fields on `Plan`, and `go get` / `os/exec` in `plan.go`. Process starts belong in `ExecRunner` only.

## Contract freeze

**The acquisition plan is closed.** Export surface: `FromResult`, `Targets`, `Plan.GoGetArg`. **The Apply contract is closed.** Apply surface: `Invoke` / `Runner` / `ExecRunner` / `Execute` / `Inspect` / `ExecuteTargets` / `ApplyResult` / `SnapshotFiles` / `RecoverFiles`. No `func Apply`. Apply contract SPEC: [`APPLY.md`](APPLY.md). A new CLI command remains not authorized; it is not an Apply-contract remaining gate.

| Contract | Status |
|----------|--------|
| Architecture | Frozen |
| Package contract | Frozen |
| Official packages | Frozen |
| `zatrano.package/v1` | Frozen |
| Registry | Frozen |
| Registry CLI consumer | Frozen |
| Acquisition Plan | **Frozen** |
| Module Acquisition Apply | **Frozen** (`Invoke` / `Execute` / `Inspect` / `ExecuteTargets` / `RecoverFiles`; no `func Apply`) |
| Dry-run / CLI acquisition / Acquisition ↔ enablement | **COMPLETE / FROZEN** — A complete / FROZEN; B (`package:acquire`) implemented; C (`--enable`) COMPLETE / FROZEN at `v2.0.27`; [`ORCHESTRATION.md`](ORCHESTRATION.md) |
| Production Hardening & Ecosystem Validation | **COMPLETE**; [`HARDENING.md`](HARDENING.md) |

Today's `package:install` remains enablement. It is not module acquisition. Apply contract must not overwrite that meaning.

## Out of Apply contract (later specification)

Acquisition/enablement: [`ORCHESTRATION.md`](ORCHESTRATION.md). SPEC accepted. Contract A (dry-run) is complete / FROZEN. Contract B (`package:acquire`) is implemented. Contract C is COMPLETE / FROZEN at `v2.0.27`: `--enable` after successful acquisition; default acquire does not enable. Do not fold this into `package:install`. Do not treat `go mod tidy` as an acquisition lockfile. Private GOPROXY, offline, and GOPROXY as a ZATRANO HTTP registry stay deferred.

Production hardening: [`HARDENING.md`](HARDENING.md). SPEC is accepted. Implementation is COMPLETE. Do not reopen the Apply contract or redesign orchestration contracts.
