# Phase 11 — Runtime & Application Lifecycle Hardening

**Status:** Complete
**Prerequisite:** Phase 10 complete (`v2.0.27` baseline plus Phase 10 hardening)
**Acceptance:** ACCEPTED
**Implementation:** COMPLETE

This SPEC is an evidence-backed hardening plan. It does not invent APIs that are absent from the repository. It does not reopen Phase 8–10 acquisition architecture.

Implementation A–L is complete.

**How to read this file:** §0 decisions and §7 increment contracts are the accepted Phase 11 contract and are unchanged. §§2–6 (except subsections labeled **implemented**) record the **historical pre-implementation baseline** used to write the SPEC. §3.2, §5.1, and §6.1 record the **final implemented** Phase 11 behavior.

---

## 0. SPEC decisions (must be accepted with the SPEC)

These four decisions close the open points that blocked calling the previous draft “accepted”. They are part of the SPEC, not implementation notes.

### Decision C — Startup failure: Stop errors are observable

Historical pre-implementation code (`kernel/application.go` `Start`):

```go
_ = app.stopLifecycle(ctx, started)
return err // the Start error
```

**Decision:** Keep the existing Start-failure semantics (prior successful `LifecycleProvider`s are stopped; the failed provider is not Stop’d by the kernel; state returns to Booted; retry is allowed). Do **not** introduce a lifecycle manager or Register() rollback.

**Reporting:** If `stopLifecycle` returns a non-nil error, `Start` MUST NOT discard it. Return `errors.Join(startErr, stopErr)` (standard library). If cleanup succeeds, return `startErr` alone.

Callers classify the outcome as **startup failure**. Cleanup failure is an additional wrapped error, inspectable with `errors.Is` / `errors.As` / `errors.Unwrap`, not a second state machine.

Bootstrap `Register`/`Boot` failure stays terminal (`lifeBootFailed`) and does **not** call `Stop` (those providers never `Start`ed).

### Decision E — Context on Bootstrap and Start

**Decision:** Use the existing `context.Context` type only. Do not create a ZATRANO context abstraction.

**Compatibility (required):**

```text
contracts.App.Bootstrap() error   // remains; means BootstrapContext(context.Background())
contracts.App.Start() error       // remains; means StartContext(context.Background())
contracts.App.Stop(ctx) error     // unchanged
```

Zero-argument `Bootstrap` / `Start` stay on `contracts.App` so existing callers (`console`, tests, consumer apps) compile without changes.

**Context-bearing API:** After the contracts API compatibility review in increment E, add:

```text
BootstrapContext(ctx context.Context) error
StartContext(ctx context.Context) error
```

to `contracts.App` (lifecycle surface, not package capabilities like `Auth()`). `kernel.Application` implements them. Architecture freeze tests for `contracts` are updated in the same increment.

**What context does and does not do:**

* Nil ctx is treated as `context.Background()` (same as `Stop`).
* Context is checked **between** providers (before each `Register`, `Boot`, and `LifecycleProvider.Start`). Cancel/timeout returns `ctx.Err()` and is not converted into a generic success.
* `Provider.Register` / `Provider.Boot` / `LifecycleProvider.Start(app)` signatures do **not** change. In-flight provider calls are not force-killed.
* On `StartContext` failure after some LPs started, cleanup uses the same ctx when not already done; if ctx is already cancelled, cleanup uses a short derived timeout (keep today’s 15s bound) so shutdown can still run. Stop errors still follow Decision C.
* Acquisition `--timeout` stays in `package:acquire` and MUST NOT be wired into `Bootstrap`/`Start`.
* CLI `serve` / `Run` may pass a process context into `StartContext` in increment E or L; that is runtime, not acquire.

### Decision H — `framework_min` is distribution-time, not a boot invariant

```text
registry.Resolve     → validates framework_min (version selection)
acquire.FromResult   → MUST NOT revalidate (Phase 7/8 freeze)
runtime App()/Bootstrap → MUST NOT validate framework_min
package:doctor       → validates imported addons vs running VERSION
```

**Decision:** `framework_min` is a **distribution-time** compatibility constraint (select a compatible release; doctor the imported set). It is **not** a runtime boot invariant of `App()` / `Bootstrap` / `Start`.

