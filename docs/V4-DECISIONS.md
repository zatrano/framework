# v4 decisions

Breaking changes that the `safe-by-construction` work prepares but does not make in v3. Each item lists what v3 does now, what v4 should do, and why the change waits for a major version.

## Remove `Request.Ctx()`

- **v3 now:** deprecated. Returns the raw `*rawhttp.Ctx` while the handler runs and `nil` after the handler returns. `zatrano doctor` reports every call: APP-HTTP-003 (error) under `app/http/handlers`, APP-HTTP-008 (warning) elsewhere.
- **v4:** remove the method. Nothing in the framework calls it.
- **Why it waits:** removing an exported method breaks compilation for every caller. Every value the raw context offers has an owned `Request` method (`Method`, `Path`, `QueryString`, `Header`, `HeadersMap`, `Cookie`, `Body`, `RemoteIP`, `Secure`, `Host`), so the migration is mechanical.

## Remove the pool API

- **v3 now:** `SetRequestPooling`, `RequestPooling`, `AcquireRequest`, `ReleaseRequest`, `ReleaseResponse`, `ReleaseAfterCommit`, and `Request.Generation` are deprecated no-ops. `HTTP_POOL_REQUESTS` is ignored with a boot warning.
- **v4:** delete the functions and the warning.
- **Why it waits:** application and benchmark code calls them; deleting them breaks builds without changing behaviour.

## Remove `Request.PathBytes()`

- **v3 now:** deprecated; returns a fresh copy of the path.
- **v4:** delete it. `Path()` returns the same bytes as a string without a copy.

## Do not reintroduce pooling or zero-copy views

Request and response pooling and views into the connection buffer were removed because each one made a retained pointer read another request. A v4 performance change must keep invariants I1–I3 in `docs/SECURITY-GUARANTEES.md` and their tests. Allocation reductions that keep ownership (arena sizing, fewer header copies, inline storage) are in scope; reuse across requests is not.

## Make `Own` an internal detail

- **v3 now:** `Request.Own` and `Response.OwnedBy` are exported so custom middleware composers can follow the same rule as the router.
- **v4:** consider making them unexported once the router is the only composer. Exporting them costs nothing today, and removing them is breaking.

## Opt-out keys

The production refusals added in this branch each have their own opt-out key (`APP_DEBUG_IN_PRODUCTION`, `APP_KEY_WEAK_IN_PRODUCTION`, `COOKIE_INSECURE_IN_PRODUCTION`, `CSRF_DISABLED`, `CORS_NULL_ORIGIN_WITH_CREDENTIALS`). v4 should keep them separate. A shared "insecure mode" switch is out of scope.
