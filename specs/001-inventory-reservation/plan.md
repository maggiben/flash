# Implementation Plan: Inventory Reservation System

## Architecture Overview

```text
┌─────────────┐     poll 3s      ┌──────────────┐
│  React SPA  │ ◄──────────────► │  Go API      │
│  (Vite)     │   REST + headers │  (Chi)       │
└─────────────┘                  └──────┬───────┘
                                        │
                                 ┌──────▼───────┐
                                 │ PostgreSQL   │
                                 │ + migrations │
                                 └──────────────┘
        ┌──────────────────────────────────────┐
        │ Expiration worker (goroutine, 5s)    │
        └──────────────────────────────────────┘
```

## Docker topology

| Service   | Image / build      | Port | Role |
|-----------|--------------------|------|------|
| postgres  | postgres:16-alpine | 5432 | Data store |
| migrate   | backend (one-shot) | —    | Apply SQL migrations + seed |
| api       | backend            | 8080 | REST API + expiration worker |
| frontend  | frontend           | 5173 | Vite dev server (proxy to api) |
| test      | backend (profile)  | —    | `go test ./...` against postgres |

All services share a Docker network. Host filesystem is not modified beyond the project directory.

## API surface

See `contracts/openapi.yaml`. Summary:

- `GET /api/v1/inventory`
- `POST /api/v1/reservations` (+ `Idempotency-Key`, `X-User-Id`)
- `GET /api/v1/reservations` (+ `X-User-Id`)
- `DELETE /api/v1/reservations/{id}` (+ `X-User-Id`)

## Concurrency implementation

### Reserve transaction

1. Begin transaction.
2. If `Idempotency-Key` present: `SELECT` from `idempotency_keys FOR UPDATE`; return cached if hash matches; 409 if hash differs.
3. `SELECT * FROM items WHERE id = $1 FOR UPDATE`.
4. `UPDATE items SET reserved_quantity = reserved_quantity + $q WHERE id = $1 AND total_quantity - reserved_quantity >= $q RETURNING *`.
5. If 0 rows: rollback with insufficient stock error.
6. `INSERT INTO reservations (… status='active', expires_at=now()+60s)`.
7. Store idempotency record; commit.

### Release transaction

1. Begin transaction.
2. `SELECT * FROM reservations WHERE id = $1 FOR UPDATE`.
3. If not found: commit no-op 404 or 200 with `already_released` (choose 200 idempotent success per spec).
4. If status != `active`: return 200 no-op (already released/expired).
5. Verify `user_id` matches caller else 403.
6. `UPDATE reservations SET status='released' WHERE id = $1 AND status='active'`.
7. If row updated: `UPDATE items SET reserved_quantity = reserved_quantity - quantity WHERE id = item_id`.
8. Commit.

### Expiration worker

Every 5 seconds:

```sql
UPDATE reservations SET status = 'expired', updated_at = now()
WHERE status = 'active' AND expires_at <= now()
RETURNING id, item_id, quantity;
```

For each returned row, decrement `items.reserved_quantity` in the same transaction batch.

## Frontend structure

```text
src/
  api/client.ts          # fetch wrapper, headers
  hooks/useInventory.ts  # TanStack Query, refetchInterval 3000
  hooks/useReservations.ts
  components/InventoryList.tsx
  components/ReservationPanel.tsx
  components/ReservationTimer.tsx
  utils/reservationTimer.ts  # pure timer logic (tested)
  App.tsx
```

## Testing strategy

### Go (integration against real PostgreSQL in Docker)

- `TestConcurrentLastItem`: 50 goroutines, 1 available → 1 success.
- `TestConcurrentTenUnits`: 100 goroutines, 10 available → 10 success, 90 failures.
- `TestReserveIdempotencyParallel`: same key × 2 parallel → 1 reservation.
- `TestReleaseIdempotency`: release twice → stock +1 total not +2.

### React

- `reservationTimer.test.ts`: remaining seconds, expired state.
- `InventoryList.test.tsx`: happy path reserve mock.
- `InventoryList.test.tsx`: insufficient stock error display.

## Directory layout

```text
/
├── .specify/memory/constitution.md
├── specs/001-inventory-reservation/
├── backend/
├── frontend/
├── docker-compose.yml
├── seeds/seed.sql
├── README.md
└── spec-kit-notes.md
```

## Rollout

1. `docker compose up --build` brings full stack.
2. `docker compose --profile test run --rm test` runs Go tests.
3. `docker compose run --rm frontend-test` runs Vitest.

## Time estimate

| Phase | Duration |
|-------|----------|
| Spec Kit artifacts | ~45 min |
| Backend + tests | ~2 h |
| Frontend + tests | ~1.5 h |
| Docker + docs | ~45 min |
| **Total** | **~4.5 h** |