Phase 11 MUST NOT add `MeetsFrameworkMin` to `bootstrap.App` or `kernel.Application.Bootstrap`. An incompatible imported addon may still Register/Boot; `package:doctor` reports `compatibility.framework`.

Increment H: lock this split with tests; add tests that `registry.MeetsFrameworkMin` and `addons.MeetsFrameworkMin` agree on fixtures; do not merge the two copies into acquire; do not add a third copy in `console`.

### Decision L — Runtime CLI exit codes are a separate table

Acquisition codes in `console/cli_exit.go` are **acquire/registry/enablement only**:

| Code | Name | Owners |
|------|------|--------|
| 0 | `ExitSuccess` | any success |
| 1 | `ExitGeneral` | default for commands without a classifier |
| 2 | `ExitUsage` | acquire/registry/enable usage |
| 3 | `ExitResolution` | `package:resolve` / acquire resolve |
| 4 | `ExitPlanning` | acquire plan |
| 5 | `ExitAcquisition` | acquire execute |
| 6 | `ExitEnablement` | enable / install / acquire `--enable` |
| 7 | `ExitCanceled` | acquire timeout/cancel only |

**Decision:** `serve`, `Run`, and other runtime boot/shutdown failures MUST NOT use codes 2–7. Phase 11 does not reuse `ExitAcquisition` or `ExitCanceled` for HTTP/lifecycle errors.

Runtime table (CLI boundary only, not `distribution/acquire`):

| Code | Name | Meaning |
|------|------|--------|
| 0 | success | process succeeded |
| 1 | `ExitGeneral` | unclassified runtime (until a command opts in) |
| 20 | `ExitRuntimeBoot` | `Bootstrap` / `Start` / `Run` failed to become ready |
| 21 | `ExitRuntimeShutdown` | `Stop` or `server.Shutdown` failed after a running server |
| 22 | `ExitRuntimeCanceled` | runtime context cancelled |
| 23 | `ExitRuntimeTimeout` | runtime context deadline exceeded |

`cmd/zatrano` maps `*CLIError` via `CodeFromError` as today. Increment L: define the 20–23 constants, wrap `serve`/`Run` errors, test that acquire JSON and codes 2–7 are unchanged. No `serve --format=json` boot payload in Phase 11 (would be a new output contract).

---

## 1. Objective

Harden the runtime/application lifecycle **after** package acquisition and enablement:

```text
search → info → resolve → acquire → optional --enable → boot → runtime → shutdown
```

The objective is not to redesign ZATRANO. The objective is to close **measurable** gaps in boot determinism, lifecycle semantics, startup failure, shutdown, cancellation/timeout, registration integrity, Enabled ∩ Imported, `framework_min`, service vs library behavior, runtime E2E, race safety, and CLI/runtime error semantics.

---

## 2. Current Baseline (repository evidence)

ZATRANO v2 already has:

* `contracts` as the minimum stable ABI (`contracts/app.go`)
* kernel typed runtime primitives (`kernel/application.go`)
* framework independent of `github.com/zatrano/packages` (`bootstrap/register_addons.go`, `tests/architecture_test.go` `TestKernelHasZeroThirdPartyDependencies`)
* official packages in `github.com/zatrano/packages`
* `zatrano.package/v1`, registry resolution, acquisition plan/apply, DryRun, CLI acquire, explicit `--enable`
* Phase 10: recovery reporting, acquisition exit codes, JSON, timeout into `ExecuteTargets`, ecosystem E2E, CI

### Frozen (do not reopen)

* Phase 8 acquisition engine (`Execute` / `ExecuteTargets` / `Inspect` / `RecoverFiles`)
* Contract A DryRun
* Contract B CLI acquisition orchestration
* Contract C explicit `--enable`
* `func Apply` and any equivalent
* no second resolver, acquisition engine, or process abstraction
* no lockfile, automatic `go mod tidy`, implicit transaction, automatic rollback
* no framework dependency on official packages
* `contracts.App` must not grow package capabilities (`Auth()`, `Queue()`, …)
* Growing `contracts.App` for **lifecycle context** (`BootstrapContext` / `StartContext`) is allowed only in increment E per Decision E and an API compatibility review

