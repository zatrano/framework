# TCP benchmark harness

Separate module (`github.com/zatrano/framework/v3/bench`). The `replace` directive stays here and must not move into the framework `go.mod`. Run with `GOWORK=off` so a parent `go.work` cannot pin a local rawhttp.

Rival framework names appear only in this module.

## Tiers

Tier-0 is the router alone: one frozen `GET /plaintext` that writes `Hello, World!`, no kernel middleware.

Tier-1 is `Application.Bootstrap` plus `Application.Handle` (request id, recover, security headers, CORS, trim). The plaintext route is the same string.

## Published server shape

`BenchmarkCanonT0`, `BenchmarkCanonT1`, and `BenchmarkCanonT1Echo` are the published ZATRANO rows. They use the deadline-faithful connection.

The router is frozen the way `Bootstrap` freezes it. The `rawhttp.Server` matches `Application.Run` with timeouts and `TRUSTED_PROXIES` unset:

| Field | Value |
|---|---|
| ReadHeaderTimeout | 10s |
| ReadTimeout / WriteTimeout | 60s |
| IdleTimeout | 120s |
| MaxHeaderBytes | 16 KiB (v3.0.0–v3.0.1 was 1 MiB) |
| MaxRequestBodySize | 32 MiB |
| HeaderReceived | production hook; bodyless requests allocate nothing |
| AllowUpgrade | false |
| ConnState, Concurrency, buffer sizes | unset (carrier defaults) |
| KeepHijackedConns | true |

`bench/serverconfig.golden` is that shape plus the header-time body cap for GET, HEAD, POST, OPTIONS, and a `BodyLimit` route. `core/kernel` compares `httpServer` and `decideBodyLimit` to the file. This module compares `serverRunHead` and `ceiling` to the same file. A drift fails the test.

`BenchmarkCanonT1` omits `X-Request-ID` so the server generates a 32-hex id. `BenchmarkCanonT1Echo` sends a valid id and the server reflects it.

## Drivers

Each benchmark warms up one request outside the timer, then pipelines `b.N` requests on one connection (`ServeConn`). There is no accept loop and no per-request dial.

The published driver is `faithConn`. Reads and writes stay in memory. `SetReadDeadline`, `SetWriteDeadline`, and `SetDeadline` are forwarded to one idle loopback TCP connection, so the runtime timer cost is real. Nothing is read from or written to that socket.

`BenchmarkFreeCanonT0`, `BenchmarkFreeCanonT1`, and `BenchmarkFreeCanonT1Echo` are the same servers on `memConn`, whose deadline methods return without arming a timer. That table is diagnostic. The published table is the faithful one.

`BenchmarkT0Old`, `BenchmarkT0Run`, `BenchmarkT0RunV301`, `BenchmarkT1OldEcho`, `BenchmarkT1RunEcho`, and `BenchmarkT1RunGen` keep the unlimited-timeout server or the unfrozen router. They are not published numbers.

Gin and Echo use `net/http` on one connection, with the same deadlines as production ZATRANO: read-header 10s, read 60s, write 60s, idle 120s. They stay informational beside `rawhttp` and fasthttp `ServeConn`.

## Fiber rows

Fiber tier-0 and tier-1 use `fasthttp.Server.ServeConn`. Two rows are published, both on the faithful connection:

| Row | Benchmarks | Timeouts |
|---|---|---|
| Default | `BenchmarkCanonT0Fiber`, `BenchmarkCanonT1FiberEcho`, `BenchmarkCanonT1FiberGen` | none |
| Equal | `BenchmarkCanonT0FiberEq`, `BenchmarkCanonT1FiberEqEcho`, `BenchmarkCanonT1FiberEqGen` | Read 60s, Write 60s, Idle 120s |

The equal-timeout row is the comparison row. fasthttp has no `ReadHeaderTimeout`. Header time sits inside `ReadTimeout` (60s). ZATRANO keeps a separate 10s `ReadHeaderTimeout`. The free-deadline Fiber rows (`BenchmarkT0Fiber`, `BenchmarkFreeT0FiberEq`, and the tier-1 pair) are diagnostic.

Fiber tier-1 sets the same security and CORS headers, recovers panics, and either reflects `X-Request-ID` or generates 16 random bytes as hex.

## Where the Run shape spends the time

The published Run row uses `Application.HeaderBodyConfig`. A bodyless request returns the carrier default and does not allocate. `headFastPath` / `ceiling` stay as the diagnostic copy (`BenchmarkAblateHook` only).

`GateRunVsOld` (1.10) requires Run-shaped tier-0 to stay within 1.10× the unlimited-timeout shape, with the same allocs/op. B/op matches that shape aside from the amortized +1 of `MaxHeaderBytes` (one 16 KiB buffer per connection; v3.0.0–v3.0.1 allocated 1 MiB). The hook itself is 0 B and 0 allocs on a bodyless GET, inside about 100 ns. `TestGateDesign` only logs the gate. Phase 6 enforces it on Linux CI. Windows numbers are informational.

