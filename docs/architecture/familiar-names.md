# Familiar names audit

ZATRANO applications use familiar application folders. That is **intentional ergonomics**, not a port of another product.

Do not rename folders just to look unfamiliar. Do not copy another product's behavior because a folder name matches.

| Similarity | Verdict | Notes |
|---|---|---|
| `app/http/controllers` | Intentional, beneficial | Matches `make:controller` and doctor layout |
| `app/http/requests` FormRequest | Intentional, beneficial | Real package contract (`Rules`/`Authorize`) |
| `app/providers` Register/Boot | Intentional, beneficial | `contracts.Provider` |
| `app/models` + query names | Intentional, **misleading if assumed identical** | First-party ORM: loader funcs not string `With`; no nested TX; `sql.ErrNoRows` |
| `make:auth` | Intentional, beneficial | User model is an **app stub**, not a framework User type |
| `csrf.Except("/api")` | Intentional | Kernel CSRF, not a package |
| `Auth::` / `app.Auth()` | **Not ZATRANO** | `auth.From(app)` only |
| View `@csrf` as another template language | **Misleading** | Go templates; view package directives are ZATRANO |
| CLI as another ecosystem's generator | **Misleading** | `zatrano` CLI; acquire/enable are not a second module installer |
| Model `$casts` / `$with` | **Misleading** | `make:cast`; eager is typed funcs |
| Policies + Gates | Intentional, beneficial | Real package |
| Dashboard roles files | Accidental stub gravity | Dashboard files ≠ AuthZ API |
| Resources / API Resources | Optional, misleading if required | Default is `http.JSON` |
| Mailables | **Not present** | `notification` channels |
| Fragment / live-UI addons | **Not present** | Views only |
| Migrations as foreign-language classes | **Misleading** | Go migration registration via database package |
| `config` in another language | **Misleading** | `.env` + ConfigRepository + package env merge |
| Service container autowire | **Not present** | Factory binds, no reflection |

**Beneficial:** application-shaped tree so humans find HTTP, routes, and views quickly.

**Misleading:** query API names that look like another ORM; FormRequest that does not implement nested `sometimes` / fluent rule objects; no `App` service accessors.