---

## 3. Runtime Architecture Map

**Historical baseline (pre-implementation evidence).** Do not infer stages that had no code at SPEC write time. The implemented Phase 11 runtime is §3.2.

```text
CLI / process entry
  cmd/zatrano/main.go  bootForCLI → bootstrap.App() or bootstrap.FromEnv("app")
        ↓
Application creation
  bootstrap.App / bootstrap.Boot / kernel.NewApplication
        ↓
Enablement selection (process-global registry ∩ per-App subset)
  bootstrap.RegisterEnablement | WithAddons | DefaultMetas
  addons.Resolve / OrderMetas / providersFromMetas
        ↓
Provider registration (not yet Register())
  kernel.Application.RegisterProviders
        ↓
Bootstrap (synchronous; context per Decision E after increment E)
  kernel.Application.Bootstrap
    env + config + secrets + logger
    Provider.Register (slice order)
    HTTP middleware + HTTPBridge
    Provider.Boot (slice order)
    router.Freeze + config.Freeze + container.Freeze
        ↓
HTTP ready (Booted, not necessarily Running)
  ServeHTTP dispatches; Start is optional
        ↓
Start (synchronous; context per Decision E after increment E)
  LifecycleProvider.Start in provider-slice order
        ↓
Run / serving
  kernel.Application.Run → Start + http.Server + SIGINT/SIGTERM
        ↓
Stop (context)
  LifecycleProvider.Stop in reverse order of the list passed to stopLifecycle
  Run also server.Shutdown(15s)
        ↓
Stopped (no restart)
```

| Stage | Entry | Owner | Exported API | Error behavior | Context | Hidden/global state | Concurrency |
|-------|-------|-------|--------------|----------------|---------|---------------------|-------------|
| CLI boot | `cmd/zatrano/main.go` `bootForCLI` | `cmd/zatrano`, `bootstrap` | `console.New`, `cli.Run` | `os.Exit(CodeFromError)`; runtime unclassified = `ExitGeneral` until L | none into `App()` today | process cwd, `.env` | single process |
| App construction | `bootstrap.App` | `bootstrap/app.go` | `App`, `WithAddons`, `WithProviders`, `Boot` | **panics** on `addons.Resolve` / `Boot` error | none | `addons.registry`, `enablement` mutexes | process-global registry (**usage contract**, §3.1) |
| Addon discovery | `addons.Register` from package `init()` | `bootstrap/addons/registry.go` | `Register`, `Available`, `Lookup`, `Resolve`, `OrderMetas` | duplicate **name panic**; unknown/missing Requires **error** | n/a | process-global `registry` | `registryMu`; not app-isolated |
| Enablement | `RegisterEnablement` / `WithAddons` | `bootstrap/enablement.go`, `bootstrap/app.go` | `RegisterEnablement`, `WithAddons` | empty registered manifest = kernel-only; unknown WithAddons names **skipped** | n/a | process-global `enablement` | `enablementMu` |
| Config | `bootstrapLocked` | `kernel/application.go` | `Config()`, env | production secrets fail Bootstrap | none today | `.env` on disk | serialized by `transitionMu` |
| Register/Boot | `bootstrapLocked` loops `app.providers` | `kernel` + each `Provider` | `contracts.Provider` | first error → `lifeBootFailed`; **no retry**; **no rollback** of earlier `Register` | Decision E after increment E | provider side effects | serialized |
| HTTP ready | `httpReady` | `kernel/application.go` | `ServeHTTP` | 503 until Booted (incl. after Stop) | request ctx only | router freeze | `lifeMu` |
| Start | `Application.Start` | `kernel/application.go` | `Start()` | Start error stops **successfully started** LPs only; returns to **Booted**; Decision C for Stop errors | Decision E after increment E; today failure cleanup 15s | provider list | `transitionMu` |
| Run | `Application.Run` | `kernel/application.go` | `Run(addr)` | Listen error then Stop; signal → `server.Shutdown` + `Stop` | 15s timeout hardcoded | OS signals | HTTP goroutine + main |
| Stop | `Application.Stop` | `kernel/application.go` | `Stop(ctx)` | no-op unless `lifeRunning`; reverse order; first Stop error kept; nil ctx → Background | **yes** | — | `transitionMu`; concurrent Stop once (`TestLifecycleConcurrentStopOnce`) |

