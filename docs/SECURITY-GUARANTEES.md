# ZATRANO security guarantees

This file lists the properties the framework is built to hold, the tests that
check each one, and where each one stops. A test shows a property for the
inputs it covers; it does not prove it for every input. HTTP parsing and
framing belong to rawhttp; its properties (R1–R4) are in
`github.com/zatrano/rawhttp/docs/SECURITY-GUARANTEES.md`.

The latest run of every test named here is recorded in
[REVIEW-LOG.md](REVIEW-LOG.md) and in the pull request that changed it.

## I1: a retained `*Request` never returns another request's data

The kernel copies the request line, headers, peer address, TLS state and body
into a per-request arena before any handler runs. Route parameters are
substrings of that copy. After the handler returns, the request no longer
references the rawhttp connection context. A `*Request` kept in a goroutine,
a cache or a closure returns its own values for as long as it is referenced.

Tests:

- `TestRetainedRequestKeepsItsOwnData`: 300 keep-alive requests with different
  field offsets, a concurrent reader of every retained request, then each
  retained request is checked field by field.
- `TestRequestUsedAfterResponseByGoroutine`
- `TestRouteParamSurvivesCarrierBufferReuse`
- `TestReleasedRequestAndResponseStayReadable`
- `TestCanaryRequestIsolation` (`tests/safety`): the full stack under
  pipelined keep-alive load, with and without `-race`. The release gate runs
  10,000,000 requests (`ZATRANO_CANARY=1`).

Latest local run, 2026-10-23, Windows/amd64, Go 1.26.9, `-race` off:
10,000,000 requests served, 0 mismatches, 9,028 retained requests re-read
after later traffic, 12m5s (13,787 req/s). `-race` was not run on this host
(`-race requires cgo` and no C compiler is installed). The `safety` workflow
runs the same gate with race off and race on on ubuntu-24.04; that job starts
after the pull request is open and had not been started when this file was
written.

Limits:

- `Request.Ctx()` is deprecated. While the handler runs it returns the live
  `*rawhttp.Ctx`, and slices read from it alias the connection buffer. It
  returns `nil` after the handler returns. `zatrano doctor` reports each call
  (APP-HTTP-003 in handlers, APP-HTTP-008 elsewhere). It is removed in v4.
- Two goroutines that call `Set`, `SetHeader` or other mutators on the same
  `*Request` at once race with each other. Isolation is between requests, not
  between goroutines sharing one request.

## I2: no returned string or `[]byte` aliases memory that the engine or another request writes

Strings are built from the arena; byte slices are the request's own copies.
`Body()` returns the request's body copy; callers that change it change what
later `Body()` calls on the same request return, nothing else.

Tests: `TestBodyOwnedCopyDoesNotMutateRawHTTPBuffer`,
`TestBindJSONDoesNotAliasDecodedStrings`, `TestRetainedRequestKeepsItsOwnData`
(under `-race`, a shared buffer would be reported), `TestCanaryRequestIsolation`.

Limits: the same `Ctx()` exception as I1. Values a handler stores in
`Request.Set` are whatever the handler stored.

## I3: there is no unsafe mode

Request and response pooling is gone. `HTTP_POOL_REQUESTS` is ignored; when it
is set, boot logs one deprecation warning. `AcquireRequest`, `ReleaseRequest`
and the other pool functions remain for source compatibility and never reuse
an object.

Tests: `TestHTTPPoolRequestsIsDeprecatedAndIgnored`,
`TestHTTPPoolRequestsDoesNotEnablePooling`, `TestDeprecatedPoolAPINeverReuses`,
`TestReleasedRequestAndResponseStayReadable`.

Limits: none known. The deprecated functions are removed in v4
([V4-DECISIONS.md](V4-DECISIONS.md)).

## I4: the engine never writes to a `*Response` a handler or middleware returned

`Request.Own` copies a response before the kernel, the bridge or a middleware
boundary adds headers or cookies. Every copy is owned by one request. A
response kept in a package variable or a cache keeps its status, headers,
cookies and body, and no request sees another request's cookie.

Tests: `TestSharedCachedResponseIsNeverMutated` (a global cached response and a
group middleware that memoises `next(req)`, decorated per request by an outer
middleware and the bridge, over 8 concurrent keep-alive connections), and the
cached route in `TestCanaryRequestIsolation`.

