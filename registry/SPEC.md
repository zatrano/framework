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

## Discovery

`Search` filters identity (name, description, kind, layer, heavy). It does not pick a version.

## Resolution

`Resolve(Query)` picks one release:

1. Unknown `name` is an error.
2. Optional `kind` filter must match.
3. `framework` (kernel VERSION): when set, skip releases whose `framework_min` is not met. Empty min always meets. An empty `framework` query does not filter.
4. `version` empty or `latest`: highest tagged semver that remains; if none, channel `main` if present and compatible.
5. `version` `main`: that channel.
6. other `version`: exact tagged match (`v` prefix optional).

Resolution does not blank-import, enable, or run `Register`. Runtime remains Enabled ∩ Imported.

## Integrity

If `digest` is set, `Verify` compares SHA-256 of the manifest bytes (hex). Missing digest is not an error in v1.

## Deferred

HTTP registry service, `package:install` / `update`, GOPROXY as the transport, publisher identity, licenses, checksums of zip/mod files (Go sumdb), yank/retract beyond skipping a version in the index.

## Compatibility

Same policy as the package manifest: unknown JSON fields ignored; new required fields need `zatrano.registry/v2`.