**Historical (pre-implementation):** not present at SPEC write time: a global lifecycle manager, a second Provider interface, context on `Bootstrap`/`Start`, a boot-result type listing partial Register success, CLI JSON for boot/shutdown, runtime enforcement of `framework_min` inside `App()`.

### 3.1 Process-global registry — usage contract (not an automatic bug)

`addons.registry` and `RegisterEnablement` are **process-global by design**:

* Blank-import `init()` registers into the process, not into one `Application`.
* Two `App()` values in one process share the imported set (`registry.go` comment).
* `WithAddons` selects a **per-App** subset; it does not clone or isolate the registry.
* Tests that need isolation already call `addons.ClearRegistry` / `clearEnablement`.

Phase 11 MUST document and test this contract (including two `App()` instances and `WithAddons` isolation). Phase 11 MUST NOT introduce a per-application addon registry unless a later SPEC explicitly replaces this model.

Embedded hosts that need two isolated package sets in one process are outside this phase.

### 3.2 Implemented Phase 11 runtime (final)

This subsection is the authoritative post-A–L runtime. Decisions C, E, H, and L are implemented; they are not reopened.

Bootstrap and Start:

```text
Bootstrap()              → BootstrapContext(context.Background())
Start()                  → StartContext(context.Background())
BootstrapContext(ctx)
StartContext(ctx)
Stop(ctx)                → unchanged
```

A nil `ctx` is `context.Background()`. Cancellation is checked **between** providers (before each `Register`, `Boot`, and `LifecycleProvider.Start`). In-flight provider calls are not force-killed. `Provider` / `LifecycleProvider` signatures are unchanged. Acquisition `--timeout` stays on `package:acquire` only.

**Startup failure (Decision C):** previously started `LifecycleProvider`s are stopped; the failed provider is not Stop’d; if cleanup fails, `errors.Join(startErr, stopErr)`; if cleanup succeeds, `startErr` alone; state returns to **Booted**; retry is allowed. Bootstrap `Register`/`Boot` failure remains terminal **BootFailed** and does not call `Stop`.

**Runtime E2E (increment J):** isolated consumer module, real official-package acquisition (no fake `ExecuteTargets`), enable, `bootstrap.App()`, Bootstrap, `StartContext`, capability check, `Stop`. The framework module is not mutated.

**CLI (Decision L):** `serve` / `Run` classify through codes 20–23. Acquisition codes 2–7 and acquire JSON are unchanged. There is no runtime `serve --format=json` contract.

Embedded hosts that need two isolated package sets in one process remain outside this phase.

---

## 4. Existing Lifecycle Contracts

### 4.1 `contracts.Provider` / `contracts.LifecycleProvider`

File: `contracts/app.go`.

* **Registration (kernel):** `RegisterProviders` appends to `app.providers` while `lifeCreated`. After bootstrap: panic (`kernel/application.go` `RegisterProviders`).
* **Package registration (addons):** blank-import `init()` → `addons.Register`. Duplicate name panics (`addons.Register`). Empty name ignored.
* **Boot:** `Provider.Register` then `Provider.Boot` during `Bootstrap`. Synchronous. HTTP is ready after successful Bootstrap **without** `Start` (`httpReady` includes `lifeBooted`).
* **Ready (HTTP):** Booted (and later states including Stopped). Created / Bootstrapping / BootFailed → 503 (`kernel/application_http_ready_test.go`).
* **Ready (process workers):** `lifeRunning` after all `LifecycleProvider.Start` return nil.
* **Shutdown:** `Stop(ctx)` only if Running. Reverse of the **provider-slice** LifecycleProviders (not only those that started — see increment D). Start-failure cleanup iterates **started** only.
* **Service vs library:** kernel has no Kind. Catalog `KindLibrary` is rejected by `package:enable` (`console/package_cmd.go` `enablePackage`). Libraries may still `addons.Register` for CLI (`package:doctor` comment). Factory-less Meta contributes a name to `enabled` but no Provider (`providersFromMetas` skips `Factory == nil`).
* **Partial boot:** Bootstrap: first `Register`/`Boot` error → `lifeBootFailed`, later providers not called, earlier Register side effects remain, retry forbidden (`TestBootstrapFailureIsTerminal`). Start: failed LP is not Stop’d by kernel; prior LPs are Stop’d; state returns to Booted; retry allowed (`TestLifecycleFailedStartThenRetry`).

