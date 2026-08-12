<p align="center">
  <strong>ZATRANO</strong>
</p>

<p align="center">
  <em>Go performance. Thin kernel. Opt-in packages. One opinionated path from request to production.</em>
</p>

<p align="center">
  <a href="https://github.com/zatrano/framework/actions"><img src="https://github.com/zatrano/framework/actions/workflows/tests.yml/badge.svg" alt="Tests"></a>
  <a href="https://github.com/zatrano/framework/actions"><img src="https://github.com/zatrano/framework/actions/workflows/static-analysis.yml/badge.svg" alt="Static Analysis"></a>
  <a href="https://github.com/zatrano/framework/actions"><img src="https://github.com/zatrano/framework/actions/workflows/coding-style.yml/badge.svg" alt="Coding Style"></a>
  <a href="https://pkg.go.dev/github.com/zatrano/framework"><img src="https://img.shields.io/badge/go-1.25+-00ADD8?logo=go&logoColor=white" alt="Go"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="VERSION"><img src="https://img.shields.io/badge/version-0.2.5-green.svg" alt="Version"></a>
</p>

<p align="center">
  <a href="https://zatrano.com/docs">Documentation</a> ·
  <a href="https://zatrano.com/docs/boot-profiles">Boot Profiles</a> ·
  <a href="https://zatrano.com/docs/package-ecosystem">Packages</a> ·
  <a href="https://github.com/zatrano/framework">GitHub</a>
</p>

## Why ZATRANO?

Most Go web apps start fast and then slow down in the glue: auth, migrations, queues, mail, validation, sessions, CLI, and project structure. ZATRANO ships those as first-party packages under a **thin kernel** — you choose how much boots.

- **Thin `core/`** — Application, container, catalog, secure HTTP hooks
- **Foundation** — DB, auth, mail, session, cache, queue, views when you need a web/API stack
- **Opt-in addons** — mongo, oauth, billing, … via `EnabledAddons` / presets / `APP_BOOT`
- **Full Auth** — guards, remember me, password reset, email verification, lockout, MFA, trusted devices, multi-device logout

## Quick start

```bash
git clone https://github.com/zatrano/framework.git
cd framework
cp .env.example .env
go mod tidy
go run ./cmd/zatrano key:generate

# Lean production-style boot (optional)
# echo APP_BOOT=api >> .env
# go run ./cmd/zatrano package:init api

go run ./cmd/zatrano serve
```

Open [http://localhost:8080](http://localhost:8080).

`cmd/zatrano` defaults to the **demo** profile when `APP_BOOT` is unset so exploration works. For production set `APP_BOOT=app|api|web|minimal`.

```bash
go get github.com/zatrano/framework@latest
```

## Architecture

```text
core/                 Thin kernel
bootstrap/            Boot profiles, foundation, EnabledAddons, APP_BOOT
packages/             First-party packages (auth, orm, mail, mongo, …)
app/ · routes/ · …    Your application
```

| Profile | `APP_BOOT` | Boots |
|---------|------------|-------|
| `CoreApp` | `core` | Kernel only |
| `MinimalApp` | `minimal` | Foundation, no addons |
| `App` | `app` | Minimal + `EnabledAddons` |
| `APIApp` / `WebApp` | `api` / `web` | Lean presets |
| `DemoApp` | `demo` | Full demo addons |

```go
app := bootstrap.FromEnv()       // respects APP_BOOT
app := bootstrap.APIApp()
auth.From(app)                   // resolve services — not app.Auth()
```

```bash
go run ./cmd/zatrano package:list
go run ./cmd/zatrano package:init api
go run ./cmd/zatrano package:doctor
```

## Documentation

Guides are **only** on the website — there is no `docs/` folder in this repository.

**[https://zatrano.com/docs](https://zatrano.com/docs)**

Start with:

1. [Installation](https://zatrano.com/docs/installation)
2. [Boot Profiles](https://zatrano.com/docs/boot-profiles)
3. [Package Ecosystem](https://zatrano.com/docs/package-ecosystem)
4. [Resolving Services](https://zatrano.com/docs/accessors)
5. [Authentication](https://zatrano.com/docs/authentication)

Optional addons (MongoDB, OAuth server, Billing, AI, WebAuthn, …) are documented under [Packages](https://zatrano.com/docs/package-ecosystem) on the same site.

## What you get

| Area | Capabilities |
|------|----------------|
| **HTTP** | Router, controllers, middleware, form/JSON input, cookies, flash, URLs |
| **Views** | Layouts, components, Blade-like directives, markdown, file-based pages |
| **Auth & security** | Session guards, 2FA + trusted devices, lockout, OAuth/WebAuthn/social, encryption, hashing |
| **Data** | Query builder, schema, migrations, ORM, factories, seeders, Mongo addon |
| **Async & mail** | Queues, notifications, broadcasting, scheduler |
| **Platform** | Config + `.env`, sessions, cache, filesystem, localization, validation |
| **Tooling** | CLI (`serve`, `migrate`, `make:*`, `package:*`), OpenAPI, health, docs engine |

## Who is ZATRANO for?

- Teams that want **Go speed** without inventing scaffolding on every project
- Builders shipping **server-rendered apps and APIs** with shared conventions
- Engineers who prefer **batteries included** with **explicit opt-in** for heavy addons

## Roadmap

| Version | Focus |
|---------|--------|
| **v0.2** | Package ecosystem, docs depth, auth parity, starter polish |
| **v0.3** | Production recipes (deploy, observability, scaling) |
| **v1.0** | Stable public API and long-term support |

Current release: **[v0.2.5](https://github.com/zatrano/framework/releases/tag/v0.2.5)** (module line; ecosystem docs describe the current kernel/packages layout).

## Community

- Documentation: [zatrano.com/docs](https://zatrano.com/docs)
- GitHub: [github.com/zatrano/framework](https://github.com/zatrano/framework)
- LinkedIn: [linkedin.com/company/zatrano](https://www.linkedin.com/company/zatrano)

## Contributing

Open an issue or pull request on GitHub. Keep changes focused, include tests when practical, and follow existing style (`gofmt`, clear package boundaries). See the [Contribution Guide](https://zatrano.com/docs/contributions).

## Code of Conduct

Be respectful in issues, pull requests, and discussions. Harassment and discrimination are not tolerated.

## Security Vulnerabilities

If you discover a security vulnerability within ZATRANO, email Serhan KARAKOÇ at [serhankarakoc@zatrano.com](mailto:serhankarakoc@zatrano.com). All reports will be promptly addressed.

## License

The ZATRANO framework is open-sourced software licensed under the [MIT license](LICENSE).

Copyright (c) 2026 Serhan KARAKOÇ.
