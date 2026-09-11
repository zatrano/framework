# __APP_NAME__

Empty ZATRANO application overlay stub (not produced by `zatrano new`).

`zatrano new` always generates HTML at `/` and JSON at `/api`. This tree exists only as the `add:*` stub baseline.

```bash
cp .env.example .env
go run ./cmd/app key:generate
go run ./cmd/app serve
```

Add optional packages from `github.com/zatrano/packages` with `go run ./cmd/app package:enable <name>`. That updates `bootstrap/enabled.go`, writes a blank-import in `bootstrap/addons.go`, runs `go get`, and merges that package's environment keys into `.env.example`. `bootstrap.App()` boots Enabled ∩ Imported.

A database is optional. After `package:enable database`, link a driver with `go run ./cmd/app db:setup --drivers=sqlite` (or `mysql`, `pgsql`, …).

HTTP is not served until Bootstrap completes (`Booted`). Until then the kernel responds `503`. Generated `/up` is process liveness after Bootstrap. `package:enable health` adds `/health` checks. `LifecycleProvider.Start` (workers) runs only after `Start()` (`Running`). `SIGINT`/`SIGTERM` on `serve` stop HTTP then providers in reverse order.
