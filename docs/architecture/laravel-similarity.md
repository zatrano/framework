# Laravel similarity audit

ZATRANO applications use familiar application folders. That is **intentional ergonomics**, not a port.

Do not rename folders to look unlike Laravel. Do not copy Laravel behavior because a name matches.

| Similarity | Verdict | Notes |
|---|---|---|
| `app/http/controllers` | Intentional, beneficial | Matches `make:controller` and doctor layout |
| `app/http/requests` FormRequest | Intentional, beneficial | Real package contract (`Rules`/`Authorize`) |
| `app/providers` Register/Boot | Intentional, beneficial | `contracts.Provider` |
| `app/models` + Eloquent-like query names | Intentional, **misleading if assumed identical** | Custom ORM: loader funcs not string `With`; no nested TX; `sql.ErrNoRows` |
| `make:auth` | Intentional, beneficial | User model is an **app stub**, not a framework User type |
| `csrf.Except("/api")` | Intentional | Kernel CSRF, not a package |
| Facades `Auth::` / `app.Auth()` | **Not ZATRANO** | `auth.From(app)` only |
| Blade / `@csrf` as PHP | **Misleading** | Go templates; view package directives are not Blade |
| Artisan | **Misleading** | `zatrano` CLI; acquire/enable are not `composer require` |
| Eloquent `$casts` / `$with` | **Misleading** | `make:cast`; eager is typed funcs |
| Policies + Gates | Intentional, beneficial | Real package |
| Spatie roles | Accidental stub gravity | Dashboard files ≠ AuthZ API |
| Resources / API Resources | Optional, misleading if required | Default is `http.JSON` |
| Mailables | **Not present** | `notification` channels |
| HTMX / Livewire | **Not present** | Views only |
| Migrations as PHP classes | **Misleading** | Go migration registration via database package |
| `config/*.php` | **Misleading** | `.env` + ConfigRepository + package env merge |
| Service container autowire | **Not present** | Factory binds, no reflection |

**Beneficial:** application-shaped tree so humans find HTTP, routes, and views quickly.

**Misleading:** query API names that look like Eloquent; FormRequest that does not implement nested Laravel-style `sometimes`/`Rule::` objects; no `App` service accessors.
