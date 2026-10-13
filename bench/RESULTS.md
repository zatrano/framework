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
