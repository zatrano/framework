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
| MaxHeaderBytes | 1 MiB |
| MaxRequestBodySize | 32 MiB |
| HeaderReceived | body-limit hook (GET/HEAD skip route lookup) |
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

Gin and Echo use `net/http` on one connection. They stay informational beside `rawhttp` and fasthttp `ServeConn`.

## Fiber rows

Fiber tier-0 and tier-1 use `fasthttp.Server.ServeConn`. Two rows are published, both on the faithful connection:

| Row | Benchmarks | Timeouts |
|---|---|---|
| Default | `BenchmarkCanonT0Fiber`, `BenchmarkCanonT1FiberEcho`, `BenchmarkCanonT1FiberGen` | none |
| Equal | `BenchmarkCanonT0FiberEq`, `BenchmarkCanonT1FiberEqEcho`, `BenchmarkCanonT1FiberEqGen` | Read 60s, Write 60s, Idle 120s |

The equal-timeout row is the comparison row. fasthttp has no `ReadHeaderTimeout`. Header time sits inside `ReadTimeout` (60s). ZATRANO keeps a separate 10s `ReadHeaderTimeout`. The free-deadline Fiber rows (`BenchmarkT0Fiber`, `BenchmarkFreeT0FiberEq`, and the tier-1 pair) are diagnostic.

Fiber tier-1 sets the same security and CORS headers, recovers panics, and either reflects `X-Request-ID` or generates 16 random bytes as hex.

## Where the Run shape spends the time

On the free-deadline driver, frozen tier-0 with every timeout off is about 1.1 µs faster than the frozen Run shape, and the allocation counts differ (6 allocs/op and 704 B/op versus 10 allocs/op and about 1200 B/op). A single-factor matrix pins that on `HeaderReceived`.

`HeaderReceived` runs on every request, including GET `/plaintext`. With no `Content-Type`, `HeaderBodyLimit` takes the server ceiling and calls `env.Get` for `MAX_BODY_BYTES` and `MAX_UPLOAD_BYTES`. That is the extra allocations and the extra microsecond. The production `decideBodyLimit` (measured from `core/kernel`, because the method is unexported) lands on the same numbers, within one allocation. The harness copy is not serving a different request.

The other Run fields do not explain it:

| Factor | Allocations versus the old server |
|---|---|
| ReadHeader, Read, Write, Idle, or all four | same 704 B/op, 6 allocs/op. A small ns delta, in the noise on this machine |
| `MaxHeaderBytes` or `ReadBufferSize` 1 MiB | +1 B/op. One 1 MiB buffer per connection, not pooled (`defaultBufSize` is 8 KiB), amortized across `b.N` |
| `KeepHijackedConns`, explicit `Concurrency` 262144, `AllowUpgrade` false, `MaxRequestBodySize` 32 MiB | same 704 B/op, 6 allocs/op. Production leaves `Concurrency` at 0 |

`bench/shape` and `bench/dl` are bare rawhttp checks (no framework). Linux reference points, for a free in-memory connection: Run minus RunNoTimeouts about +210 ns; `time.Now` 57 ns; `SetDeadline` about 185 ns. This Windows laptop does not reproduce those constants. Publish the machine's own medians. The shape file's `-count=10` is sequential inside one process, so the last benchmark runs hottest; the rotating matrix above is the attribution.

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

## How to measure

One `go test -c` binary per variant. Rotate the binaries (`A B C A B C`), at least 10 samples each, 3 seconds between samples. Report median, min, and max. A single-process run warms the CPU for whichever variant starts last and is not a published number.

Record the active power plan. Balanced (`381b4222-f694-41f0-9685-ff5bb260df2e`, Turkish name Dengeli) stretches one binary across a wide min/max. High performance did not close that band on this laptop either.

```
go test -c -o canon.exe .
canon.exe -test.bench ^BenchmarkCanonT0$ -test.run ^$ -test.benchtime 1s -test.count 1
```

`cmd/tcp` is a separate real-TCP informational probe (`go run ./cmd/tcp`). Its log `cmd/tcp/results.txt` is gitignored.