### 4.2 Boot ordering

* Addon graph: `Requires` must exist (error) or be dropped (`Bootable` for DefaultMetas). `Optional` edges only if present. Kahn sort + tie-break `Order` then `Name` (`OrderMetas`, tests in `bootstrap/addons/registry_order_test.go`).
* Kernel executes **provider slice order**, not the graph by itself. Slice is built from `OrderMetas` then `WithProviders`.
* Independent packages: deterministic via `Order` then `Name`, not random.
* Default (no manifest): `DefaultMetas` = `Bootable(Available())` then `OrderMetas` — unsatisfied Requires **dropped**, not error.
* Explicit `WithAddons` / manifest: `Resolve` → missing Requires **error** → `App()` **panic**.

### 4.3 Shutdown

* Idempotent while not Running (no-op).
* Double Stop: one `Stop` on the LP (`TestLifecycleConcurrentStopOnce`).
* Stopped Start rejected (`TestKernelStartStopThenStartRejected`).
* Start-failure cleanup: 15s bounded context when the original ctx is already cancelled. **Historical (pre-C):** Stop errors were discarded. **Implemented:** Decision C — `errors.Join(startErr, stopErr)` when cleanup fails.
* `Run`: `server.Shutdown` 15s then `app.Stop` same ctx.

### 4.4 Context (historical pre-E vs implemented)

| Boundary | Historical (pre-E) | Implemented (Decision E) |
|----------|--------------------|--------------------------|
| CLI → `App()` / `FromEnv` | none | still no ctx into `App()` construction |
| `Bootstrap` / `Start` | none | `BootstrapContext` / `StartContext`; zero-arg wrappers use Background |
| `Provider.Register` / `Boot` | none | unchanged signatures |
| `LifecycleProvider.Start` | none | unchanged; kernel checks ctx between LPs |
| `Stop` / `LifecycleProvider.Stop` | yes | unchanged |
| Acquisition CLI `--timeout` | yes (Phase 10) | acquire only |
| HTTP request | request ctx | unchanged |

### 4.5 Registration integrity

| Case | Actual | Evidence |
|------|--------|----------|
| Duplicate addon name | panic | `addons.Register`, `TestRegisterDuplicatePanics` |
| Duplicate name in OrderMetas input | error | `OrderMetas` |
| Duplicate container Instance before freeze | last write wins | `container.Instance` overwrites |
| Bind after freeze | panic | `TestContainerFrozenAfterBootstrap` |
| Duplicate static route | Freeze error | `TestDuplicateStaticRouteFreezeErrors` |
| Duplicate route name | Freeze error | `TestDuplicateRouteNameFreezeErrors` |
| Second Bootstrap | no-op | `TestBootstrapIsIdempotent` |
| RegisterEnablement twice | last list wins | `RegisterEnablement` replaces under mutex |

### 4.6 Enabled ∩ Imported

`bootstrap.App` comments and `enablement_test.go`:

| Imported | Enabled | Result |
|----------|---------|--------|
| false | false | not booted (`intersectImported` skip) |
| false | true | not booted (same) |
| true | false | not booted if a **registered** manifest or `WithAddons` omits it |
| true | true | booted (Factory present) |
| true | no manifest | **booted** via `DefaultMetas` (legacy G-001) |

`bootstrap.EnabledAddons` var is **not** read by `App()` (`TestAppIgnoresFrameworkEnabledAddonsVar`). Acquisition does not enable (`package:acquire` tests). Enablement does not `go get` except `wireEnabledAddon` / `ensurePackagesModule` on CLI enable — not App boot. Boot does not rewrite `bootstrap/enabled.go`. `package:install` stays enablement + stubs.

