# rawhttp changes required by this branch

## Read buffer grows from 4 KiB

| | |
| --- | --- |
| File | `conn.go`, `server.go` (`connReadBufSize`, `fillHeaders`, `shrinkIdle`) |
| Why | Framework sets `MaxHeaderBytes` to 16 KiB. rawhttp used to allocate that as the connection read buffer, so 4096 connections pinned about 64 MiB before any body. The buffer now starts at 4 KiB when the ceiling is above 8 KiB, and grows 4 → 8 → 16 KiB only while the header block is incomplete. A buffered Content-Length body is a view of that buffer during the handler; a response that still points into it is copied once before the buffer is compacted. On idle, with nothing buffered, a grown buffer returns to a pool capped at 64 per size. |
| Version | On rawhttp `main`, commit `693b49c`. Not tagged. The framework module still requires v0.2.6 and must not contain a `replace`. The harness module replaces rawhttp for the Linux bench; a `replace` in the framework module is ignored by that build and fails `TestReleaseModuleHasNoReplace`. A ceiling of 8 KiB or less keeps the previous buffer size. A header past `MaxHeaderBytes` is still 431. |

## v0.2.7 — `Ctx.SetBodyRelease`

| | |
| --- | --- |
| File | `ctx.go` (`SetBodyRelease`, `SetBodyReleaseDebug`, `finishBodyRelease`), `response.go` (`writeResponse` after the write returns), `server.go` (hijack, stream drain error, connection end) |
| Why | A pooled response body cannot be released at the end of `Handle`. rawhttp copies it into the write buffer and calls `write` after `Handle` returns. The hook runs once after that write, and also once when the write fails, the handler hijacks, or the connection ends without a write. |
| Version | The `perf/body-release` branch was removed. rawhttp `main` has no release hook. v0.2.6 has no release hook. The framework still requires v0.2.6 and does not call `SetBodyRelease`. |

`SetBodyRelease` turns off the keep-alive body cache for that response. The cache keys on the slice pointer.

## JSON buffer pool — measured, not adopted

`JSON` encoded with `json.NewEncoder` into a `sync.Pool` buffer and returned the buffer from the hook. Bytes matched `encoding/json.Marshal`, including `<`, `>`, and `&`. Buffers over 64 KiB were not returned. `APP_DEBUG` panicked on `Content` after release.

Same host, `go test -benchmem` (`Hello, World!` map):

| Path | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `encoding/json.Marshal` | 866.7 | 688 | 5 |
| pooled `Encoder` | 1071 | 688 | 6 |

`Marshal` already pools its encode buffer and copies the result out. `NewEncoder` adds that copy plus an encoder allocation. The pooled path is slower, so a general value stays on `encoding/json.Marshal`. A one-key or empty `map[string]string` of ASCII is encoded directly: `BenchmarkJSONWithSecurityHeaders` is 1 alloc and 32 B, and `BenchmarkPlaintext` is 0 alloc. Bytes still match `encoding/json.Marshal`, including `<`, `>`, and `&`. A custom codec (`SetJSONCodec`, the goccy track) skips that path. The framework module still requires rawhttp v0.2.6 and does not call `SetBodyRelease`.
