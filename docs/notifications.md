# Notifications

Channels: database (inbox), mail, SMS, broadcast, and push.

## Push drivers

| `PUSH_DRIVER` | Behavior |
|---------------|----------|
| `memory` (default) | Records deliveries in `MemoryPushSender` (tests/demos). |
| `http` | `HTTPPushSender` — `POST` JSON `{"token","payload"}` to `PUSH_URL` with `Authorization: Bearer PUSH_TOKEN`. |

```env
PUSH_DRIVER=http
PUSH_URL=https://push.example.com/send
PUSH_TOKEN=secret
```

## SMS drivers

See `SMS_DRIVER` (`memory`, `log`, `http`, `twilio`) and related `SMS_*` / `TWILIO_*` variables in `.env.example`.

## Channels

Configure defaults with `NOTIFICATION_CHANNELS`. Per-message channel lists override when sending via the manager or `/api/notifications` endpoints.
