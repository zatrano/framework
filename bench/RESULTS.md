# ZATRANO vs Fiber v3.6.0

These results were measured on v3.2.4 and earlier. Rows for pooled configurations (`HTTP_POOL_REQUESTS`) describe a mode that no longer exists; the setting is now ignored. The cost of the copy-on-parse request and the always-copy response is measured in [docs/PERFORMANCE-IMPACT.md](../docs/PERFORMANCE-IMPACT.md).

## Round 2 gate

A cell is a paired comparison. Each round runs ZATRANO and Fiber back to back, and the round order alternates. The throughput ratio is ZATRANO/Fiber for that round. The cell ratio is the median of those pairs. The 95% interval is a 10,000-draw bootstrap of that median. Default cell size is 10 rounds. Peak RSS is reported and is not a gate.

| Verdict | Rule |
|---|---|
| ustun | CI lower bound > 1, p99 median ≤ Fiber, CPU µs/request median ≤ Fiber |
| esit | The CI contains 1.00. This is not a lead. |
| fiber | CI upper bound < 1 |
| esik-yok | Throughput CI is above 1, and p99 or CPU is worse than Fiber |

Linux numbers, when present, are paired ratios only. Absolute requests/sec from a shared runner are not a result. Windows numbers stay in their own section. `go test -race ./...` runs on the Linux `tests` workflow (ubuntu-24.04). This host has no gcc and no WSL, so the race binary is that job.

## What `default` and `pooled` mean

| Table | `HTTP_POOL_REQUESTS` | What the process does |
|---|---|---|
| Windows round 2 `default` | unset, and that build treated unset as off | A new `Request` and `Response` on every request |
| Windows round 2 `pooled` | `true` | Reuse `Request` and the `Text`/`JSON` `Response` after the handler returns |
| Linux from the pool change onward, label `default` | unset | Pool on. Unset is the product default |
| Linux label `pooled` | `true` | The same pool |

After that change the two Linux labels are one configuration. The tables below keep the label that was recorded on the row. A later Linux report uses the `default` rows only. `HTTP_POOL_REQUESTS=false` is the off switch and is not a published table.

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

Of 16 cells, 1 is ustun, 9 are esit, 5 have Fiber ahead, and 1 is esik-yok (throughput CI above 1, p99 worse). The plaintext/256 interval is wide. That cell is ustun on this run because the lower bound is above 1 and p99 and CPU are lower; its peak RSS is 1.15× Fiber and is not part of the gate. POST /echo at 64, 1024, and 4096, and GET /json at 256, and GET /users/{id} at 64 sit entirely under 1.

## POST /echo after BindJSON (config `default`)

The echo handler now calls `BindJSON`. Fiber still decodes `c.Body()` with no copy. `Body()` is unchanged.

Microbench, 27-byte object: unmarshal in place 568 B and 10 allocs; copy then unmarshal 600 B and 11 allocs. The removed copy is one allocation and 32 bytes. A 14s CPU profile at 64 connections (8.82s of samples) is still 45% `WSASend` inside `writeResponse` (flat 0.45%, one write). `BindJSON` is 11.8% cumulative and `encoding/json.Unmarshal` is 11.6%. Alloc space: `JSON` 129MB, `NewRequest` 120MB, `reflect.mapassign` 59MB, `json.Unmarshal` 30MB. The body copy is not in that list.

| Conns | Baseline CI | After CI | Verdict | Intervals |
|---:|---|---|---|---|
| 64 | 0.866–0.951 fiber | 0.892–0.992 fiber | fiber | overlap |
| 256 | 0.925–1.014 esit | 0.871–1.072 esit | esit | overlap |
| 1024 | 0.909–0.944 fiber | 0.914–0.988 fiber | fiber | overlap |
| 4096 | 0.896–0.971 fiber | 0.946–1.021 esit | esit | overlap |

The after interval overlaps the baseline interval on every echo cell. That is not a gain.

`NewRequest` is 480 B and 1 alloc. The opt-in pool (`HTTP_POOL_REQUESTS=true`, default off) is 0 B and 0 allocs on the same microbench (531 ns to 88 ns). Debug mode panics after release and bumps the generation. The pooled configuration is a separate table. JSON bytes still match `encoding/json.Marshal`, including `<` and `&`.

## Write path and connection buffers

`writeResponse` already appends the header and the body into one buffer and calls `WSASend` once. There is no second write to merge. A deferred flush was not added: the samples are the send itself, not an extra syscall. rawhttp starts the connection buffers at 512 bytes. It drops a body or response buffer above 4096 bytes only when `ReduceMemoryUsage` is on, and that flag is off. Starting those buffers at 4KB would raise RSS. The baseline peak-RSS ratios are the measurement; plaintext at 256 and 4096 are above 1.10× and the rest of the baseline cells are not.

## Windows pooled (`HTTP_POOL_REQUESTS=true`)

Same host, same 10-round paired protocol, Go client. This is not the default. Fiber is unchanged. The pool reuses the request and the text/JSON response objects. It does not pool the JSON bytes.

| Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 1.159 | 1.119 | 1.201 | 0.846 | 0.886 | 0.901 | 0.994 | ustun |
| GET /plaintext | 256 | 1.018 | 0.974 | 1.046 | 0.998 | 1.013 | 0.954 | 0.961 | esit |
| GET /plaintext | 1024 | 1.056 | 1.002 | 1.132 | 0.981 | 0.819 | 0.893 | 0.933 | ustun |
| GET /plaintext | 4096 | 1.044 | 1.004 | 1.144 | 0.964 | 0.935 | 0.925 | 0.919 | ustun |
| GET /json | 64 | 1.046 | 0.966 | 1.084 | 1.002 | 0.948 | 0.952 | 1.011 | esit |
| GET /json | 256 | 1.012 | 0.939 | 1.041 | 0.970 | 0.995 | 0.994 | 1.006 | esit |
| GET /json | 1024 | 0.976 | 0.943 | 0.989 | 1.081 | 1.044 | 1.032 | 1.018 | fiber |
| GET /json | 4096 | 0.985 | 0.954 | 1.055 | 0.975 | 1.011 | 0.997 | 1.003 | esit |
| GET /users/{id} | 64 | 0.996 | 0.963 | 1.020 | 0.994 | 1.044 | 0.991 | 1.013 | esit |
| GET /users/{id} | 256 | 1.005 | 0.957 | 1.041 | 0.979 | 1.285 | 0.990 | 1.013 | esit |
| GET /users/{id} | 1024 | 0.996 | 0.952 | 1.028 | 1.005 | 1.288 | 0.995 | 1.017 | esit |
| GET /users/{id} | 4096 | 1.005 | 0.972 | 1.026 | 0.872 | 0.967 | 0.963 | 0.961 | esit |
| POST /echo | 64 | 0.993 | 0.936 | 1.052 | 0.993 | 1.060 | 0.991 | 1.013 | esit |
| POST /echo | 256 | 0.998 | 0.953 | 1.052 | 0.997 | 1.011 | 0.991 | 1.007 | esit |
| POST /echo | 1024 | 1.027 | 0.971 | 1.093 | 0.961 | 0.867 | 0.988 | 0.993 | esit |
| POST /echo | 4096 | 0.999 | 0.976 | 1.027 | 1.055 | 1.050 | 1.004 | 1.048 | esit |

