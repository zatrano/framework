# Phase 8 — Module Acquisition Apply

**Status:** Draft — specification only  
**Previous phases:** 1–7 frozen  
**Phase 7 boundary:** `GoGetArg`  
**This phase:** Apply contract only  
**Implementation:** not authorized

```
Resolve → FromResult → Plan → Targets → GoGetArg
                                         ✕ Phase 7
                                         │
                                         ▼
                                       Apply
                                         ↓
                                       go get
                                         ↓
                                   go.mod / go.sum
```

Nothing below Apply is Phase 7. Nothing above `GoGetArg` is redesigned here. `package:install` remains enablement.

## 1. Purpose

Phase 8 defines how a resolved acquisition plan may be applied to a Go application's module graph.

Apply converts an already-resolved plan into **controlled Go module mutations**. It MUST NOT become another resolver.

It MUST NOT change: package resolution, manifest, registry, CLI resolution, enablement, framework boot, or Enabled ∩ Imported.

## 2. Non-goals

Phase 8 MUST NOT introduce: a new resolver or version selector; a ZATRANO lockfile; package-level semver beside Go modules; marketplace / registry HTTP / publisher / licensing; package boot or enablement; automatic blank-imports; automatic edits of `bootstrap/enabled.go` or package registration; automatic framework runtime changes.

Today's `package:install` remains enablement. It MUST NOT silently become module acquisition.

## 3. Input

Apply consumes Phase 7 output (`Plan`, `Targets`, `GoGetArg`). It MUST receive concrete `module@version` strings. It MUST NOT receive `latest`, version ranges, `framework_min`, or package kind as a resolution instruction and then interpret them.

Acceptable targets are of the form:

```
github.com/zatrano/packages@main
github.com/zatrano/packages@v1.4.0
github.com/zatrano/packages/mongo@main
github.com/zatrano/framework/v2@main
```

## 4. Resolver boundary

Apply treats Phase 7 as authoritative. It MUST NOT call `Resolve`, `Search`, `Lookup`, `compareSemver`, or `latestCompatible` to decide what to acquire.

```
Registry → Resolve → Phase 7 Plan → Phase 8 Apply
```

never `Apply → Resolve again`. Resolution happens **exactly once** before acquisition.

## 5. Mutation boundary

Apply is the first layer allowed to mutate the application's Go module state. Permitted surface: `go.mod` and `go.sum` **as produced by Go's module tooling**.

Apply MUST NOT edit those files as arbitrary text. Preferred mechanism: `go get <concrete-module>@<concrete-version>` in the target application module directory. Apply MUST NOT ship its own Go module parser/editor unless a later SPEC authorizes it.

## 6. `go get` invocation

Each target has a concrete `GoGetArg`. Apply MAY run `go get <GoGetArg>`. It MUST NOT replace a concrete version with `latest` or another symbolic selector. `module@v1.4.0` stays `module@v1.4.0` throughout Apply.

## 7. `main`

`main` is a valid target when Phase 7 produced it. Apply passes `@main` to Go as-is. It MUST NOT invent a ZATRANO pseudo-version. Go tooling writes the resulting pin into go.mod / go.sum. That rewrite is Go module state, not registry resolution.

## 8. Shared-module targets

Phase 7 may collapse `session@main` + `auth@main` into one `github.com/zatrano/packages@main`. Apply operates on **acquisition targets**, not catalog names. It MUST NOT `go get` the same module twice because two packages share it.

## 9. Conflicting pins

Phase 7 already rejects two pins for one module. If a conflict reaches Apply, it is invalid input: **no mutation**. Apply MUST NOT invent last-write-wins, first-write-wins, highest-version-wins, or main-wins.

## 10. Atomicity

Planning (`Plan` → `Targets` → `GoGetArg`) is pure. Mutation (`GoGetArg` → Go tooling → go.mod/go.sum) is not. A failed mutation MUST NOT be reported as a successful acquisition. The result must make the mutation outcome explicit. A public `ApplyResult` type is implementation work and MUST NOT be invented until implementation is authorized.

## 11. go.mod failure

If `go get` fails before successfully changing module state: Apply → error. Surface command failure, exit status, relevant stderr, and the target module/version. MUST NOT hide Go errors behind only “installation failed”.

## 12. go.sum failure

go.sum is Go's verification state. Apply MUST NOT fabricate or repair checksums. Proxy, sumdb, auth, network, or verification failures: Apply → error, preserving the underlying reason.

## 13. `go mod tidy`

Apply MUST NOT treat `go mod tidy` as pinning or resolution. Default Phase 8: `go get` → inspect result → return. **No automatic tidy.** Whether tidy is appropriate after a particular mutation is a separate policy, not implicit in this contract.