Limits: a middleware that mutates the `*Response` it received from `next`
mutates that object; when `next` returned a shared response, the copy made at
the inner boundary is what it mutates. Code that calls `Header` on a shared
response itself, outside the handler chain, changes that response.

## I5: `unsafe` and `//go:linkname` appear only in allowlisted packages

`TestUnsafeOnlyInAllowlistedPackages` parses every Go file in the repository,
tests and nested modules included, and fails on an `unsafe` import or a
`//go:linkname` outside the allowlist. The allowlist is empty.
`TestUnsafeAllowlistScannerSeesImports` checks the scanner itself.

Limits: dependencies are not scanned here. rawhttp runs its own check
(`TestUnsafeOnlyInAllowlistedFiles`, two files).

## I6: an insecure production configuration fails at boot

With `APP_ENV=production`, boot stops with an error naming the setting and,
where there is one, the key that accepts the risk. Each opt-out covers one
check only.

| Check | Opt-out | Test |
|-------|---------|------|
| `APP_KEY` empty, a known default, or the wrong length | none | `TestProductionWeakAppKeyOptOutKeepsLengthCheck` and the existing secrets tests |
| `APP_KEY` with fewer distinct bytes than half its length | `APP_KEY_WEAK_IN_PRODUCTION=allow` | `TestProductionRejectsWeakAppKey`, `TestProductionWeakAppKeyOptOut` |
| `APP_DEBUG` on (debug now defaults to off in production) | `APP_DEBUG_IN_PRODUCTION=allow` | `TestProductionRejectsDebug`, `TestProductionDebugOptOut`, `TestProductionDebugDefaultsOff`, `TestLocalDebugDefaultsOn` |
| `COOKIE_SECURE` or `SESSION_SECURE` false while `APP_URL` is https | `COOKIE_INSECURE_IN_PRODUCTION=allow` | `TestProductionRejectsInsecureCookiesOverHTTPS`, `TestProductionInsecureCookieOptOut`, `TestProductionPlainHTTPCookiesBoot` |
| A session bridge without CSRF middleware | `CSRF_DISABLED=true` | `TestProductionRejectsSessionsWithoutCSRF` |
| CORS credentials with `*` origin | none | `TestCORSRejectsCredentialsWithAnyOrigin` |
| CORS credentials with the `null` origin | `CORS_NULL_ORIGIN_WITH_CREDENTIALS=allow` | `TestCORSRejectsCredentialsWithNullOrigin` |
| `TRUSTED_PROXIES=*`, `0.0.0.0/0` or `::/0` | none | `TestProductionRejectsTrustAllProxyNetworks` |

`TestProductionBaselineBoots` checks that a correct production configuration
boots. `TestForwardedHeadersIgnoredWithoutTrustedProxies` checks that forwarded
headers are ignored when `TRUSTED_PROXIES` is empty. The CSRF middleware no
longer accepts the CSRF cookie as the token (`TestCSRFCookieAloneIsNotAToken`,
`TestCSRFHeaderTokenStillAccepted`).

The three checks without an opt-out already refused to boot before this change;
adding a switch would weaken them.

Limits:

- The checks read configuration, not behaviour. A handler that disables a
  protection at run time is not seen.
- The CSRF check knows that a CSRF middleware was composed in this process;
  it does not check which routes it covers. `CSRFExcept` paths are trusted as
  configured.
- `packages/session` sets its own cookie flags; it is outside this repository
  and its `Secure` default is an open item (see REVIEW-LOG.md).

## Not guaranteed

- **Application logic and authorization.** The framework cannot tell whether a
  user may call a route, see a record or change it. Missing or wrong
  authorization checks are application bugs.
- **Volumetric DDoS.** Body budgets and rawhttp limits bound the cost of each
  request. A flood that fills the network link or the connection slots needs
  upstream filtering.
- **Vulnerabilities in dependencies.** The framework depends on rawhttp, the
  Go standard library and the modules in `go.mod`. Their fixes arrive through
  updates; run `govulncheck ./...` and build with a supported Go patch release.
- **Independent human review.** The properties above are checked by the tests
  named here and by AI-assisted adversarial review
  ([REVIEW-LOG.md](REVIEW-LOG.md)). No independent third party has reviewed
  them; that review is pending.
