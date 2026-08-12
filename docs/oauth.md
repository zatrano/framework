# OAuth2 Authorization Server

Minimal built-in OAuth2 authorization server for first-party and trusted clients.

## Grants

| Grant | Notes |
|-------|--------|
| `authorization_code` | Requires **PKCE S256** (`code_challenge` / `code_verifier`). Issues access + refresh tokens. |
| `refresh_token` | Exchanges a refresh token for a new access token (refresh rotated). |
| `client_credentials` | Machine-to-machine; no refresh token. |

## Handlers

- `GET /oauth/authorize` — `AuthorizeHandler()` (query: `client_id`, `redirect_uri`, `user_id`, `scope`, `state`, `code_challenge`, `code_challenge_method=S256`)
- `POST /oauth/token` — `TokenHandler()`
- `POST /oauth/introspect` — `IntrospectHandler()`

Register clients with `Server.RegisterClient` (or load from the JSON store after a previous run).

## Environment

| Variable | Description |
|----------|-------------|
| `OAUTH_STORE_PATH` | Optional JSON file path. When set, clients, access tokens, and refresh tokens are persisted. When empty, storage is in-memory only. |
| `OAUTH_CLIENT_ID` / `OAUTH_CLIENT_SECRET` / `OAUTH_REDIRECT_URI` | Typical client app env for consuming an AS (not required by the server itself). |

Example:

```env
OAUTH_STORE_PATH=storage/framework/oauth.json
```

## PKCE (S256)

1. Client generates `code_verifier` and `code_challenge = BASE64URL(SHA256(verifier))`.
2. Authorize with `code_challenge` and `code_challenge_method=S256`.
3. Token exchange must include the matching `code_verifier`.

## Limits

- No OpenID Connect / ID tokens.
- No consent UI (caller supplies `user_id`).
- Authorization codes are memory-only (lost on restart); clients and tokens persist when `OAUTH_STORE_PATH` is set.
- Suitable as a small AS for apps that need code + PKCE + refresh; not a full identity provider suite.
