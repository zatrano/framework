# Module acquisition v1 — contract

How a **registry-resolved** package identity becomes a Go module dependency. **Not enablement, not a lockfile, not `package:install`.**

```
zatrano.package/v1
       ↓
Registry Resolve
       ↓
Resolved identity (name + module + selected release)
       ↓
acquire.Plan          ← this package (translation only)
       ↓
go get <Plan.Query>   ← later Apply; mutates go.mod / go.sum
       ↓
Go verification (go.sum + checksum database)
       ↓
Application enablement (Enabled ∩ Imported) — still a separate step
```

Phases 1–6 stay frozen. Today's `package:install` remains enablement (enable + stubs). `package:enable`'s `go get github.com/zatrano/packages@main` is a wiring convenience, not this protocol.

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
| `Package.Name` / `Import` | identity for later enablement | **not** a go.mod path |

`latest` never appears in `Plan.Query`. Resolve already picked a tag or `main`.

`go get module@main` is allowed as the **input** to the toolchain. Go then writes a **pseudo-version** (`v0.0.0-<timestamp>-<commit>`) into go.mod. That rewritten pin, plus go.sum, is what CI reproduces — not the word `main`.

Heavy packages (`mongo`, `webauthn`, `qr`) have their own module path; the same table applies. Shared-module names produce identical `Query` strings; acquiring `auth` after `session` is a no-op at the module layer.

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

## Verification (later Apply)

1. `Plan` came from `registry.Resolve` (not from a CLI-local picker).
2. `go get` / `go mod download` using `Plan.Query`.
3. Trust Go's `go.sum` + sumdb for source.
4. Optional: `registry.VerifyDigest` on manifest bytes — metadata only.
5. Enablement is a **following** step, not part of Apply.

Failure/rollback of Apply is “leave go.mod/go.sum as Go left them or restore the previous pair.” That policy is for the Apply implementation, not this translation contract.

## Deferred

`Apply` (`exec.Command("go", "get", …)`), upgrade/downgrade UX, private GOPROXY, offline, `package:add` CLI, GOPROXY as a ZATRANO HTTP registry, folding any of this into `package:install`.
