# Reservation confirmation — design decision (initial `main` implementation)

## Challenge wording

> Reservation TTL: Reservations must automatically expire after 60 seconds if not **“confirmed”**. Upon expiration, the reservation is permanently removed and cannot be interacted with, and the stock is returned to the “Available” pool.

## Question

Is there a reservation **confirmation** step that was forgotten in the first implementation?

## Answer on `main` (before `feat/reservation-confirmation`)

**Yes — the strict reading implies a separate confirm step.** The initial build did **not** forget it by accident; it **omitted** confirmation as a documented MVP assumption.

### What `main` implemented

| Behavior | Detail |
|----------|--------|
| Reserve | `POST /reservations` → `status = active`, `expires_at = now + 60s` |
| Confirm | **Not present** — no endpoint, no UI, no `confirmed` status |
| TTL | All `active` reservations expire; stock returned via worker |
| Release | `DELETE /reservations/{id}` cancels an active hold |

Every successful reserve is an **unconfirmed hold**. Because nothing can transition to `confirmed`, “expire if not confirmed” collapses to “expire every hold after 60 seconds,” which still satisfies TTL/stock-return mechanics but **not** the two-phase hold → confirm lifecycle.

### Why confirmation was deferred

1. The PDF does not define a confirm API, payload, or UI (only the word “confirmed”).
2. Spec Kit `spec.md` assumption #2: no separate confirm endpoint; `active` = unconfirmed.
3. Constitution marks payment/order confirmation as out of scope for the MVP.
4. Challenge focus: concurrency, idempotency, TTL, and dashboard — not checkout.

### Gaps vs strict interpretation

- No way for a user to **keep** stock past 60s without re-reserving.
- No `confirmed` state distinct from `active`.
- Reviewers may expect `POST /reservations/{id}/confirm` and expiration **only** for unconfirmed holds.

### Follow-up work

Branch **`feat/reservation-confirmation`** adds:

- `confirmed` status and `POST /api/v1/reservations/{id}/confirm`
- Expiration limited to `active` (unconfirmed) reservations
- Frontend **Confirm** action
- Tests and OpenAPI updates

See that branch and `specs/001-inventory-reservation/` updates for the executable plan.
