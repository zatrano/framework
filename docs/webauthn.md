# WebAuthn

Passkey / WebAuthn ceremonies via [go-webauthn/webauthn](https://github.com/go-webauthn/webauthn).

## Configuration

| Variable | Description |
|----------|-------------|
| `WEBAUTHN_RP_ID` | Relying Party ID (e.g. `localhost` or `example.com`). **Required**. |
| `WEBAUTHN_RP_ORIGIN` | Allowed origin (e.g. `http://localhost:8080`). **Required**. |
| `WEBAUTHN_RP_DISPLAY_NAME` | Display name (falls back to `WEBAUTHN_RP_NAME` / `APP_NAME`). |

If RP ID or origin is missing, `BeginRegistration` / `BeginLogin` / finish methods return a clear error. There is no accept-any stub mode.

## API

```go
m := app.WebAuthn()
opts, err := m.BeginRegistration(userID, userName, displayName)
// send opts.Options to navigator.credentials.create; keep opts.ChallengeID
cred, err := m.FinishRegistration(opts.ChallengeID, responseJSON)

req, err := m.BeginLogin(userID)
ok, err := m.FinishLogin(req.ChallengeID, assertionJSON)
```

Credentials are kept in an in-memory store by default. Replace with `SetStore` implementing `CredentialStore` for persistence.

## Limits

- Full cryptographic verification when configured; end-to-end browser tests need a real authenticator or soft authenticator.
- In-memory credential store unless you plug in your own `CredentialStore`.