Pooled, on this host: 3 ustun, 12 esit, 1 fiber. Default, on this host: 1 ustun, 9 esit, 5 fiber, 1 esik-yok. The one pooled cell still behind Fiber is GET /json at 1024 (CI 0.943–0.989). The last change measured there was the object pool. The remaining allocs in the echo profile are `encoding/json` and `reflect` map assignment, and 45% of CPU samples are `WSASend`. A JSON buffer pool was not added: `SetBody` aliases the bytes and rawhttp writes them after `Handle` returns, so recycling the buffer first would change the response.

## Linux equal track (config `default`, both sides `encoding/json`)

`ubuntu-latest`, 4 vCPU, CPU governor unavailable, so this section is paired ratios only. Server on CPU 0 with `GOMAXPROCS=1`, load generator on CPUs 1–3. 20 rounds. Bootstrap is 10,000 draws, seed 20261013. Go client. `bench/results/linux/go-default.jsonl`.

| Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 0.990 | 0.986 | 1.000 | 1.012 | 1.074 | 1.011 | 0.976 | fiber |
| GET /plaintext | 256 | 1.122 | 1.083 | 1.163 | 0.903 | 0.911 | 0.898 | 0.979 | ustun |
| GET /plaintext | 1024 | 1.083 | 1.030 | 1.106 | 0.925 | 0.962 | 0.929 | 1.062 | ustun |
| GET /plaintext | 4096 | 1.059 | 1.047 | 1.072 | 0.930 | 1.006 | 0.938 | 1.377 | esik-yok |
| GET /json | 64 | 1.021 | 1.000 | 1.047 | 0.983 | 1.039 | 0.980 | 1.082 | esit |
| GET /json | 256 | 1.022 | 1.012 | 1.033 | 0.980 | 1.047 | 0.982 | 0.981 | esik-yok |
| GET /json | 1024 | 1.042 | 0.978 | 1.079 | 0.968 | 1.015 | 0.979 | 1.002 | esit |
| GET /json | 4096 | 1.051 | 1.039 | 1.061 | 0.945 | 0.990 | 0.959 | 1.085 | ustun |
| GET /users/{id} | 64 | 0.979 | 0.967 | 0.986 | 1.028 | 1.082 | 1.021 | 0.977 | fiber |
| GET /users/{id} | 256 | 0.962 | 0.950 | 0.969 | 1.040 | 1.107 | 1.042 | 0.981 | fiber |
| GET /users/{id} | 1024 | 0.986 | 0.946 | 1.021 | 1.006 | 1.092 | 1.006 | 1.002 | esit |
| GET /users/{id} | 4096 | 1.032 | 1.008 | 1.041 | 0.964 | 0.886 | 0.977 | 1.079 | ustun |
| POST /echo | 64 | 0.929 | 0.920 | 0.935 | 1.086 | 1.113 | 1.081 | 0.974 | fiber |
| POST /echo | 256 | 0.954 | 0.924 | 0.984 | 1.027 | 1.006 | 1.011 | 0.978 | fiber |
| POST /echo | 1024 | 0.988 | 0.943 | 1.018 | 1.048 | 1.027 | 1.064 | 1.013 | esit |
| POST /echo | 4096 | 0.982 | 0.972 | 0.992 | 1.007 | 1.003 | 1.014 | 1.071 | fiber |

GET /plaintext at 64 prints a CI high of 1.000; the stored upper bound is 0.9999, so the interval is entirely under 1 and the verdict is fiber. GET /json at 64 prints a CI low of 1.000; the stored lower bound is 0.9998, so the interval contains 1 and the verdict is esit. GET /plaintext at 4096 has peak RSS 1.38× Fiber. RSS is not a gate. Its p99 ratio is 1.006, which is why the cell is esik-yok.

## Linux pooled (`HTTP_POOL_REQUESTS=true`, both sides `encoding/json`)

Same runner and split. `bench/results/linux/go-pooled.jsonl`.

| Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 1.036 | 1.030 | 1.044 | 0.967 | 0.966 | 0.967 | 0.974 | ustun |
| GET /plaintext | 256 | 1.042 | 1.036 | 1.047 | 0.960 | 0.959 | 0.961 | 0.981 | ustun |
| GET /plaintext | 1024 | 1.112 | 1.010 | 1.175 | 0.940 | 0.913 | 0.947 | 0.921 | ustun |
| GET /plaintext | 4096 | 1.098 | 1.086 | 1.106 | 0.910 | 0.856 | 0.904 | 0.950 | ustun |
| GET /json | 64 | 1.067 | 1.057 | 1.078 | 0.937 | 0.933 | 0.935 | 0.974 | ustun |
| GET /json | 256 | 1.068 | 1.063 | 1.080 | 0.939 | 0.904 | 0.935 | 0.980 | ustun |
| GET /json | 1024 | 1.108 | 1.059 | 1.134 | 0.902 | 0.901 | 0.901 | 0.990 | ustun |
| GET /json | 4096 | 1.093 | 1.082 | 1.103 | 0.907 | 0.905 | 0.912 | 0.979 | ustun |
| GET /users/{id} | 64 | 1.000 | 0.992 | 1.016 | 0.992 | 0.994 | 0.991 | 0.976 | esit |
| GET /users/{id} | 256 | 0.995 | 0.990 | 1.003 | 1.006 | 1.007 | 1.001 | 0.981 | esit |
| GET /users/{id} | 1024 | 1.014 | 0.991 | 1.061 | 0.984 | 0.939 | 0.977 | 0.990 | esit |
| GET /users/{id} | 4096 | 1.049 | 1.032 | 1.060 | 0.949 | 0.961 | 0.959 | 1.046 | ustun |
| POST /echo | 64 | 0.976 | 0.960 | 0.986 | 1.032 | 1.039 | 1.031 | 0.970 | fiber |
| POST /echo | 256 | 0.967 | 0.951 | 0.976 | 1.041 | 1.044 | 1.037 | 0.978 | fiber |
| POST /echo | 1024 | 0.988 | 0.970 | 0.997 | 1.025 | 0.996 | 1.021 | 0.988 | fiber |
| POST /echo | 4096 | 1.009 | 1.001 | 1.023 | 0.991 | 0.990 | 0.992 | 1.023 | ustun |

Pooled, on this runner: 10 ustun, 3 esit, 3 fiber. The three fiber cells are POST /echo at 64, 256, and 1024. GET /plaintext at 4096 and GET /json at 1024 are ustun once the request pool is on; the default run left the first esik-yok (p99 1.006, peak RSS 1.38×) and the second esit.

`JSON` still uses `encoding/json.Marshal`. A pooled `json.Encoder` buffer, released from rawhttp `SetBodyRelease` after the write, was slower on the microbench (866.7 ns, 5 allocs versus 1071 ns, 6 allocs) and was not adopted. See `RAWHTTP_CHANGES.md`.

## Linux p99 profile

`GODEBUG=gctrace=1`, `GOMAXPROCS=1`, CPU 0, 12s load, default config (pool off). Pauses below are the steady-state clock time, not the forced GC at process exit.

