# Performance impact of copy-on-parse

This file records the measured cost of the `safe-by-construction` branch
against tagged v3.2.4. Speed is not a gate. Numbers appear only after they
are measured with the existing `bench/` harness. Invented figures are not
written here.

The request is copied into a per-request arena (method, path, query, headers,
peer, TLS, body). `Request` and `Response` objects are not pooled.
`HTTP_POOL_REQUESTS` is ignored.

## Method

Compare this branch with `v3.2.4` on the same machine, same Go toolchain,
same `bench/` client. Report CPU µs/request, allocs/request, peak RSS, rps
and p99 for the routes the harness already runs.

Host: Windows/amd64, Intel Core i5-6200U @ 2.30 GHz, `GOTOOLCHAIN=go1.26.9`,
`GOWORK=off`. Command:

```
go test -bench='^BenchmarkCanonT0$|^BenchmarkCanonT1$|^BenchmarkCanonT1Echo$' -benchmem -count=3 -benchtime=2s
```

in `bench/`. That is the published Canon in-process harness (`faithConn`).
It prints ns/op, B/op and allocs/op. It does not print p99 or peak RSS;
those come from the separate `ztfb` load generator, which was not re-run
this session.

v3.2.4 was measured from a git worktree at `db49c35`. This branch was
measured while `go test ./...` was still finishing, so ns/op on this
branch is noisy. allocs/op and B/op were identical across the three
counts and are the comparison that does not depend on that load.

µs/request = ns/op / 1000. rps ≈ 1e9 / ns/op. The table uses the median
of the three counts.

## Results

| Route | Metric | v3.2.4 (median) | this branch (median) | Delta |
|-------|--------|----------------:|---------------------:|------:|
| Canon T0 `GET /plaintext` | µs/request | 3.23 | 5.94 | +2.70 |
| Canon T0 | allocs/request | 2 | 3 | +1 |
| Canon T0 | B/request | 1088 | 2000 | +912 |
| Canon T0 | rps (from median ns/op) | 3.09e5 | 1.68e5 | −46% |
| Canon T1 | µs/request | 5.45 | 59.2 | noisy; see notes |
| Canon T1 | allocs/request | 2 | 12 | +10 |
| Canon T1 | B/request | 1088 | 5424 | +4336 |
| Canon T1 Echo | µs/request | 5.44 | 23.5 | noisy; see notes |
| Canon T1 Echo | allocs/request | 3 | 12 | +9 |
| Canon T1 Echo | B/request | 1104 | 5456 | +4352 |
| any Canon row | p99 | not reported by this harness | not reported by this harness | — |
| any Canon row | peak RSS | not reported by this harness | not reported by this harness | — |

Arena allocation cost (bytes copied per typical GET /plaintext): the T0
row is that request. The extra 912 B/op and 1 alloc/op are the owned
arena plus the unpooled `Request` object against v3.2.4 with pooling off
(the published default). Tier-1 adds kernel middleware, so the extra
10 allocs / ~4.3 KiB include `Request.Own` copies of the response at
middleware boundaries.

## Notes

- v3.2.4 T0 counts: 6635, 2603, 3234 ns/op (median 3234). This branch:
  12687, 5935, 5062 ns/op (median 5935). First counts on both sides are
  cold.
- v3.2.4 T1 counts: 4981, 5452, 5933 ns/op. This branch: 16672, 97564,
  59199 ns/op. The 97 µs sample ran under a competing `go test ./...`
  and is not a fair time measurement. Allocs/op stayed 12 on every count.
- Rows that used `HTTP_POOL_REQUESTS=true` on v3.2.4 describe a mode that
  no longer exists. The comparison for this branch is against v3.2.4 with
  the pool off (the published default).
- Speed is not a gate. The copy-on-parse cost is the extra allocations.
