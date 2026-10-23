# Adversarial review log

Findings from review of the `safe-by-construction` (framework) and
`rawhttp-hardening` (rawhttp) branches. Each row is one finding: the change
it concerns, the model that reported it, the issue, severity, and the
decision. A regression test is named when the finding was fixed.

Independent human review is pending; this log is AI-assisted review of the
diff, not a third-party audit.

## Method

After coding finished and before the test phase, two models that did not
write the code reviewed the security-critical diffs: request/response
lifetime, pooling removal, production boot checks, rawhttp framing, the
state machine, write-side sanitization, and resource limits. The prompt
asked for exploit scenarios (data leak, race, request smuggling, unbounded
memory, escape bypass), not style.

Attempted reviewers: GPT-5.6, Claude Sonnet, Gemini (Other Models quota
exhausted; those launches were not used). Completed reviewers:

- grok-4.7-high-fast (ids R-GRK-*)
- composer-2.5-fast (ids R-CMP-*)

## Findings

| Change | Model | Finding | Severity | Decision |
|--------|-------|---------|----------|----------|
| Chunked body cap `len(buf)+size` | grok-4.7 | R-GRK-1: a 16-digit chunk size of `7fffffffffffffff` wraps the signed add and skips `MaxRequestBodySize`. One connection can allocate gigabytes until `ReadTimeout`. Violates R2/R3. | high | Fixed. `chunkWouldExceed` rejects before adding. Same check on the stream reader and the client. Test: `TestChunkSizeMaxIntDoesNotBypassBodyLimit`. |
| Keep-alive encoded-body cache | grok-4.7 | R-GRK-2: `reset` left `cachedBodyPtr` and `outBuf` set. Two `SetBody` calls that share a backing array replay the first response. Stock framework `Handle` sets extra headers so it misses this path; a raw `rawhttp.Server` handler does not. | high | Fixed. `reset` clears `cachedBodyPtr`, `cachedCL`, `cachedCode`, `cachedDateOff`, and `outBuf`. Test: `TestKeepAliveBodyCacheDoesNotReplayPreviousBody`. |
| `X-Forwarded-Host` / `X-Forwarded-Proto` | grok-4.7, composer | R-GRK-3 / R-CMP-4: with a trusted peer, Host and Proto took the leftmost hop. A client that prepends `evil.example` or `https` wins. X-Forwarded-For already walked from the right. | medium | Fixed. `lastForwardedValue` takes the rightmost hop across every line. Test: `TestForwardedHostAndProtoUseRightmostHop`. |
| `Commit` file `Content-Disposition` | composer | R-CMP-8: `Commit` wrote `filename=`+`filepath.Base` unquoted. A backslash or quote in the name could break the quoted-string. `Download` already used `contentDisposition`. | medium | Fixed. `Commit` uses `contentDisposition`. Existing test: `TestContentDispositionCannotBreakQuoting`. |
| `Request.Ctx()` | composer | R-CMP-1: slices from the live `*rawhttp.Ctx` still alias the connection buffer. | high (known) | Accepted for v3. Deprecated, `nil` after the handler, doctor APP-HTTP-003/008. Removal is a v4 decision (`docs/V4-DECISIONS.md`). Not an I1/I2 break of the owned `Request` API. |
| Shared `*Response` body bytes | composer | R-CMP-2: `Own` shares the body backing array. | low | Accepted. `Content()` returns a copy; `Commit` only reads. Test: `TestContentReturnsACopyOfSharedBody`. |
| `TRUSTED_PROXIES=*` leftmost XFF | composer | R-CMP-3: TrustAll takes the leftmost hop. | high in that mode | Accepted. Production `Validate` refuses `*` / `0.0.0.0/0` / `::/0`. Development-only. |
| `Middleware(proxies)` skips `Validate` | composer | R-CMP-5 | medium | Accepted. Callers that need production policy use `FromEnv`. Documented. |
| Cookie `APP_URL` not https | composer | R-CMP-6 | medium | Accepted as I6 limit: the check reads `APP_URL`, not the live TLS state. |
| Production opt-out keys | composer | R-CMP-7 / R-CMP-17 | operational | Accepted. Each key is one check; there is no shared insecure mode. |
| Duplicate `Cookie` lines in rawhttp | composer | R-CMP-12: `Ctx.Cookie` keeps the last line; the framework arena visits every line. | medium | Accepted. Framework I1/I2 use the arena. rawhttp last-wins matches its indexed headers. Not a framing (R1) issue. |
| Remaining residual | both | Application auth, volumetric DDoS, dependency CVEs, independent human review, front-end proxy desync. | — | Listed under "Not guaranteed" in SECURITY-GUARANTEES.md. |
| `RedirectBack` | local regression | R-LOC-1: `url.Parse` percent-encodes `\`, so `/\evil.example` became path `/%5Cevil.example` and was treated as same-origin. | medium | Fixed. `safeBackURL` rejects `\` and `%5c` in the original Referer before parse. Test: `TestRedirectBackRejectsBackslashPaths`. |