### 4.7 `framework_min`

Locked by **Decision H**. Split remains Resolve + doctor; acquire and `App()` do not validate.

### 4.8 Service vs library vs heavy

* Service: catalog KindService; `enablePackage` allowed; typically `Meta.Factory` → Provider.
* Library: `package:enable` error (import-only). May register CLI. No forced Start/Stop.
* Heavy: `Meta.Heavy` + own module path in manifest Derive. No extra kernel lifecycle.
* Shared-module: one Go module, separate addon names; boot order is per-addon graph, not module.

---

## 5. Test coverage (historical baseline)

**Historical.** This table is the runtime coverage recorded when the SPEC was written, before increments A–L.

| Area | Tests |
|------|--------|
| Enablement ∩ imported | `bootstrap/enablement_test.go` |
| Kernel without packages | `tests/package_boot_test.go` |
| OrderMetas / Expand / Bootable | `bootstrap/addons/registry_order_test.go` |
| Duplicate addon register | `bootstrap/addons/registry_test.go` |
| Bootstrap idempotent / freeze / fail terminal | `kernel/application_lifecycle_test.go` |
| Start/Stop order, concurrent, retry | same |
| HTTP 503 until Booted | `kernel/application_http_ready_test.go` |
| Duplicate routes | `kernel/routing/router_extra_test.go` |
| `framework_min` registry | `distribution/registry/resolve_test.go` |
| `framework_min` doctor | `console/package_doctor.go` + doctor tests |
| Acquire ≠ enable | `console/package_acquire_test.go` |
| Acquire+enable+Bootstrap Bound | `console/package_acquire_e2e_test.go` `TestPackageAcquireE2EEnableThenBoot` (subprocess Bootstrap only; **no** `Start`/`Stop` at SPEC write time) |
| `go test -race` | `.github/workflows/security.yml` job `race` |

### 5.1 Implemented Phase 11 tests (final)

| Area | Tests |
|------|--------|
| A Boot determinism | `bootstrap/addons/registry_order_test.go`; `bootstrap/boot_determinism_test.go` |
| B Lifecycle contract | `kernel/lifecycle_contract_test.go` |
| C Start-fail Join | `kernel/start_fail_cleanup_test.go` |
| D Shutdown alignment | `kernel/shutdown_alignment_test.go` |
| E Context APIs | `kernel/application_context_test.go` |
| F Registration / process-global registry | `bootstrap/registration_integrity_test.go`; container last-wins |
| G Enabled ∩ Imported | `bootstrap/enablement_test.go` |
| H `framework_min` agreement | `tests/framework_min_agreement_test.go` |
| I Service / library / heavy | `bootstrap/kind_lifecycle_test.go`; `console/package_kind_lifecycle_test.go` |
| J Runtime E2E | `console/package_acquire_e2e_test.go` `TestPackageAcquireE2ERuntimeLifecycle`: real acquire → enable → App → Bootstrap → `StartContext` → capability → `Stop` |
| L Runtime CLI 20–23 | `console/cli_runtime_exit_test.go`; `kernel/runtime_cli_test.go` |

---

## 6. Gap Matrix (historical, pre-implementation)

**Historical.** This matrix is the evidence-only gap list at SPEC write time. It is not the current runtime. See §6.1.

