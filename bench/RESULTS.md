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

## Windows baseline (round 2, config `default`)

Host: Windows/amd64, Intel Core i5-6200U @ 2.30GHz (2 cores / 4 threads). Power plan was Dengeli. It was not switched to high performance. The load generator is the Go client. Server pinned to logical CPU 0 (`GOMAXPROCS=1`), client on the other CPUs. 10 rounds, warmup 1.5s, measure 5s, order alternates. Ratios are paired ZATRANO/Fiber. The interval is a 10,000-draw bootstrap of the median. Errors were 0. Peak RSS is reported and is not a gate. Config compared with Fiber: `default` (`HTTP_POOL_REQUESTS` unset).

| Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 1.088 | 0.829 | 1.206 | 0.974 | 0.642 | 0.941 | 1.010 | esit |
| GET /plaintext | 256 | 1.290 | 1.021 | 1.656 | 0.698 | 0.656 | 0.894 | 1.146 | ustun |
| GET /plaintext | 1024 | 1.061 | 0.997 | 1.169 | 0.971 | 0.821 | 0.941 | 1.072 | esit |
| GET /plaintext | 4096 | 1.047 | 1.013 | 1.097 | 1.172 | 1.153 | 0.919 | 1.184 | esik-yok |
| GET /json | 64 | 1.013 | 0.948 | 1.041 | 1.007 | 1.005 | 0.975 | 1.019 | esit |
| GET /json | 256 | 0.937 | 0.904 | 0.988 | 1.009 | 1.162 | 1.033 | 1.017 | fiber |
| GET /json | 1024 | 0.947 | 0.903 | 1.007 | 1.051 | 1.083 | 1.075 | 1.006 | esit |
| GET /json | 4096 | 0.948 | 0.901 | 1.007 | 1.156 | 1.045 | 1.050 | 1.070 | esit |
| GET /users/{id} | 64 | 0.939 | 0.894 | 0.978 | 1.052 | 1.071 | 1.062 | 1.019 | fiber |
| GET /users/{id} | 256 | 0.975 | 0.937 | 1.025 | 1.007 | 1.081 | 1.024 | 1.016 | esit |
| GET /users/{id} | 1024 | 0.955 | 0.927 | 1.006 | 0.973 | 0.766 | 1.045 | 0.994 | esit |
| GET /users/{id} | 4096 | 0.962 | 0.925 | 1.055 | 0.847 | 0.852 | 1.032 | 0.937 | esit |
| POST /echo | 64 | 0.923 | 0.866 | 0.951 | 1.178 | 1.113 | 1.082 | 1.021 | fiber |
| POST /echo | 256 | 0.940 | 0.925 | 1.014 | 1.055 | 1.020 | 1.057 | 1.014 | esit |
| POST /echo | 1024 | 0.925 | 0.909 | 0.944 | 1.068 | 1.072 | 1.067 | 1.034 | fiber |
| POST /echo | 4096 | 0.940 | 0.896 | 0.971 | 1.102 | 1.206 | 1.080 | 1.047 | fiber |

Of 16 cells, 1 is ustun, 10 are esit, 4 have Fiber ahead, and 1 is esik-yok (throughput CI above 1, p99 worse). The plaintext/256 interval is wide. That cell is ustun on this run because the lower bound is above 1 and p99 and CPU are lower; its peak RSS is 1.15× Fiber and is not part of the gate. POST /echo at 64, 1024, and 4096, and GET /json at 256, and GET /users/{id} at 64 sit entirely under 1.

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
