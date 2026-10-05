# TCP benchmark harness

Separate module (`github.com/zatrano/framework/v3/bench`). The `replace` directive stays here and must not move into the framework `go.mod`. Run with `GOWORK=off` so a parent `go.work` cannot pin a local rawhttp.

Rival framework names appear only in this module.

## Tiers

Tier-0 is the router alone: one frozen `GET /plaintext` that writes `Hello, World!`, no kernel middleware.

Tier-1 is `Application.Bootstrap` plus `Application.Handle` (request id, recover, security headers, CORS, trim). The plaintext route is the same string.

## Published server shape

`BenchmarkCanonT0`, `BenchmarkCanonT1`, and `BenchmarkCanonT1Echo` are the published ZATRANO rows.

The router is frozen the way `Bootstrap` freezes it. The `rawhttp.Server` matches `Application.Run` with timeouts and `TRUSTED_PROXIES` unset:

| Field | Value |
|---|---|
| ReadHeaderTimeout | 10s |
| ReadTimeout / WriteTimeout | 60s |
| IdleTimeout | 120s |
| MaxHeaderBytes | 1 MiB |
| MaxRequestBodySize | 32 MiB |
| HeaderReceived | body-limit hook (GET/HEAD skip route lookup) |
| AllowUpgrade | false |
| ConnState, Concurrency, buffer sizes | unset (carrier defaults) |
| KeepHijackedConns | true |

`bench/serverconfig.golden` is that shape plus the header-time body cap for GET, HEAD, POST, OPTIONS, and a `BodyLimit` route. `core/kernel` compares `httpServer` and `decideBodyLimit` to the file. This module compares `serverRunHead` and `ceiling` to the same file. A drift fails the test.

`BenchmarkCanonT1` omits `X-Request-ID` so the server generates a 32-hex id. `BenchmarkCanonT1Echo` sends a valid id and the server reflects it.

## Diagnostic rows

`BenchmarkT0Old`, `BenchmarkT0Run`, `BenchmarkT0RunV301`, `BenchmarkT1OldEcho`, `BenchmarkT1RunEcho`, and `BenchmarkT1RunGen` are not published numbers. They keep the unlimited-timeout server (`Read`/`Write`/`Idle` = -1) and the unfrozen router so a regression can be separated from the production shape. Gin and Echo rows use `net/http` on one connection; they are informational next to `rawhttp` / fasthttp `ServeConn`.

## Drivers

Each benchmark warms up one request outside the timer, then pipelines `b.N` requests on a single in-memory connection (`ServeConn`). There is no accept loop and no per-request dial.

Fiber tier-0 and tier-1 use `fasthttp.Server.ServeConn`. Fiber tier-1 sets the same security and CORS headers, recovers panics, and either reflects `X-Request-ID` or generates 16 random bytes as hex.

## Targets

Measured on the published (frozen, Run-shaped) rows:

| Gate | Rule |
|---|---|
| H1 | tier-1 ≤ 0.90 × Fiber, and allocations ≤ Fiber |
| H2 | framework tax ≤ 1.0 µs, allocations ≤ 6 |
| H3 | tier-0 ≤ 1.00 × Fiber |
| Regression | tier-0 and tier-1 must not be more than 5% slower than the previous tag |

`TestGateDesign` logs H3 and the regression gate. It does not fail the build. Phase 6 turns the gate into a Linux CI check.

## How to measure

One `go test -c` binary per variant. Rotate the binaries (`A B C A B C`), at least 10 samples each, 3 seconds between samples. Report median, min, and max. A single-process run warms the CPU for whichever variant starts last and is not a published number.

Windows: published samples need the High performance power plan (`powercfg /setactive 8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c`). Balanced (`381b4222-f694-41f0-9685-ff5bb260df2e`, Turkish name Dengeli) stretches the same binary across a wide min/max. Record the active plan in the report and restore the previous plan when finished.

```
go test -c -o canon.exe .
canon.exe -test.bench ^BenchmarkCanonT0$ -test.run ^$ -test.benchtime 1s -test.count 1
```

`cmd/tcp` is a separate real-TCP informational probe (`go run ./cmd/tcp`). Its log `cmd/tcp/results.txt` is gitignored.
