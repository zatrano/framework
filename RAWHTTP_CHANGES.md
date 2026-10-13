# rawhttp changes required by this branch

## v0.2.7 — `Ctx.SetBodyRelease`

| | |
| --- | --- |
| File | `ctx.go` (`SetBodyRelease`, `SetBodyReleaseDebug`, `finishBodyRelease`), `response.go` (`writeResponse` after the write returns), `server.go` (hijack, stream drain error, connection end) |
| Why | A pooled response body cannot be released at the end of `Handle`. rawhttp copies it into the write buffer and calls `write` after `Handle` returns. The hook runs once after that write, and also once when the write fails, the handler hijacks, or the connection ends without a write. |
| Version | Implemented on rawhttp branch `perf/body-release`, changelog section v0.2.7. Not tagged. v0.2.6 has no release hook. The framework still requires v0.2.6. |

`SetBodyRelease` turns off the keep-alive body cache for that response. The cache keys on the slice pointer.

## JSON buffer pool — measured, not adopted

`JSON` encoded with `json.NewEncoder` into a `sync.Pool` buffer and returned the buffer from the hook. Bytes matched `encoding/json.Marshal`, including `<`, `>`, and `&`. Buffers over 64 KiB were not returned. `APP_DEBUG` panicked on `Content` after release.

Same host, `go test -benchmem` (`Hello, World!` map):

| Path | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| `encoding/json.Marshal` | 866.7 | 688 | 5 |
| pooled `Encoder` | 1071 | 688 | 6 |

`Marshal` already pools its encode buffer and copies the result out. `NewEncoder` adds that copy plus an encoder allocation. The pooled path is slower, so `JSON` stays on `encoding/json.Marshal`. The framework module still requires rawhttp v0.2.6 and does not call `SetBodyRelease`.