| Cell | ZATRANO | Fiber |
|---|---|---|
| GET /plaintext 4096 | ~1 GC/s, pause ~14 ms, live heap 165 MB against a 169 MB goal | one steady GC, pause ~15 ms, heap grew to 119 MB |
| GET /json 1024 | ~6 GC/s, pause ~3–4 ms, heap 41 MB | fewer cycles, pause ~4.5 ms, heap 41 MB |
| POST /echo 64 | ~60 GC/s, pause ~0.45 ms, heap 3 MB | ~25 GC/s, pause ~0.36 ms, heap 3 MB |
| GET /users/{id} 64 | ~80 GC/s, pause ~0.4 ms, heap 3 MB | ~20 GC/s, pause ~0.5 ms, heap 3 MB |

Plaintext at 4096 sits on the GC trigger because each connection's read buffer is the 16 KiB header ceiling (`HTTP_MAX_HEADER_BYTES`). 4096 connections keep that memory live, so the collector runs about once a second and stops the single server thread for ~14 ms. That is the p99 ratio of 1.006 and the 1.38× peak RSS. The same cell with `HTTP_POOL_REQUESTS=true` is ustun (p99 0.856, peak RSS 0.95): dropping the per-request garbage moves the heap off that edge. The pool stays off unless the variable is set. `GOGC` was not changed.

Echo and users at 64 are not a long pause. Both heaps are 3 MB and pauses are under half a millisecond. ZATRANO collects several times more often, which is the unpooled request allocation. The pooled matrix still leaves POST /echo at 64, 256, and 1024 entirely under 1, with CPU about 1.03×. That remainder is the handler around `encoding/json`. Encoding into a pooled buffer through `SetBodyRelease` was slower (866.7 ns / 5 allocs versus 1071 ns / 6 allocs) and was not adopted.

`strace -f -e trace=write,writev` recorded two `write` calls on fd 8 when a TCP readiness probe opened a connection: a 104-byte `400`, then one 371-byte `write` for `GET /plaintext`. The response itself is one write. The writes were not combined. The check now waits on the listen socket with `ss`, so that probe is not a request.

## Linux wrk (config `default`, both sides `encoding/json`)

Same split and 20 rounds. wrk 4.1, 2 threads. The first attempt failed because wrk rejects Go's `1.5s`; the warmup is now `2s`. Errors 0. `bench/results/linux/wrk-default.jsonl`.

| Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 1.065 | 1.044 | 1.102 | 0.934 | 0.959 | 0.934 | 0.961 | ustun |
| GET /plaintext | 256 | 1.050 | 0.989 | 1.100 | 0.951 | 0.954 | 0.952 | 0.972 | esit |
| GET /plaintext | 1024 | 1.107 | 1.073 | 1.133 | 0.922 | 0.947 | 0.901 | 1.095 | ustun |
| GET /plaintext | 4096 | 1.073 | 1.039 | 1.113 | 0.906 | 1.033 | 0.931 | 1.308 | esik-yok |
| GET /json | 64 | 1.097 | 1.047 | 1.140 | 0.917 | 0.935 | 0.918 | 0.974 | ustun |
| GET /json | 256 | 1.111 | 1.055 | 1.133 | 0.908 | 0.931 | 0.902 | 0.977 | ustun |
| GET /json | 1024 | 1.121 | 1.082 | 1.169 | 0.890 | 0.961 | 0.906 | 1.016 | ustun |
| GET /json | 4096 | 1.048 | 1.030 | 1.081 | 0.932 | 1.149 | 0.943 | 1.060 | esik-yok |
| GET /users/{id} | 64 | 1.030 | 1.004 | 1.048 | 0.973 | 0.979 | 0.970 | 0.973 | ustun |
| GET /users/{id} | 256 | 1.021 | 0.988 | 1.071 | 0.986 | 0.995 | 0.981 | 0.977 | esit |
| GET /users/{id} | 1024 | 0.999 | 0.967 | 1.067 | 1.021 | 1.017 | 0.986 | 1.016 | esit |
| GET /users/{id} | 4096 | 1.005 | 0.993 | 1.054 | 0.955 | 1.328 | 0.980 | 1.065 | esit |
| POST /echo | 64 | 0.937 | 0.925 | 0.981 | 1.042 | 1.033 | 1.047 | 1.072 | fiber |
| POST /echo | 256 | 0.986 | 0.959 | 1.013 | 0.996 | 1.064 | 0.998 | 0.974 | esit |
| POST /echo | 1024 | 0.960 | 0.937 | 0.977 | 1.029 | 1.079 | 1.035 | 0.996 | fiber |
| POST /echo | 4096 | 0.927 | 0.902 | 0.954 | 1.033 | 1.388 | 1.070 | 1.052 | fiber |

wrk, on this runner: 6 ustun, 5 esit, 3 fiber, 2 esik-yok. Fiber is ahead on POST /echo at 64, 1024, and 4096.

## Linux matched codec (both sides `goccy/go-json` v0.10.5)

Not the gate. Both servers use the same codec: Fiber `JSONEncoder`/`JSONDecoder` and ZATRANO `SetJSONCodec`. The bytes for the bench samples match `encoding/json`, including `<`, `>`, and `&`. Default config, Go client, 20 rounds. `bench/results/linux/goccy-default.jsonl`.

| Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| GET /plaintext | 64 | 0.978 | 0.967 | 0.985 | 1.019 | 1.093 | 1.023 | 0.968 | fiber |
| GET /plaintext | 256 | 1.019 | 0.991 | 1.042 | 0.963 | 1.031 | 0.964 | 0.971 | esit |
| GET /plaintext | 1024 | 1.062 | 1.038 | 1.078 | 0.912 | 1.006 | 0.913 | 1.095 | esik-yok |
| GET /plaintext | 4096 | 1.061 | 1.046 | 1.068 | 0.931 | 0.898 | 0.945 | 1.365 | ustun |
| GET /json | 64 | 0.990 | 0.981 | 1.000 | 0.974 | 1.173 | 1.009 | 0.974 | esit |
| GET /json | 256 | 0.966 | 0.947 | 0.997 | 0.973 | 1.126 | 1.013 | 0.982 | fiber |
| GET /json | 1024 | 1.007 | 0.975 | 1.043 | 0.946 | 1.086 | 0.984 | 1.013 | esit |
| GET /json | 4096 | 0.996 | 0.993 | 1.006 | 0.951 | 1.050 | 0.999 | 1.072 | esit |
| GET /users/{id} | 64 | 0.901 | 0.896 | 0.910 | 1.061 | 1.258 | 1.110 | 0.977 | fiber |
| GET /users/{id} | 256 | 0.874 | 0.865 | 0.901 | 1.084 | 1.235 | 1.142 | 0.985 | fiber |
| GET /users/{id} | 1024 | 0.997 | 0.963 | 1.017 | 0.957 | 1.118 | 1.012 | 1.013 | esit |
| GET /users/{id} | 4096 | 0.961 | 0.948 | 0.971 | 0.979 | 1.092 | 1.037 | 1.073 | fiber |
| POST /echo | 64 | 0.918 | 0.895 | 0.923 | 1.070 | 1.214 | 1.097 | 1.017 | fiber |
| POST /echo | 256 | 0.881 | 0.867 | 0.890 | 1.094 | 1.205 | 1.129 | 0.954 | fiber |
| POST /echo | 1024 | 0.955 | 0.923 | 0.975 | 1.028 | 1.123 | 1.076 | 0.986 | fiber |
| POST /echo | 4096 | 0.940 | 0.930 | 0.955 | 1.018 | 1.078 | 1.065 | 1.062 | fiber |

