# Feature Specification: Inventory Reservation System

**Feature branch**: `001-inventory-reservation`  
**Created**: 2026-05-29  
**Status**: Approved for planning

## Problem Statement

In flash-sale scenarios, multiple users compete for limited stock within milliseconds. The system must hold inventory for a bounded time window without double-selling, while remaining safe under network retries and concurrent requests.

## User Stories

### US-1 — View inventory dashboard (P1)

As a shopper, I want to see each item's total, reserved, and available counts so I can decide what to reserve.

**Acceptance criteria**

- Given seeded inventory, when I open the dashboard, then each item shows name, total stock, reserved count, and available count (`total - reserved`).
- Given another user reserves stock, when I poll or refresh within 3 seconds, then available counts update without a manual page reload.

### US-2 — Atomic reservation (P1)

As a shopper, I want to reserve N units atomically so I am not charged availability that was already taken.

**Acceptance criteria**

- Given available stock ≥ N, when I POST a reservation with valid payload, then one reservation is created and available decreases by N.
- Given available stock < N, when I POST, then the request fails with a clear insufficient-stock error and no reservation is created.
- Given 50+ simultaneous requests for the last unit, when all complete, then exactly one succeeds and reserved + available always sums to total.
- Given 100 concurrent requests each for 1 unit when 10 remain, when all complete, then exactly 10 succeed and 90 fail with no negative available stock.

### US-3 — Reservation TTL (P1)

As the system, I must expire unconfirmed reservations after 60 seconds and return stock to the available pool.

**Acceptance criteria**

- Given an active reservation at T₀, when T₀ + 60s passes without confirmation, then the reservation becomes expired, is removed from active lists, and stock returns exactly once.
- Given an expired reservation that was fully processed, when a client attempts release or other interaction, then the API returns a well-defined not-found or no-op response.

### US-4 — Manual release (P1)

As a shopper, I want to cancel my reservation at any time, including when my UI timer may be out of sync with the server.

**Acceptance criteria**

- Given an active reservation I own, when I DELETE it, then stock returns once and it disappears from my active list.
- Given a reservation whose TTL elapsed but expiration worker has not yet run, when I DELETE, then release succeeds and stock returns once.
- Given an already-released reservation, when I DELETE again, then the call succeeds (idempotent no-op) without double-returning stock.

### US-5 — Idempotent reserve and release (P1)

As a frontend under unreliable networks, I need duplicate requests to be safe.

**Acceptance criteria**

- Given two parallel POST /reservations with the same `Idempotency-Key` and identical body, when both complete, then one reservation exists and stock decrements once.
- Given two POST requests with the same key but different bodies, when the second arrives, then it is rejected with HTTP 409 and a clear error.
- Given two DELETE calls for the same reservation, when both complete, then stock returns at most once.

### US-6 — Error feedback in UI (P2)

As a shopper, I want visible feedback for success, validation errors, conflicts, and loading states.

**Acceptance criteria**

- Reserve success shows confirmation; insufficient stock shows an inline error; loading indicators appear during async calls.
- Active reservations list shows countdown to expiry and a release button with feedback on success/failure.

## Edge Cases & Resolutions

| Edge case | Resolution |
|-----------|------------|
| Same idempotency key, different JSON field order | Hash canonical JSON (sorted keys) before compare |
| Expiration worker races manual release | Single `UPDATE … WHERE status IN ('active')` winner; loser is no-op |
| User releases after expiration processed | DELETE returns 200 with `already_released` or 404 — documented as no-op success |
| Clock skew on TTL display | UI uses server `expires_at`; timer is display-only |
| Reservation for zero or negative quantity | 400 validation error before touching DB |
| Unknown item id | 404 not found |
| User tries to release another user's reservation | 403 forbidden |
| DB connection lost mid-transaction | Transaction rolls back; client retries with same idempotency key |

## Assumptions (Documented Ambiguity)

1. **User identity**: `X-User-Id` header (UUID string) identifies the shopper; no login UI.
2. **Confirmation**: No separate confirm endpoint; "unconfirmed" means any reservation still in `active` status.
3. **Visual reference**: PDF references a design guideline not included in the document; UI follows a clean flash-sale dashboard pattern (card list, primary CTA, status badges).
4. **Polling interval**: 3 seconds for inventory and reservations sync.
5. **Repository naming**: Public repo must not include the company name from the challenge PDF.

## Success Metrics

- Zero over-reservation in automated concurrency tests.
- All idempotency tests pass under parallel execution.
- Frontend tests cover timer logic, reserve happy path, and insufficient-stock error path.
