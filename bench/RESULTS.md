# ZATRANO vs Fiber v3.6.0

## Round 2 gate

A cell is a paired comparison. Each round runs ZATRANO and Fiber back to back, and the round order alternates. The throughput ratio is ZATRANO/Fiber for that round. The cell ratio is the median of those pairs. The 95% interval is a 10,000-draw bootstrap of that median. Default cell size is 10 rounds. Peak RSS is reported and is not a gate.

| Verdict | Rule |
|---|---|
| ustun | CI lower bound > 1, p99 median ≤ Fiber, CPU µs/request median ≤ Fiber |
| esit | The CI contains 1.00. This is not a lead. |
| fiber | CI upper bound < 1 |
| esik-yok | Throughput CI is above 1, and p99 or CPU is worse than Fiber |

Linux numbers, when present, are paired ratios only. Absolute requests/sec from a shared runner are not a result. Windows numbers stay in their own section. `go test -race ./...` runs on the Linux `tests` workflow (ubuntu-24.04). This host has no gcc and no WSL, so the race binary is that job.

## Round 1 record (4-round medians, no interval)

Measured on this host, not on the 1 vCPU sandbox from earlier notes. Windows/amd64, Intel Core i5-6200U @ 2.30GHz (2 cores / 4 threads), power plan Dengeli. The harness pins the server to logical CPU 0 (`GOMAXPROCS=1`) and the client to the other CPUs. Both servers are one `ztfb` binary built with Go 1.26.0. Warmup 1.5s, measure 5s, 4 rounds (3 at 4096), server order alternates, figures are medians. Peak RSS is `PeakWorkingSetSize` (the Windows stand-in for `VmHWM`). Errors were 0 in every cell.

Ratios are ZATRANO / Fiber. Throughput passes at 1.10 or higher. Latency, CPU, and RSS pass at 1.00 or lower. A cell passes only when every one of those holds.

| Route | Conns | Baseline rps | Final rps | p50 | p99 | CPU/req | Peak RSS | Misses |
|---|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 0.975 | 1.135 | 0.952 | 0.675 | 0.946 | 0.993 | |
| GET /plaintext | 256 | 0.944 | 0.914 | 1.071 | 1.194 | 1.049 | 1.065 | rps, p50, p99, cpu, rss |
| GET /plaintext | 1024 | 1.101 | 1.048 | 0.935 | 1.168 | 0.954 | 1.068 | rps, p99, rss |
| GET /plaintext | 4096 | 1.102 | 0.998 | 0.883 | 2.154 | 0.974 | 1.049 | rps, p99, rss |
| GET /json | 64 | 1.045 | 0.960 | 1.046 | 1.070 | 0.987 | 1.023 | rps, p50, p99, rss |
| GET /json | 256 | 0.937 | 0.914 | 1.014 | 2.060 | 1.000 | 1.019 | rps, p50, p99, rss |
| GET /json | 1024 | 1.092 | 0.988 | 1.063 | 1.229 | 0.997 | 1.063 | rps, p50, p99, rss |
| GET /json | 4096 | 1.056 | 1.000 | 1.137 | 0.989 | 1.042 | 1.003 | rps, p50, cpu, rss |
| GET /users/{id} | 64 | 0.872 | 0.956 | 1.047 | 0.846 | 1.053 | 1.028 | rps, p50, cpu, rss |
| GET /users/{id} | 256 | 0.885 | 0.920 | 1.085 | 1.071 | 1.091 | 1.014 | rps, p50, p99, cpu, rss |
| GET /users/{id} | 1024 | 1.480 | 1.037 | 0.960 | 1.241 | 0.970 | 1.033 | rps, p99, rss |
| GET /users/{id} | 4096 | 0.982 | 0.915 | 1.035 | 1.228 | 1.060 | 0.976 | rps, p50, p99, cpu |
| POST /echo | 64 | 0.908 | 1.034 | 1.002 | 0.950 | 1.010 | 1.018 | rps, p50, cpu, rss |
| POST /echo | 256 | 0.892 | 1.025 | 0.998 | 0.921 | 1.010 | 1.017 | rps, cpu, rss |
| POST /echo | 1024 | 0.869 | 1.021 | 1.020 | 1.126 | 0.993 | 1.025 | rps, p50, p99, rss |
| POST /echo | 4096 | 0.826 | 0.843 | 1.138 | 1.005 | 1.209 | 1.082 | rps, p50, p99, cpu, rss |

`GET /plaintext` at 64 connections is the only cell that met every threshold on this run. That route does not read a body, so the two kept edits do not run there. The same cell's baseline ratio was 0.975. Round-to-round swings on this laptop are often larger than 10%, so that single pass is noise, not a stable lead.

## What the profiles showed

`POST /echo`, 64 connections, `GOMAXPROCS=1`, 5s CPU profile (4.81s of samples):

- 43% of samples are `runtime.cgocall` inside `WSASend`. `writeResponse` is 1.59s cumulative and 0.02s flat. The time is the socket send, not header formatting.
- `encoding/json` unmarshal plus marshal is the handler. Fiber uses the same `encoding/json` for this harness.
- After the first edit, an alloc profile still showed `MaxBodyBytes` calling `os.LookupEnv` from `readBody` (40MB, 176k objects). That was the second edit.
- Framework flat time with `encoding/json` hidden is 16%, and no single function is above about 2% flat. `mallocgc` is about 7%. Pooling `*Request` or `*Response` would cut that alloc, and it was not done: a handler may keep the pointer, and `Body()` returns an owned slice. Skipping the body copy was not done either: `json.RawMessage` would alias the connection buffer.
- rawhttp's keep-alive response cache requires an empty extra-header buffer. Security headers and `X-Request-Id` fill that buffer, so the cache never hits. Rebuilding it is not where the samples are. rawhttp was not changed.

The last edit copied the boot-time `MAX_BODY_BYTES` onto the request in `Handle`. On the echo remeasure, CPU µs/request at 64 connections moved from 44.0 to 41.0 (Fiber 41.1). Throughput at 64, 256, and 1024 moved from about 0.91–0.87× Fiber to about 1.03–1.02×. That is above the ±5% noise bar and still short of 1.10×. The 4096 echo cell stayed near 0.84×.

## Soak

512 connections, 25s per route, one round, errors 0 on both servers. Current RSS was sampled once a second. The first samples include the connection ramp. From five seconds in to the end:

| Route | ZATRANO | Fiber |
|---|---:|---:|
| GET /plaintext | +16 KB | +4.7 MB |
| GET /json | +741 KB | +8 KB |
| GET /users/{id} | −1.0 MB | +16 KB |
| POST /echo | −70 KB | +459 KB |

ZATRANO's steady RSS did not grow. Fiber's plaintext process did.