goccy, on this runner: 1 ustun, 5 esit, 9 fiber, 1 esik-yok. Plaintext does not use the codec and still shows the same shape as the equal track. JSON cells do not move ZATRANO ahead of Fiber. `go test -race ./...` passed on the Linux race job, including the harness under Go 1.26.

## Soak (config `default`)

512 connections, 25s per route, one round, errors 0. RSS from five seconds in to the last sample:

| Route | ZATRANO | Fiber |
|---|---:|---:|
| GET /plaintext | −516 KB | +3.3 MB |
| GET /json | +201 KB | +131 KB |
| GET /users/{id} | +246 KB | −70 KB |
| POST /echo | +319 KB | +131 KB |

ZATRANO stayed within the flat check (growth has to exceed both 5% and 1MB). Fiber plaintext did not.

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

## Round 6 — same GC window

Run `38640064351` at `2f5541e` measured `GET /plaintext` 4096 for 5 seconds. The profile job on that run (`GODEBUG=gctrace=1`, `GOMAXPROCS=1`, server CPU 0) showed why the p99 comparison was not the same condition:

| Server | Steady cycles in the ~8s profile | Mark | Live heap |
|---|---:|---:|---:|
| ZATRANO | 1 (gc 5 at 5.146s) | 17 ms concurrent mark, 0.10 ms stop-the-world | 33 MB |
| Fiber | 0 after startup (last natural gc at 0.783s; heap 97 MB at the forced exit, next goal 151 MB) | startup mark 21 ms | ~72–80 MB |

`GOGC` was not changed and the live heap was not padded. On one CPU the concurrent mark runs on the server thread, so ZATRANO's 17 ms mark lands inside the 5 second window and Fiber's next cycle does not.

The harness now counts `gc N @T` lines whose timestamp falls inside the measured window and stores that count on each row as `gc_cycles`. `-gc-min 3` runs one unscored pass per server, stops that pass when the window contains 3 cycles, then scores both servers for the longer of those two durations plus 10%. The cap is `-measure` (180s for the plaintext 4096 job). The 5 second matrix is unchanged.

Profile window, 40 seconds, `GOMAXPROCS=1`, encoding/json, `GET /plaintext` 4096. Each row is one natural collection at or after 1 second. Forced collections at process exit are not counted. `işaretleme süresi (eşzamanlı)` is the middle gctrace clock number (the mark). On one CPU that mark runs on the server thread. `dünya durdurma (STW)` is the sum of the two stop phases around that mark. Live heap is that cycle's live heap. `GOGC` was not changed.

| Server | Cycle | At | işaretleme süresi (eşzamanlı) | dünya durdurma (STW) | Live heap |
|---|---:|---:|---:|---:|---:|
| ZATRANO | 6 | 5.905 s | 18 ms | 0.135 ms | 36 MB |
| ZATRANO | 7 | 19.556 s | 19 ms | 0.125 ms | 36 MB |
| ZATRANO | 8 | 33.564 s | 18 ms | 0.125 ms | 36 MB |
| Fiber | 7 | 36.757 s | 26 ms | 0.148 ms | 84 MB |

ZATRANO: 3 cycles, mark 55 ms, STW 0.385 ms. Fiber: 1 cycle, mark 26 ms, STW 0.148 ms.

`GOMAXPROCS=4` for plaintext 4096 was not measured. The runner has 4 vCPUs, and pinning the server to CPUs 0–3 leaves no core for the load generator. The single-core series stays the classification basis.

## Round 6 — allocs per request

One keep-alive connection, `net.Pipe`, the same client for ZATRANO, Fiber (resmi middleware), and Fiber (tuned): 64 warmup requests, then 2000 requests. `GOMAXPROCS=1`. Same handlers. Both servers send a request id and the four security headers. ZATRANO is the production `Handle` path with an empty `public/` index and the production timeouts (60s/60s/120s/10s). The framework row excludes the handler's `map[string]string` literal. Echo has no handler literal, so its framework row is the whole handler.

`pipe` is `alloc_objects` whose stack contains `pipeDeadline` (`TestPipeAllocSplitPlaintext`, plaintext, `MemProfileRate=1`). fasthttp calls `SetReadDeadline` and `SetWriteDeadline` on that pipe from `serveConn`; each call allocates a timer. ZATRANO coalesces a still-valid deadline, so its pipe count on this burst is 0. The same 6 pipe allocs are subtracted on the other Fiber routes: the deadline calls do not depend on the handler, and 2000 requests finish inside the one-second coalesce slack. `server` is total minus pipe. Bytes are the `-benchmem` total, pipe timers included.

| Route | ZATRANO total | ZATRANO pipe | ZATRANO server | Fiber (resmi) total | Fiber (resmi) pipe | Fiber (resmi) server | Fiber (tuned) total | Fiber (tuned) pipe | Fiber (tuned) server |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|
| `GET /plaintext` | 0 | 0 | 0 | 8 | 6 | 2 | 6 | 6 | 0 |
| `GET /json` | 3 | 0 | 3 | 14 | 6 | 8 | 13 | 6 | 7 |
| `GET /users/{id}` | 3 | 0 | 3 | 14 | 6 | 8 | 13 | 6 | 7 |
| `POST /echo` | 11 | 0 | 11 | 22 | 6 | 16 | 21 | 6 | 15 |

Bytes, same runs, `B/op`:

| Route | ZATRANO | Fiber (resmi) | Fiber (tuned) |
|---|---:|---:|---:|
| `GET /plaintext` | 0 | 432 | 384 |
| `GET /json` | 368 | 880 | 848 |
| `GET /users/{id}` | 352 | 864 | 832 |
| `POST /echo` | 672 | 1112 | 1080 |

Framework, handler literal excluded. Pipe column is the plaintext profile above.

| Route | ZATRANO total | ZATRANO server | Fiber (resmi) total | Fiber (resmi) server | Fiber (tuned) total | Fiber (tuned) server |
|---|---:|---:|---:|---:|---:|---:|
| `GET /plaintext` | 0 | 0 | 8 | 2 | 6 | 0 |
| `GET /json` | 1 | 1 | 12 | 6 | 11 | 5 |
| `GET /users/{id}` | 1 | 1 | 12 | 6 | 11 | 5 |
| `POST /echo` | 11 | 11 | 22 | 16 | 21 | 15 |

The JSON and users literals are 2 allocs and 336 bytes on both servers. ZATRANO's remaining JSON byte is the encoded body. The users body is 16 bytes. Echo's 11 allocs are `BindJSON` plus the encoded object.

`Fiber (tuned)` keeps `recover` and the four `c.Set` security headers. The request id is not the `requestid` middleware: 16 random bytes are hex-encoded into a pooled `[32]byte` and written with `SetBytesV`, and `Locals` is not used. ZATRANO and Fiber (tuned) use the same entropy source: a `math/rand/v2` ChaCha8 generator seeded once from `crypto/rand` when the pool creates it. Fiber (resmi middleware) calls `crypto/rand` and `hex.EncodeToString` on every generated id.

Official plaintext `alloc_objects`, 2000 requests after 64 warmup, `MemProfileRate=1` (`TestFiberPlaintextAllocObjects` and `TestPipeAllocSplitPlaintext`). Eight mallocs per request:

