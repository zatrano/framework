# __APP_NAME__

ZATRANO application overlay (scaffold __SCAFFOLD_NAME__).

JSON routes and `health` + `validation`. Folders such as `app/views` are canonical placeholders. `AGENTS.md` is a describe dump for agents. Framework docs are not copied into the app.

```bash
cp .env.example .env
go run ./cmd/app key:generate
go run ./cmd/app serve
```

Add optional packages from `github.com/zatrano/packages` with `go run ./cmd/app package:enable <name>`. That updates `bootstrap/enabled.go`, writes a blank-import in `bootstrap/addons.go`, runs `go get`, and merges that package's environment keys into `.env.example`. `bootstrap.App()` boots Enabled ∩ Imported.

A database is optional. After `package:enable database`, link a driver with `go run ./cmd/app db:setup --drivers=sqlite` (or `mysql`, `pgsql`, …).

HTTP is not served until Bootstrap completes (`Booted`). Until then the kernel responds `503`. Generated `/up` is process liveness after Bootstrap. Enabled `health` adds `/health` and `/api/health`. `LifecycleProvider.Start` (workers) runs only after `Start()` (`Running`). `SIGINT`/`SIGTERM` on `serve` stop HTTP then providers in reverse order.