| Area | Baseline (pre-A–L) | Evidence | Risk | Required change |
|------|------------------|----------|------|-----------------|
| A Boot determinism | Graph sort is deterministic (Order, Name). Kernel follows provider slice. DefaultMetas drops incomplete graphs. | `OrderMetas`; `registry_order_test.go`; no test that **App() provider order** is stable across N runs | Low if tests lock slice order | **Tests** locking OrderMetas + App() provider order for a fixture set. Production change only if a test finds non-determinism |
| B Lifecycle contract | States exist but are unexported; “ready” means HTTP Booted ≠ Running | `lifeState` in `application.go`; `httpReady` vs `Start` | Operators confuse Bootstrap with Start | Contract tests naming Created→Bootstrapping→Booted→Starting→Running→Stopping→Stopped / BootFailed. **Do not** add a new lifecycle type |
| C Startup Stop errors | Discarded | `_ = app.stopLifecycle` | Silent cleanup failure | **Decision C:** `errors.Join` |
| D Shutdown list | `Stop` while Running iterates all LPs; Start-fail iterates started only | `Stop` vs `Start` loops | LP that never Start’d may receive Stop | **Test** and document; align only if a contract test fails |
| D Shutdown timeout | 15s hardcoded in `Run` and Start-fail cleanup | `application.go` | Inflexible | Document; no new manager |
| E Context | Lost on Bootstrap/Start | signatures | Cannot cancel boot | **Decision E** |
| F Duplicate Instance | Replace | `container.Instance` | Two packages same Key | Test + document last-wins **before freeze** |
| G Enabled ∩ Imported | Covered for fixtures; DefaultMetas boots all imported | `enablement_test.go` | Consumer without `RegisterEnablement` boots everything imported | Document G-001; E2E consumer uses `enabled.go` init |
| G Process-global registry | Shared imported set | `registry.go` | Test isolation / embed | **§3.1 usage contract** + tests; no per-App registry |
| H `framework_min` | Not a boot check | `App()` | Confusion with Resolve | **Decision H:** distribution-time only |
| H Dual MeetsFrameworkMin | Two copies | `registry/compat.go`, `addons/compat.go` | Drift | Agreement tests; no acquire merge |
| I Library enable | Rejected | `enablePackage` | — | Keep |
| J Runtime E2E | Acquire+enable+Bootstrap Bound | `package_acquire_e2e_test.go` | No Start/Stop | Isolated Start+Stop E2E |
| K Race | Serialized Start/Stop; CI `-race` | lifecycle tests; `security.yml` | Shared registry | Test two `App()`; no speculative locks |
| L CLI runtime errors | `serve`/`Run` → exit 1 | `CodeFromError` | Acquire code reuse | **Decision L:** codes 20–23; never 2–7 |

Gaps that are **not** work: second lifecycle manager; `func Apply`; moving packages into kernel; runtime re-resolve; automatic rollback of `Register`; per-App addon registry; `framework_min` inside `Bootstrap`.

### 6.1 Implemented closeout (final)

Increments A–L closed the historical gaps above without redesign. Remaining non-goals in §8 still hold.

| Area | Implemented |
|------|-------------|
| C Startup failure | Prior LPs stopped; failed LP not Stop’d; `errors.Join(startErr, stopErr)` when cleanup fails; Booted; retry allowed |
| E Context | `BootstrapContext` / `StartContext`; zero-arg wrappers use Background; nil ctx = Background; cancel between providers; no force-kill; acquire `--timeout` not wired into runtime |
| J Runtime E2E | Real acquire → enable → App → Bootstrap → `StartContext` → capability → `Stop` |
| L Runtime CLI | `serve` / `Run` use 20–23; acquire 2–7 and acquire JSON unchanged; no runtime JSON contract |

---

## 7. Increment contracts

Implement only after SPEC acceptance. Prefer tests that lock existing behavior. Decisions C, E, H, L override any older “or document discard” language.

### A — Boot determinism

Same imported+enabled set → same `OrderMetas` names. Repeated `OrderMetas` identical. Dependencies before dependents. Duplicate registration panics (already).

### B — Lifecycle stages (document, then test)

```text
Created → Bootstrapping → Booted → Starting → Running → Stopping → Stopped
Created → Bootstrapping → BootFailed
```

HTTP ready from Booted through Stopped. Process workers only while Running.

### C — Startup failure

Follow **Decision C**. Matrix:

| Failure | Expected |
|---------|----------|
| Provider.Register error | Bootstrap error, BootFailed, no retry, no Stop |
| Provider.Boot error | same |
| Missing Requires (explicit Resolve) | `App()` panic remains unless a later SPEC changes construction errors |
| Missing Requires (DefaultMetas) | drop via Bootable |
| Lifecycle Start error | stop prior LPs; Booted; return start error; Join stop error if cleanup failed |
| Partial Bootstrap | not success; no structured partial Register report |
| Shutdown after partial Start | prior LPs stopped; `Stop()` while Booted is no-op |