| Allocs/request | Function | Source |
|---:|---|---|
| 3 | `time.newTimer` via `net.(*pipe).SetReadDeadline` / `SetWriteDeadline` | fasthttp on `net.Pipe` |
| 3 | `net.(*pipeDeadline).set` | same pipe |
| 1 | `encoding/hex.EncodeToString` | resmi harness generator |
| 1 | `requestid.New.func1` → `fiber.StoreInContext` | resmi harness `Locals` |

`c.Set` of the four security headers and `recover` do not allocate on this path. Fiber core is not in the eight. Tuned drops the generator and `StoreInContext`. The 6 that remain are the pipe. Tuned server allocs on this bench are 0.

The comparison that decides the alloc result is the server process. The parent reads the child `runtime.MemStats` twice. `allocs_per_req` is the steady window: the snapshot at the start of the measured interval (connections already up) through the end, divided by measured requests. `allocs_per_req_with_setup` is the accept through the end of that same interval, divided by warmup plus measured requests. One earlier window on this host, before the hijack-mark fix, Intel Core i5-6200U, `GOMAXPROCS=1`, plaintext, encoding/json, 1 s warmup, 3 s measured:

| Server | Requests | alloc/request | bytes/request |
|---|---:|---:|---:|
| ZATRANO | 67710 | 1.114 | 41.1 |
| Fiber (resmi middleware) | 60182 | 2.190 | 116.3 |
| Fiber (tuned) | 82662 | 0.128 | 47.4 |

That 1.114 was `serveLoop` allocating `ctx.markHijacked = func() { cs.hijacked.Store(true) }` on every request (`alloc_objects` flat 78755 of 114663 at 4096). `rawhttp` 60afd8f builds the closure once per connection and assigns that value after `reset`.

After that fix, one 40 second window on this host, `GOMAXPROCS=1`, plaintext, encoding/json, 2 s warmup, 4096 connections. Not a Linux class. Steady is `allocs_per_req`. With-setup is `allocs_per_req_with_setup`. GC is `gc_cycles` inside the measured 40 s (`GODEBUG=gctrace=1`). `GOGC` was not changed.

| Server | Steady alloc/req | Steady bytes/req | With setup alloc/req | With setup bytes/req | GC cycles |
|---|---:|---:|---:|---:|---:|
| ZATRANO | 0.007 | 1.95 | 0.013 | 3.49 | 0 |
| Fiber (resmi middleware) | 2.011 | 51.7 | 2.015 | 53.5 | 10 |
| Fiber (tuned) | 0.010 | 3.57 | 0.019 | 6.99 | 1 |

Steady ZATRANO alloc/request is under Fiber (tuned), and the GC count in that window is under Fiber (tuned). The one-round paired rps on this laptop is not a cell class.

The fiber pair now runs three servers. `zatrano/fiber-tuned` is the binding paired row. `zatrano/fiber` stays the official-middleware column. `cellVerdict` is unchanged; on the tuned row `fiber` means the tuned server won. Run `38702939949` (`9abed55`) and run `38712986781` (`9c63264`) do not contain the tuned column.

Run `38702939949`, matrix jobs finished, strace still running. Stable class is two of three runs. The gate is `hedef` only when `ci_low > 1`, ratio ≥ 1.05, p99 < 1, CPU/request < 1, and RSS ≤ 1.10. These classes are against Fiber (resmi middleware).

go-default: 15 harness `ustun` and gate `hedef`. `GET /plaintext` 4096 is harness `esik-yok`, gate p99 (ratios 1.089 / 1.126 / 1.091, p99 1.035 / 0.894 / 1.034). That cell is the 5 second window, not the fair window above.

wrk-default: 16 harness `ustun` and 16 gate `hedef`.

goccy-default, harness then gate:

| Cell | Harness | Gate |
|---|---|---|
| GET /json 64 | ustun | hedef |
| GET /json 256 | esik-yok | kararsız |
| GET /json 1024 | esik-yok | p99 |
| GET /json 4096 | ustun | rps |
| GET /plaintext 64, 256, 1024, 4096 | ustun | hedef |
| GET /users/{id} 64 | ustun | rps |
| GET /users/{id} 256 | kararsız | rps+p99 |
| GET /users/{id} 1024 | esik-yok | p99 |
| GET /users/{id} 4096 | ustun | rps |
| POST /echo 64, 256, 1024, 4096 | fiber | rps+p99+cpu |

The goccy echo cells and goccy `GET /users/{id}` 256 stay closed by the ceiling table below.

rawhttp/fasthttp, median of the three paired ratios. A harness `fiber` would mean fasthttp won. None of the stable classes are that.

| Conns | Harness | Gate | rps | p99 | RSS |
|---:|---|---|---:|---:|---:|
| 64 | kararsız | rps+rss | 1.008 | 0.984 | 1.214 |
| 256 | kararsız | kararsız | 1.019 | 1.051 | 1.059 |
| 1024 | ustun | kararsız | 1.075 | 0.936 | 1.060 |
| 4096 | ustun | hedef | 1.112 | 0.907 | 0.896 |

Run `38712986781` is still in progress (`go-default`, `wrk-default`, `goccy-default`, `go-plaintext-4096` run 1, strace). Its three-run class is not reported. It also has no tuned column.

Old plaintext 4096, three runs, 5 second window, encoding/json, harness verdict then the 1.05 gate:

| Run | Harness | Gate | Ratio | CI | p99 | CPU | RSS |
|---|---|---|---:|---|---:|---:|---:|
| 1 | ustun | hedef | 1.136 | 1.125–1.150 | 0.893 | 0.869 | 0.583 |
| 2 | esik-yok | p99 | 1.114 | 1.111–1.116 | 1.026 | 0.894 | 0.545 |
| 3 | esik-yok | p99 | 1.092 | 1.083–1.107 | 1.011 | 0.909 | 0.535 |

Stable class: harness esik-yok, gate p99. The 5 second window is the row above. The fair window below is run `38702939949` at `9abed55`, `go-plaintext-4096`, three jobs. `gc_cycles` is the median row. Both servers in a job share that job's calibrated duration. p99 is the paired ratio of the median p99.

| Run | Window | ZATRANO GC | Fiber GC | p99 ratio | rps ratio | Harness |
|---|---:|---:|---:|---:|---:|---|
| 1 | 94.2 s | 6 | 3 | 0.921 | 1.087 | ustun |
| 2 | 57.6 s | 7 | 3 | 0.899 | 1.155 | ustun |
| 3 | 77.2 s | 7 | 3 | 0.910 | 1.078 | ustun |

p99 is under 1 on every run, so the slow-request timestamps were not matched against gctrace and the queue was not profiled on the scheduler. The request-id allocation is not in this run. The next `go-plaintext-4096` jobs are the comparison after that allocation is gone.

## Round 6 — ceiling

The ceiling is the paired throughput ratio times the paired CPU/request ratio. That credits the whole CPU gap to the framework. A real framework-only share is smaller, so a ceiling already under 1.05 cannot be lifted over the bar by deleting framework CPU. The interval and `cellVerdict` are unchanged. These cells are closed:

| Cell | Codec | Median ratio | Median CPU ratio | Ceiling |
|---|---|---:|---:|---:|
| POST /echo 64 | goccy | 0.948 | 1.052 | 0.997 |
| POST /echo 256 | goccy | 0.920 | 1.093 | 1.006 |
| POST /echo 1024 | goccy | 0.957 | 1.046 | 1.001 |
| POST /echo 4096 | goccy | 0.914 | 1.096 | 1.002 |
| GET /users/{id} 256 | goccy | 0.997 | 1.002 | 0.999 |

