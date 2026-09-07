# Package manifest v1

Distribution protocol for ZATRANO packages. **Not a runtime architecture.**

```
Runtime contract (frozen)     Package manifest (this document)
Enabled ∩ Imported            name, import, module
addons.Register / Requires    identity + optional requires copy
Provider.Register/Boot        — not in the manifest
LifecycleProvider             — not in the manifest
        ↓
Package Registry → CLI → verification → Marketplace
```

Kernel primitives (`kernel/catalog.go`) are not packages. They do not have manifests.

## Schema

JSON object. `"schema": "zatrano.package/v1"`.

Unknown properties are ignored (forward compatible). Missing optional properties are valid (backward compatible). A future `zatrano.package/v2` is a new `schema` value; v1 documents stay readable.

### Required

| Field | Meaning |
|-------|---------|
| `schema` | `zatrano.package/v1` |
| `name` | Catalog id, `[a-z][a-z0-9]*` |
| `import` | Go import path |
| `kind` | `service` (package:enable) or `library` (import-only) |
| `layer` | `foundation`, `intelligence`, or `addon` |
| `description` | One-line summary |

### Optional (only when the real package has the fact)

| Field | Meaning |
|-------|---------|
| `module` | Go module path. Default `github.com/zatrano/packages`. Heavy packages use `github.com/zatrano/packages/<name>`. `console` uses `github.com/zatrano/framework/v2`. |
| `heavy` | Separate module / heavy dependency |
| `key` | Container binding key when the package binds one |
| `requires` | Addon names that must be imported (copy of `addons.Meta.Requires`; do not invent edges) |
| `optional` | Copy of `Meta.Optional` |
| `provider` | `factory`, `cli`, or `factory+cli`. Omit for import-only libraries with no Register |
| `framework_min` | Lowest kernel VERSION. Empty = unspecified. Do not fill on every official package |
| `capabilities` | Closed set: `import`, `enable`, `cli`, `heavy`. If present, must match kind/provider/heavy |

### Not in v1 (deferred, not reserved as required)

- Package semver (the Go module / git tag is the version at publish time)
- Publisher / author / license / pricing
- Marketplace listing, verification badges, revenue share
- Boot order, `Register`/`Boot` hooks, `LifecycleProvider`
- Open-ended “feature” taxonomies (`auth`, `mail`, …)

Official publisher is implied by the `github.com/zatrano/*` import path until community publishing exists.

## Identity

`name` is the catalog / `package:enable` id. `import` is how Go loads the code. They must stay aligned: official packages are `github.com/zatrano/packages/<name>` except `console` and heavy modules.

SQL drivers under `database/driver/*` and `bootutil` are not catalog packages.

## Prototype

Do not require a JSON file in every package directory. Derive a v1 document from the CLI catalog (plus Register hints when the packages tree is present) and `Validate` it. Fixtures under `testdata/` cover service, library, heavy, CLI-only library, intelligence library, and Requires.

## Compatibility

| Change | Policy |
|--------|--------|
| New optional field | Allowed in v1; old validators ignore it |
| New required field | New schema version |
| Rename/remove field | New schema version |
| New `kind` / `layer` / `capability` | New schema version or a documented v1 additive enum with a validator bump |
