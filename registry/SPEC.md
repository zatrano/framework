# Package registry v1 — data model and resolution

Machine-readable **index** of ZATRANO packages. **Not a marketplace, not a boot protocol, not an HTTP API.**

```
Package source → zatrano.package/v1 → registry index → CLI / marketplace / third parties
```

CLI install/update, publisher accounts, payments, ratings, and market.zatrano.com are out of scope. Those consume this model later.

This package does not import `bootstrap/addons`. The process-global addon registry stays the runtime contract.

## Authoritative facts

| Fact | Source of truth |
|------|-----------------|
| Package **name** | Catalog id / `addons.Meta.Name` (`zatrano.package/v1` `name`) |
| Go **import** | Manifest `import` |
| Version **stream** | Go **module path** (`module`), not the package name |
| Artifact **version** | Go module version (git tag) or channel `main` when the module is untagged |
| Kernel compatibility | Manifest `framework_min` (empty = unspecified) |
| Kind / layer / heavy | Manifest (frozen v1) |
| Integrity | Optional SHA-256 of the manifest document bytes |

Many official packages share `github.com/zatrano/packages` and therefore share one version stream. Today that stream is channel `main` (the module is not tagged `v2.x`). Heavy packages (`mongo`, `webauthn`, `qr`) have their own module path and their own tags. `console` versions with `github.com/zatrano/framework/v2`.

Do not invent a second semver field on the package name.

## Channels vs published versions

`channel: main` is the **source/development stream** of a Go module (typically the default git branch). It is **not** a published release version. A marketplace or UI “Latest / stable” label must show a tagged version, or state that the module is untagged — never present `main` as a release.

`latest` is a **resolve selector**, not a channel:

```
latest
  ├─ compatible tag exists → highest compatible tag
  └─ no compatible tag     → main (source channel, if present and compatible)
```

`session@main` means “this module’s source channel”, not “the current stable release”.

## Index document (`zatrano.registry/v1`)

```json
{
  "schema": "zatrano.registry/v1",
  "packages": [
    {
      "name": "session",
      "import": "github.com/zatrano/packages/session",
      "module": "github.com/zatrano/packages",
      "kind": "service",
      "layer": "foundation",
      "description": "HTTP sessions",
      "releases": [
        { "channel": "main" }
      ]
    }
  ]
}
```

A **release** is either a tagged version (`version`) or a channel (`channel`). At least one is required. `framework_min` and `digest` are optional.

Names and import paths are unique in one index.

## Discovery vs resolution vs install

These are three different operations. This package implements only the first two.

| Operation | Meaning | Mutates |
|-----------|---------|---------|
| **Search** | Metadata discovery (identity filters) | No |
| **Resolve** | Compatible version selection | No |
| **Install** | Filesystem / go.mod / source mutation | Yes — **not defined here** |

`Search` filters identity (name, description, kind, layer, heavy). It does not pick a version.

## Resolution

`Resolve(Query)` picks one release. It does not enable, import, or write `go.mod`.

1. Unknown `name` is an error.
2. Optional `kind` filter must match.
3. `framework` (kernel VERSION): when set, skip releases whose `framework_min` is not met. Empty min always meets. An empty `framework` query does not filter.
4. `version` empty or `latest`: highest tagged semver that remains; if none, channel `main` if present and compatible (source fallback, not a published version).
5. `version` `main`: that channel.
6. other `version`: exact tagged match (`v` prefix optional).

Runtime remains Enabled ∩ Imported. A CLI, marketplace, or IDE must call this algorithm rather than reimplement it.

## Consumer invariant

The CLI is **never** a second implementation of package resolution.

Consumers (`package:search`, `package:info`, `package:resolve`, and any later HTTP registry, marketplace, or IDE) call `Index.Search`, `Index.Lookup`, and `Index.Resolve`. They pass a `Query` / `Filter` and print the result. Changing `Resolve` constraints must not require a parallel update of selection logic in the CLI.

`package:enable` turns on an already imported package in the application. Today's `package:install` is the same enablement family (enable + stubs). It is **not** download, `go.mod` mutation, or `Resolve`. A future module-install command is a separate operation and must not overwrite this meaning.

## Phase freeze

Phases 1–6 are closed at this boundary:

| Surface | Status |
|---------|--------|
| Kernel architecture / `contracts.App` | Frozen |
| Package contract (Enabled ∩ Imported) | Frozen |
| Official packages | Frozen |
| `zatrano.package/v1` | Frozen |
| `zatrano.registry/v1` Search / Resolve | Frozen |
| CLI consumer (`package:search` / `info` / `resolve`) | Frozen |

The next phase is **module acquisition**. Translation of `registry.Result` → `go get` argument is [`acquire/SPEC.md`](../acquire/SPEC.md). Apply (filesystem mutation) is still deferred. Do not fold acquisition into `package:install`.

A later HTTP registry must implement the same `Search` / `Lookup` / `Resolve` contract so the CLI can swap the index source without copying semver logic.

## Integrity

If `digest` is set, `Verify` compares SHA-256 of the manifest bytes (hex). Missing digest is not an error in v1.

## Deferred

HTTP registry service, GOPROXY as a ZATRANO protocol, `Apply` of [`acquire.Plan`](../acquire/SPEC.md), publisher identity, licenses, yank/retract beyond skipping a version in the index. Existing `package:enable` / `package:install` remain enablement.

## Compatibility

Same policy as the package manifest: unknown JSON fields ignored; new required fields need `zatrano.registry/v2`.