`BindJSON` already hands the configured codec the connection buffer. A codec installed with `SetJSONCodec` sees that slice, not a copy (`TestBindJSONCodecSeesTheConnectionBuffer`). The one-key fast path still runs only when the codec is encoding/json, so the goccy track and Fiber decode with the same codec.

## Round 6 — red jobs

`go test -race` on run `38723870698` (`a335030`) failed in the harness step `framework/bench/ztfb`, not in the framework module. `TestFiberTunedResponseBytesMatchZatrano` starts ZATRANO and Fiber (tuned) in one process. Both call `loadCodec`, which writes the package variables `codecMarshal` and `codecUnmarshal`. The race is those two writes (`servers.go`). The two stacks are `serveZatrano` and `serveFiberTuned`. Production servers are separate processes and set the same codec. `loadCodec` now holds `codecMu` around the write.

`go-plaintext-4096` on the same run failed with `fiber-tuned GET /plaintext c=4096: 1 GC cycles in 180.1s, need 3`. The job asked for a window long enough for 3 GC cycles. Both servers are nearly allocation-free, so that count may never arrive (a 40 s window on this host recorded 0 cycles for ZATRANO and 1 for Fiber tuned). The job is now a fixed 60 second window for every server. `ZTFB_GCTRACE=1` records `gc_cycles` in that window, and 0 is a real count. Scoring stays inside the window. The rule is the same for both sides. `cellVerdict` and the 5 second matrix are unchanged. `-gc-min 3` remains available and is not set on this job.

## Round 6 — deadline granule

rawhttp `v0.2.9` (`112df04788eebb7013edfbe73536fdddc5fba052`) keeps the "do not refresh a still-valid deadline" path. The granule is `min(d/8, 1s)`. A kept deadline is at most one granule earlier than `now+d` and is never later. The wait between keep-alive requests is `IdleTimeout` (90s when unset). `ReadTimeout` covers the body and a read once a request has started. Tests: an idle connection with `IdleTimeout` equal to `ReadTimeout` closes within `ReadTimeout` plus the granule; a connection that sends a request every `ReadTimeout/2` stays up for `3× ReadTimeout`; a reader that does not drain the body is cut by `WriteTimeout`. The granule does not exceed 1 second, and the tests passed, so the optimization stays. The framework module requires `github.com/zatrano/rawhttp v0.2.9`. `TestReleaseModuleHasNoReplace` passed. There is no rawhttp replace.

## Round 6 — RSS baseline and slope

One sample on this host (Windows/amd64, Intel Core i5-6200U, `GOMAXPROCS=1`). Each server is measured with no connections, then with 64 and 4096 keep-alive connections held open after one `GET /plaintext`. The slope is `(RSS_4096 − RSS_0) / 4096`. This is not a Linux class. The RSS gate is still 1.10. It was not loosened.

| Server | 0 conn | 64 conn | 4096 conn | KB/conn |
|---|---:|---:|---:|---:|
| ZATRANO | 8.00 MiB | 11.31 MiB | 128.22 MiB | 30.06 |
| Fiber (tuned) | 7.74 MiB | 9.87 MiB | 120.48 MiB | 28.19 |
| rawhttp | 7.55 MiB | 8.82 MiB | 79.04 MiB | 17.87 |
| fasthttp | 7.62 MiB | 9.22 MiB | 93.94 MiB | 21.58 |

ZATRANO's idle gap against Fiber (tuned) is 272 KiB. **Sabit taban** is only this gap. It is the kernel: config, the logger, request-id and security-header middleware, trim, CORS, the empty public-file index, and the container. The bench registers no providers. The request pool is empty at 0 connections. None of this can be removed without dropping a feature.

A second sample on the same host separates the high-water mark while the connections are held from the working set five seconds after they close. Private commit is `PagefileUsage`.

| Server | 0 conn | 64 open | 64 HWM | 64 after 5s | 4096 open | 4096 HWM | 4096 after 5s |
|---|---:|---:|---:|---:|---:|---:|---:|
| ZATRANO | 8.21 MiB | 10.12 MiB | 10.12 MiB | 55.5 MiB | 127.9 MiB | 127.9 MiB | 127.0 MiB |
| Fiber (tuned) | 7.80 MiB | 10.44 MiB | 10.45 MiB | 10.48 MiB | 120.8 MiB | 120.8 MiB | 120.5 MiB |
| rawhttp | 7.61 MiB | 9.32 MiB | 9.32 MiB | 9.33 MiB | 79.8 MiB | 79.8 MiB | 78.8 MiB |
| fasthttp | 7.63 MiB | 9.63 MiB | 9.64 MiB | 9.66 MiB | 93.9 MiB | 93.9 MiB | 93.6 MiB |

While 64 connections are open, ZATRANO and fasthttp both show about 48 MiB of private commit and about 10 MiB of working set. Five seconds after the close, ZATRANO's working set rises to that commit and the others stay near the high-water mark. `runtime.MemStats` at that moment is about 8 MiB of `Sys`, so the rise is pages of the Go arena being touched, not a second copy of the handler state.

`runtime.MemStats` from 0 to 64 connections, per connection. One sample on this host. The four fields sum to about 16.8 KB on ZATRANO and about 40.5 KB on Fiber (tuned).

| Server | HeapInuse | StackInuse | GCSys | OtherSys | Sum |
|---|---:|---:|---:|---:|---:|
| ZATRANO | 9.3 KB | 8.0 KB | 1.3 KB | −1.8 KB | 16.8 KB |
| Fiber (tuned) | 21.0 KB | 16.0 KB | 1.6 KB | 1.9 KB | 40.5 KB |

The stack is one goroutine stack. The heap includes the connection read buffer. There is nothing in this split to delete without removing the connection.

The 55.5 MiB figure is the working set five seconds after the 64 connections close, on this Windows host. A repeat with `GetProcessMemoryInfo` and a `QueryWorkingSet` / `VirtualQueryEx` split of the resident pages:

| | ZATRANO open | ZATRANO +5 s | Fiber (tuned) open | Fiber (tuned) +5 s |
|---|---:|---:|---:|---:|
| Working set | 10.18 MiB | 55.54 MiB | 10.48 MiB | 10.52 MiB |
| Private commit (`PagefileUsage`) | 48.02 MiB | 48.02 MiB | 49.12 MiB | 49.13 MiB |
| Resident private | 3.48 MiB | 35.75 MiB | 3.88 MiB | 3.91 MiB |
| Resident shareable | 6.67 MiB | 19.76 MiB | 6.57 MiB | 6.57 MiB |
| Resident mapped | 0.24 MiB | 0.24 MiB | 0.22 MiB | 0.22 MiB |

Private commit does not grow. Mapped bytes do not grow. The extra residency is pages that were already committed: about 32 MiB become private resident and about 13 MiB become shareable resident. Fiber (tuned) holds the same kind of commit and leaves it off the working set. Run `38752314433` does not sample `/proc/<pid>/smaps_rollup` five seconds after close. At the end of its window, `rss_bytes` equals `hwm_bytes`. For `GET /plaintext` at 64 connections those are about 15.1 MiB for ZATRANO and about 15.9 MiB for Fiber (tuned), on all three runs (one Fiber tuned high-water mark is 17.8 MiB). That is the steady end-of-window working set, not a 55 MiB spike. The 55.5 MiB rise is a Windows residency artifact of this host.