On the free-deadline driver, before this hook, frozen tier-0 with every timeout off was about 1.1 µs faster than the frozen Run shape, and the allocation counts differed (6 allocs/op and 704 B/op versus 10 allocs/op and about 1200 B/op). That gap was `HeaderReceived` calling `env.Get` on every GET. It is not the comparison row anymore.

`bench/shape` and `bench/dl` are bare rawhttp checks (no framework). Linux reference points, for a free in-memory connection: Run minus RunNoTimeouts about +210 ns; `time.Now` 57 ns; `SetDeadline` about 185 ns. This Windows laptop does not reproduce those constants. The shape file's `-count=10` is sequential inside one process; the rotating matrix is the attribution.

## Targets

Measured on the published rows (frozen router, Run-shaped server, faithful connection):

| Gate | Rule |
|---|---|
| H1 | tier-1 ≤ 0.90 × Fiber, and allocations ≤ Fiber |
| H2 | framework tax ≤ 1.0 µs, allocations ≤ 6 |
| H3 | tier-0 ≤ 1.00 × Fiber, reported against both Fiber rows. The equal-timeout row is the comparison |
| Regression | tier-0 and tier-1 must not be more than 5% slower than the previous tag |

Windows numbers are informational. Phase 6 enforces the gates on Linux CI. `TestGateDesign` only logs them.

The frozen tier-0 target of at most v3.0.1 × 1.01 was not shown. On the High performance plan, after the router fix (`e09fa4b`), the medians were 1242 ns (HEAD) and 1133 ns (v3.0.1), ratio 1.096, and the ranges overlapped. The router fix stays. That ratio is noise on this machine, not a demonstrated pass and not a demonstrated regression.

## Snapshot (2026-10-12)

Go **1.26.9**, Windows/amd64, CPU **i5-6200U @ 2.30GHz** (2 cores / 4 threads), `GOMAXPROCS=4`, power plan **Dengeli** (`381b4222-f694-41f0-9685-ff5bb260df2e`). One `go test -c` binary. Eight rounds, the six published rows rotated, 2 seconds between samples, `-benchtime 1s`. The bench module requires rawhttp **v0.2.6**, the same release as the framework module. The rows below are the 2026-10-12 snapshot.

| Row | median ns/op | min | max | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|
| `BenchmarkCanonT0` | 1346 | 1259 | 1388 | 960 | 2 |
| `BenchmarkCanonT0FiberEq` | 2080 | 1928 | 2235 | 0 | 0 |
| `BenchmarkCanonT1` | 3137 | 3067 | 4363 | 1024 | 5 |
| `BenchmarkCanonT1Echo` | 2666 | 2581 | 3704 | 992 | 4 |
| `BenchmarkCanonT1FiberEqEcho` | 3851 | 3750 | 5022 | 0 | 0 |
| `BenchmarkCanonT1FiberEqGen` | 3410 | 3256 | 4100 | 32 | 1 |

Median ratios on this host: tier-0 / Fiber equal-timeout **0.65×** (1346 / 2080). Tier-1 echo / Fiber equal-timeout echo **0.69×** (2666 / 3851). Tier-1 generate / Fiber equal-timeout generate **0.92×** (3137 / 3410). Tier-1 echo minus tier-0 **1320 ns**. H3 holds. The echo row is under the 0.90× time bar; its allocation count is 4 against Fiber's 0, so the allocation half of H1 does not. H2's 1.0 µs tax does not. Windows stays informational; the Linux gates are unchanged.

## How to measure

One `go test -c` binary per variant. Rotate the binaries (`A B C A B C`), at least 10 samples each, 3 seconds between samples. Report median, min, and max. A single-process run warms the CPU for whichever variant starts last and is not a published number.

Record the active power plan. Balanced (`381b4222-f694-41f0-9685-ff5bb260df2e`, Turkish name Dengeli) stretches one binary across a wide min/max. High performance did not close that band on this laptop either.

```
go test -c -o canon.exe .
canon.exe -test.bench ^BenchmarkCanonT0$ -test.run ^$ -test.benchtime 1s -test.count 1
```

`cmd/tcp` is a separate real-TCP informational probe (`go run ./cmd/tcp`). Its log `cmd/tcp/results.txt` is gitignored.

## Reproduce on your own Linux machine

One command, from the repository root, on Ubuntu 22.04 or newer with at least 4 logical cores:

```bash
./bench/run_dedicated.sh
```

The script checks the CPU governor and leaves if it is not `performance`. Set it with `sudo cpupower frequency-set -g performance`, or pass `--allow-any-governor` to continue. That flag writes the result files and does not edit the README. Go 1.26.6 is installed from the official archive when it is missing. `wrk` is installed with `apt` when it is missing.

The full run is the CI matrix: Go client and wrk, pooling off and `HTTP_POOL_REQUESTS=true`, Fiber (tuned) as the binding column, 64/256/1024/4096 connections, four routes, 20 rounds, three runs. It can take hours. `--quick` is 5 rounds and one run. It only writes the result files.

Output is `bench/results/dedicated/<YYYY-MM-DD>/` (`jsonl`, `env.txt`, `SUMMARY.md`). A full run on the performance governor rewrites the note between the `bench-env` markers in the root README and commits that file locally. It does not push.
