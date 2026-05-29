# Confirmation feature — implementation summary

**Branch:** `feat/reservation-confirmation`  
**Related chat:** [`confirmation-chat-history.md`](./confirmation-chat-history.md)

## Commits on the feature branch

| Commit | Summary |
|--------|---------|
| `a6b72e9` | Plan, OpenAPI, decision doc update |
| `a2538fd` | Backend: `POST /api/v1/reservations/{id}/confirm`, migration `002`, `confirmed` status |
| `b195318` | Frontend: Confirm button + confirmed badge |
| `ed55455` | Tests: confirm prevents expiry, idempotent confirm, cannot confirm expired |

## Behavior after merge

```text
POST /reservations          → active (unconfirmed, expires in 60s)
POST /reservations/{id}/confirm → confirmed (no auto-expiry)
expiration worker         → active only, when expires_at <= now
DELETE /reservations/{id} → release active or confirmed (stock once)
```

## Test results (Docker)

```
TestConfirmPreventsExpiration  PASS
TestConfirmIdempotent          PASS
TestCannotConfirmExpired       PASS
(+ 5 existing integration tests)
ok  …/test/integration  0.157s
```

## API example

```bash
# Reserve
curl -s -X POST http://localhost:8080/api/v1/reservations \
  -H "Content-Type: application/json" \
  -H "X-User-Id: demo-user" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"item_id":"11111111-1111-1111-1111-111111111104","quantity":1}'

# Confirm (use id from response)
curl -s -X POST http://localhost:8080/api/v1/reservations/<RESERVATION_ID>/confirm \
  -H "X-User-Id: demo-user"
```