The Linux 64-connection RSS comparison is that end-of-window pair. ZATRANO's `hwm_bytes` and `rss_bytes` are both below Fiber (tuned). The ratio is about 0.95. It is not a ZATRANO excess, and it is not classified as a high-water-only gap: the two fields match.

## Round 6 — classes (`38752314433`, `6e96194`)

Binding column is Fiber (tuned). `zatrano/fiber` stays the official-middleware column and is not the verdict. A class is stable when at least two of the three runs agree. The harness verdict is `cellVerdict`. The 1.05 gate is `hedef` only when `ci_low > 1`, the paired ratio is at least 1.05, p99 and CPU/request are below 1, and the high-water RSS ratio is at most 1.10. `rss_bytes` equals `hwm_bytes` on these rows: the end-of-window working set is the high-water mark.

go-default: 16 stable harness `ustun`, 16 stable gate `hedef`.

wrk-default: 16 stable harness `ustun`, 16 stable gate `hedef`. Two runs in the third sample miss the gate alone (`GET /plaintext` 256 and `GET /users/{id}` 256 are `rps`; `POST /echo` 4096 is `p99`) and do not change the stable class.

goccy-default, as measured. There is no stable `esit` cell. 6 cells are stable `ustun` and `hedef`: `GET /plaintext` 64, 256, 1024, 4096, `GET /json` 64, and `GET /users/{id}` 64. 6 cells are stable `esik-yok`: `GET /json` 256 and 1024 and `GET /users/{id}` 256 and 1024 miss on p99; `GET /json` 4096 and `GET /users/{id}` 4096 miss on rps and p99. Fiber (tuned) is ahead on `POST /echo` 64, 256, 1024, and 4096 (stable `fiber`, rps+p99+cpu). Those four echo cells stay closed by the ceiling.

rawhttp/fasthttp, plaintext, as measured. There is no stable `esit` and Fiber's column is not this pair: a `fiber` verdict here would mean fasthttp ahead, and none of the four cells is that. 64 and 256 are stable `esik-yok` (rps+p99; ratios about 1.02 and 1.03). 1024 is stable `ustun` and the stable gate is `rps` (ratios 1.041, 1.054, 1.032), short of 1.05. 4096 is stable `ustun` and `hedef` (ratio about 1.10, p99 about 0.91, RSS ratio about 0.80).

The 60 second `go-plaintext-4096` window is stable `ustun` / `hedef` (ratios 1.122, 1.086, 1.069). GC cycles in that window: ZATRANO 0, Fiber (tuned) 0, official Fiber 2, on every run. Alloc/request there is about 0.0015 for ZATRANO and about 0.0018 for Fiber (tuned). On the 5 second go-default window, `GET /plaintext` alloc/request is about 0.00031 at 64 connections and about 0.018 at 4096 for ZATRANO, and about 0.00055 and 0.020 for Fiber (tuned). The 5 second jobs do not set `gc_cycles`.

Stable class distribution: default and wrk are 16/16 `ustun` and `hedef`; goccy is 6 `ustun`/`hedef`, 6 `esik-yok`, 4 `fiber`; rawhttp/fasthttp is 2 `ustun` and 2 `esik-yok`, and only plaintext 4096 is `hedef`.

## Pool off and pool on (`b966476`)

Shared 4-vCPU runner. SHA `b9664769deb52ac62fb5187d90c0b09c8aa6dd73`. 20 rounds, three runs. Fiber v3.6.0 (tuned), `encoding/json`. The binding row is `zatrano/fiber-tuned`. Each number is the median of the three paired summaries. The verdict is the class shared by at least two runs. Peak RSS is reported and is not a gate. `HTTP_POOL_REQUESTS` unset is the product default and leaves the pool off. `HTTP_POOL_REQUESTS=true` turns it on.

### default (pool off)

Run `38815118387`. `HTTP_POOL_REQUESTS` unset.

| Client | Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| go | GET /plaintext | 64 | 1.052 | 1.048 | 1.059 | 0.947 | 0.964 | 0.949 | 1.118 | ustun |
| go | GET /plaintext | 256 | 1.027 | 1.023 | 1.032 | 0.972 | 1.055 | 0.972 | 0.973 | esik-yok |
| go | GET /plaintext | 1024 | 1.035 | 1.031 | 1.041 | 0.950 | 1.136 | 0.964 | 0.939 | esik-yok |
| go | GET /plaintext | 4096 | 1.049 | 1.023 | 1.053 | 0.933 | 1.099 | 0.953 | 0.809 | esik-yok |
| go | GET /json | 64 | 1.105 | 1.098 | 1.107 | 0.910 | 0.917 | 0.905 | 0.955 | ustun |
| go | GET /json | 256 | 1.080 | 1.077 | 1.085 | 0.927 | 1.034 | 0.926 | 0.758 | esik-yok |
| go | GET /json | 1024 | 1.086 | 1.074 | 1.127 | 0.894 | 0.922 | 0.889 | 0.674 | ustun |
| go | GET /json | 4096 | 1.077 | 1.056 | 1.092 | 0.918 | 0.882 | 0.928 | 0.571 | ustun |
| go | GET /users/{id} | 64 | 1.075 | 1.069 | 1.080 | 0.932 | 0.939 | 0.931 | 0.956 | ustun |
| go | GET /users/{id} | 256 | 1.063 | 1.055 | 1.068 | 0.944 | 1.032 | 0.944 | 0.758 | esik-yok |
| go | GET /users/{id} | 1024 | 1.091 | 1.061 | 1.137 | 0.911 | 0.939 | 0.906 | 0.674 | ustun |
| go | GET /users/{id} | 4096 | 1.097 | 1.070 | 1.103 | 0.894 | 0.892 | 0.908 | 0.571 | ustun |
| go | POST /echo | 64 | 1.065 | 1.055 | 1.070 | 0.936 | 0.949 | 0.937 | 0.957 | ustun |
| go | POST /echo | 256 | 1.037 | 1.036 | 1.048 | 0.961 | 0.998 | 0.957 | 0.757 | ustun |
| go | POST /echo | 1024 | 1.109 | 1.081 | 1.143 | 0.893 | 0.948 | 0.895 | 0.643 | ustun |
| go | POST /echo | 4096 | 1.116 | 1.102 | 1.122 | 0.880 | 0.887 | 0.897 | 0.606 | ustun |
| wrk | GET /plaintext | 64 | 1.003 | 0.977 | 1.043 | 0.991 | 1.003 | 0.993 | 1.254 | esit |
| wrk | GET /plaintext | 256 | 1.000 | 0.956 | 1.033 | 1.000 | 1.039 | 1.000 | 0.973 | esit |
| wrk | GET /plaintext | 1024 | 1.033 | 0.992 | 1.075 | 0.954 | 1.044 | 0.971 | 0.896 | esit |
| wrk | GET /plaintext | 4096 | 1.035 | 1.008 | 1.054 | 0.993 | 1.075 | 0.974 | 0.790 | esik-yok |
| wrk | GET /json | 64 | 1.137 | 1.109 | 1.188 | 0.879 | 0.914 | 0.884 | 0.959 | ustun |
| wrk | GET /json | 256 | 1.107 | 1.069 | 1.135 | 0.924 | 0.963 | 0.918 | 0.760 | ustun |
| wrk | GET /json | 1024 | 1.153 | 1.084 | 1.164 | 0.841 | 0.978 | 0.870 | 0.674 | ustun |
| wrk | GET /json | 4096 | 1.095 | 1.085 | 1.126 | 0.898 | 0.959 | 0.907 | 0.578 | ustun |
| wrk | GET /users/{id} | 64 | 1.084 | 1.063 | 1.128 | 0.908 | 0.922 | 0.907 | 0.959 | ustun |
| wrk | GET /users/{id} | 256 | 1.052 | 1.011 | 1.093 | 0.937 | 0.977 | 0.937 | 0.760 | ustun |
| wrk | GET /users/{id} | 1024 | 1.060 | 1.019 | 1.108 | 0.933 | 1.000 | 0.940 | 0.674 | ustun |
| wrk | GET /users/{id} | 4096 | 1.076 | 1.045 | 1.120 | 0.902 | 0.969 | 0.918 | 0.572 | ustun |
| wrk | POST /echo | 64 | 1.101 | 1.073 | 1.173 | 0.882 | 0.900 | 0.890 | 0.957 | ustun |
| wrk | POST /echo | 256 | 1.072 | 1.037 | 1.104 | 0.928 | 0.959 | 0.932 | 0.761 | ustun |
| wrk | POST /echo | 1024 | 1.113 | 1.066 | 1.140 | 0.881 | 0.965 | 0.895 | 0.642 | ustun |
| wrk | POST /echo | 4096 | 1.094 | 1.084 | 1.119 | 0.896 | 0.917 | 0.906 | 0.626 | ustun |