### D — Shutdown

Keep reverse order. Keep no-op if not Running. `Stop(ctx)` still returns the first Stop error. Test Running-`Stop` vs Start-fail cleanup lists.

### E — Cancellation

Follow **Decision E**. Acquisition `--timeout` unchanged.

### F — Registration

Keep panic on duplicate addon names. Keep route Freeze errors. Document container last-wins.

### G — Enabled ∩ Imported

Keep table in §4.6. Acquisition ≠ enablement. Install unchanged. Document §3.1.

### H — Compatibility

Follow **Decision H**.

### I — Service vs library

Do not force libraries through `LifecycleProvider`.

### J — Runtime E2E

Isolated module; official package; enable; boot; capability; `Start`/`StartContext`; `Stop`.

### K — Race

`go test -race ./...` stays green. Add tests only for evidenced races. Two-`App()` sharing registry is expected.

### L — CLI runtime semantics

Follow **Decision L**. Do not extend acquire JSON.

---

## 8. Non-goals

* Redesign acquisition, registry, or resolver
* `func Apply` or equivalent
* Lockfile, automatic tidy, automatic rollback
* Framework import of `github.com/zatrano/packages`
* Package methods on `contracts.App` (`Auth()`, `Queue()`, …)
* Global App service locator
* Duplicate lifecycle abstractions or a new context type
* Per-application `addons.registry` in this phase
* `framework_min` checks inside `App()` / `Bootstrap`
* Reusing acquire exit codes 2–7 for `serve` / `Run`
* Changing Phase 8/9/10 without a regression

---

## 9. Implementation order

```text
SPEC Acceptance
      ↓
A Boot determinism (tests first)
      ↓
B Lifecycle contract tests
      ↓
C Startup failure — errors.Join Stop errors (Decision C)
      ↓
D Shutdown alignment tests
      ↓
E BootstrapContext / StartContext (Decision E, contracts review)
      ↓
F Registration integrity tests
      ↓
G Enabled ∩ Imported + process-global registry contract tests
      ↓
H framework_min split tests (Decision H)
      ↓
I Service vs library vs heavy
      ↓
J Isolated runtime E2E (acquire → enable → boot → start → stop)
      ↓
K Race (only evidenced)
      ↓
L Runtime CLI codes 20–23 (Decision L)
```

Each increment: contract → tests → smallest implementation → `go test` / `go vet` / `staticcheck` (and `-race` for K).

A later increment must not silently redefine an earlier one. E must not ship before C. L must not classify acquire errors as 20–23.

---

## 10. Acceptance checklist (SPEC)

Accepted. Implementation A–L is complete:

* [x] Phase 8–10 acquisition remains frozen
* [x] Existing `Provider` / `LifecycleProvider` remain the lifecycle ABI
* [x] No new global lifecycle manager
* [x] Process-global addon registry is a usage contract, not a silent bug (§3.1)
* [x] **C:** Start-fail Stop errors are `errors.Join`’d with the Start error
* [x] **E:** `Bootstrap()` / `Start()` remain; `BootstrapContext` / `StartContext` added after API review; Provider signatures unchanged
* [x] **H:** `framework_min` stays Resolve + doctor; not `App()`/`Bootstrap`
* [x] **L:** Runtime CLI uses 20–23; never acquire 2–7
* [x] Boot order is Kahn + Order + Name; tests lock it
* [x] Bootstrap failure stays terminal; Start failure stays retryable from Booted
* [x] No automatic Register rollback
* [x] Isolated E2E uses real packages checkout and does not mutate the framework module
* [x] Framework still does not import official packages

---

## 11. Current state

```text
Phase 8–9     FROZEN / COMPLETE
Phase 10      COMPLETE
Phase 11      COMPLETE (increments A–L)
SPEC          ACCEPTED
Implementation COMPLETE
```

Local `go test ./...`, `go vet ./...`, and `staticcheck ./...` passed after A–L.
`go test -race ./...` requires cgo; this Windows host has no C compiler. CI continues to run the race job (`.github/workflows/security.yml`). No speculative locks were added.