## 14. Concurrency

At most one Apply may mutate a given application's go.mod / go.sum at a time. Concurrent Apply against **different** module roots MAY proceed independently. The serialization mechanism is an implementation detail; the contract is exclusive mutation per module root.

## 15. Partial apply

Multiple targets can yield A success, B success, C failure, D unattempted. Apply MUST NOT claim “all targets acquired”. Successful, failed, and unattempted targets MUST be observable if the implementation applies incrementally. Exact result shape is implementation work.

## 16. Rollback

Rollback MUST NOT be assumed transactional. Distinguish **rollback guaranteed** from **rollback unavailable / recovery required**. If rollback is not guaranteed, the API MUST report partial mutation explicitly. Best-effort cleanup MUST NOT be presented as transactional rollback.

## 17. Failure strategy

Initial contract: **fail-fast**. Stop on the first mutation failure. MUST NOT continue and report success. If earlier targets already mutated the graph, the result MUST expose that.

## 18. Filesystem scope

Apply operates only in the explicitly supplied application/module root. It MUST NOT scan parent trees, mutate unrelated repos, the framework tree, the packages tree, global Go config, the module cache, or shell config. Go tooling may use its normal cache; Apply MUST NOT write those locations itself.

## 19. Enablement

Apply MUST NOT modify `bootstrap/enabled.go`, addon registration, or equivalent enablement state.

```
Acquire module  ≠  Enable package
```

A successful Apply does not enable a package. Enabling a package does not mean Phase 8 ran.

## 20. `package:install`

Existing `package:install` stays enablement for the whole of Phase 8. No implementation may silently redefine it. Connecting acquisition and enablement needs a **separate** approved contract.

## 21. Dry run

A future implementation SHOULD support a non-mutating dry-run that consumes the **same** frozen Phase 7 plan. Dry-run MUST NOT introduce a second resolver.

## 22. Idempotency

Re-applying an already-present concrete module SHOULD be safe, via Go module semantics. Apply MUST NOT create `zatrano.lock`. Authoritative state remains go.mod + go.sum.

## 23. Integrity

Manifest digest ≠ Go module zip/checksum. Digest MUST NOT be passed to `go get` as a module hash. Go verification stays with Go.

## 24. Security

`GoGetArg` is structured data from the registry/manifest pipeline. MUST NOT build a shell string (`sh -c "go get " + arg`). Preferred: `exec.Command("go", "get", concreteGoGetArg)`. No shell interpolation. How the `go` binary is resolved is an implementation contract.

## 25. Timeout / cancellation

Apply MUST accept `context.Context`. Cancellation MUST reach the `go` process. MUST NOT leave an orphaned `go` process where the OS allows process cancellation. Timeouts belong to the caller, not a hard-coded Apply default.

## 26. Observability

Expose: module root, target module, requested version, command, exit status, stdout/stderr, mutation result. MUST NOT log credentials, tokens, or unrelated environment.

## 27. Architecture invariants

1. CLI does not resolve packages.  
2. Apply does not resolve packages.  
3. `FromResult` does not resolve packages.  
4. Plan holds concrete acquisition targets.  
5. `GoGetArg` contains no `latest`.  
6. `Targets` does not select pins.  
7. Conflicting pins are errors.  
8. Apply does not modify enablement.  
9. Apply does not create a ZATRANO lockfile.  
10. Apply does not implement its own Go module resolver.  
11. Apply does not automatically run `go mod tidy`.  
12. Phase 7 remains filesystem-free.

## 28. Implementation gate

**No implementation until this SPEC is accepted.** After acceptance, order:

1. Apply contract tests  
2. Process invocation boundary  
3. `go get` execution  
4. go.mod / go.sum inspection  
5. Concurrency enforcement  
6. Partial-apply reporting  
7. Rollback / recovery behavior  
8. Integration tests  

The first implementation MUST NOT modify `package:install`, `Resolve`, `FromResult`, `Plan`, `Targets`, or `GoGetArg`.

## 29. Completion

Phase 8 is complete only when: Apply consumes frozen Phase 7 output; no second resolver; concrete args reach Go tooling; `latest` cannot reach Apply; conflicting pins cannot merge silently; mutation is scoped to the intended app module; go.mod/go.sum changes are from Go tooling; tidy is not treated as pinning; concurrent mutation of one module is serialized; partial apply is observable; rollback is documented as guaranteed **or** unavailable; enablement stays independent; `package:install` unchanged; no `zatrano.lock`; Phase 7 stays pure and filesystem-free.