Go client: 11 `ustun`, 5 `esik-yok` (`GET /plaintext` 256, 1024, 4096; `GET /json` 256; `GET /users/{id}` 256). wrk: 12 `ustun`, 3 `esit` (`GET /plaintext` 64, 256, 1024), 1 `esik-yok` (`GET /plaintext` 4096). No cell is stable `fiber`. wrk `GET /plaintext` 256 is `fiber` on one run and `esit` on the other two, so the stable verdict stays `esit`.

### HTTP_POOL_REQUESTS=true

Run `38815127116`.

| Client | Route | Conns | Ratio | CI low | CI high | p50 | p99 | CPU/req | Peak RSS | Verdict |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|
| go | GET /plaintext | 64 | 1.118 | 1.109 | 1.123 | 0.891 | 0.909 | 0.895 | 0.846 | ustun |
| go | GET /plaintext | 256 | 1.110 | 1.108 | 1.121 | 0.895 | 0.873 | 0.894 | 0.758 | ustun |
| go | GET /plaintext | 1024 | 1.122 | 1.106 | 1.153 | 0.888 | 0.869 | 0.896 | 0.624 | ustun |
| go | GET /plaintext | 4096 | 1.089 | 1.077 | 1.103 | 0.922 | 0.873 | 0.909 | 0.511 | ustun |
| go | GET /json | 64 | 1.145 | 1.124 | 1.159 | 0.879 | 0.869 | 0.878 | 0.954 | ustun |
| go | GET /json | 256 | 1.136 | 1.127 | 1.147 | 0.882 | 0.909 | 0.882 | 0.757 | ustun |
| go | GET /json | 1024 | 1.173 | 1.152 | 1.208 | 0.851 | 0.862 | 0.847 | 0.674 | ustun |
| go | GET /json | 4096 | 1.109 | 1.102 | 1.121 | 0.898 | 0.861 | 0.900 | 0.531 | ustun |
| go | GET /users/{id} | 64 | 1.113 | 1.108 | 1.129 | 0.899 | 0.887 | 0.897 | 0.958 | ustun |
| go | GET /users/{id} | 256 | 1.102 | 1.092 | 1.123 | 0.904 | 0.868 | 0.895 | 0.757 | ustun |
| go | GET /users/{id} | 1024 | 1.114 | 1.110 | 1.175 | 0.866 | 0.896 | 0.868 | 0.659 | ustun |
| go | GET /users/{id} | 4096 | 1.108 | 1.103 | 1.111 | 0.899 | 0.897 | 0.900 | 0.512 | ustun |
| go | POST /echo | 64 | 1.077 | 1.066 | 1.085 | 0.928 | 0.934 | 0.930 | 0.959 | ustun |
| go | POST /echo | 256 | 1.069 | 1.062 | 1.077 | 0.935 | 0.953 | 0.936 | 0.759 | ustun |
| go | POST /echo | 1024 | 1.119 | 1.100 | 1.161 | 0.902 | 0.897 | 0.896 | 0.644 | ustun |
| go | POST /echo | 4096 | 1.102 | 1.094 | 1.107 | 0.907 | 0.897 | 0.909 | 0.584 | ustun |
| wrk | GET /plaintext | 64 | 1.107 | 1.068 | 1.134 | 0.917 | 0.940 | 0.916 | 0.951 | ustun |
| wrk | GET /plaintext | 256 | 1.112 | 1.048 | 1.165 | 0.902 | 0.942 | 0.901 | 0.759 | ustun |
| wrk | GET /plaintext | 1024 | 1.138 | 1.102 | 1.191 | 0.860 | 0.925 | 0.874 | 0.624 | ustun |
| wrk | GET /plaintext | 4096 | 1.108 | 1.088 | 1.131 | 0.897 | 0.912 | 0.896 | 0.533 | ustun |
| wrk | GET /json | 64 | 1.158 | 1.128 | 1.203 | 0.860 | 0.859 | 0.861 | 0.955 | ustun |
| wrk | GET /json | 256 | 1.166 | 1.114 | 1.205 | 0.864 | 0.873 | 0.857 | 0.759 | ustun |
| wrk | GET /json | 1024 | 1.149 | 1.126 | 1.200 | 0.867 | 0.886 | 0.863 | 0.673 | ustun |
| wrk | GET /json | 4096 | 1.150 | 1.124 | 1.181 | 0.879 | 0.887 | 0.874 | 0.547 | ustun |
| wrk | GET /users/{id} | 64 | 1.153 | 1.138 | 1.180 | 0.885 | 0.866 | 0.884 | 0.957 | ustun |
| wrk | GET /users/{id} | 256 | 1.143 | 1.131 | 1.192 | 0.886 | 0.870 | 0.874 | 0.759 | ustun |
| wrk | GET /users/{id} | 1024 | 1.154 | 1.114 | 1.212 | 0.845 | 0.896 | 0.879 | 0.643 | ustun |
| wrk | GET /users/{id} | 4096 | 1.127 | 1.098 | 1.159 | 0.897 | 0.914 | 0.883 | 0.522 | ustun |
| wrk | POST /echo | 64 | 1.138 | 1.114 | 1.197 | 0.869 | 0.864 | 0.873 | 1.004 | ustun |
| wrk | POST /echo | 256 | 1.177 | 1.120 | 1.233 | 0.841 | 0.862 | 0.840 | 0.759 | ustun |
| wrk | POST /echo | 1024 | 1.206 | 1.173 | 1.256 | 0.792 | 0.882 | 0.814 | 0.644 | ustun |
| wrk | POST /echo | 4096 | 1.141 | 1.089 | 1.162 | 0.887 | 0.881 | 0.882 | 0.604 | ustun |

Go client and wrk are each 16/16 stable `ustun`.
