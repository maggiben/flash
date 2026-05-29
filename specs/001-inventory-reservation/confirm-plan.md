# Plan: Reservation confirmation (`feat/reservation-confirmation`)

## Analysis

The challenge TTL rule implies two phases:

```text
reserve (active, 60s) ──confirm──> confirmed (no auto-expire) ──release──> released
         │                                    │
         └── expire (60s, no confirm) ────────┴──> expired (stock returned)
```

`main` collapsed both phases into `active` + auto-expire. This branch restores the distinction.

## API

| Method | Path | Effect |
|--------|------|--------|
| `POST` | `/api/v1/reservations/{id}/confirm` | `active` → `confirmed`; idempotent if already `confirmed` |

## Rules

1. Only `active` reservations can be confirmed (owner must match `X-User-Id`).
2. Expiration worker: `WHERE status = 'active' AND expires_at <= now()` only.
3. `GET /reservations` returns `active` and `confirmed` (user's current holds).
4. `DELETE` works on `active` and `confirmed` (returns stock once).
5. `confirmed` is a no-op on repeat confirm (200, same id).

## Schema

Migration `002_add_confirmed_status.sql`: extend status check to include `confirmed`.

## Tests

- Confirm before TTL → after 61s, stock still reserved.
- Unconfirmed `active` → expires, stock returned.
- Double confirm → idempotent.
- Cannot confirm `expired` / `released`.

## Frontend

- **Confirm** on `active` rows only.
- **Confirmed** badge (no countdown) on `confirmed` rows.
