# Octane helpers

Octane in ZATRANO is **metrics and process helpers**, not a separate multi-process runtime (no worker pool process manager, no Swoole-style supervisor).

## What is real

- **Metrics middleware** — counts total requests, in-flight concurrency, and peak in-flight. Registered on the HTTP stack when Octane is bootstrapped.
- **`octane:start`** — boots the app, optionally sets `GOMAXPROCS` from `--workers` / `OCTANE_WORKERS`, then serves with the normal `Application.Run` HTTP server.
- **Stats handler** — JSON snapshot of workers hint, `GOMAXPROCS`, request counters, uptime.

## Environment

```env
OCTANE_WORKERS=0   # 0 = use NumCPU as the worker hint
```

## What it is not

- Not a multi-process application server.
- Does not replace the standard Go HTTP server with a custom event loop.
