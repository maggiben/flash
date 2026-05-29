# Flash Sale Inventory Reservation System

Atomic stock reservations with 60-second TTL, idempotent REST APIs, and a React dashboard. Built following the **Spec Kit** workflow (constitution → spec → plan → tasks → implementation).

## LLM used

**Cursor Agent (Claude)** — chosen for spec-driven iteration, multi-file scaffolding, and Docker-first verification without touching host services.

## Concurrency strategy

All stock changes run inside **PostgreSQL transactions**:

1. **Reserve**: `SELECT … FOR UPDATE` on the item row, then conditional  
   `UPDATE items SET reserved_quantity = reserved_quantity + $q WHERE … AND total_quantity - reserved_quantity >= $q`.  
   Zero rows updated → `INSUFFICIENT_STOCK` (HTTP 409).

2. **Idempotent reserve**: `pg_advisory_xact_lock(hashtext(idempotency_key))` serializes parallel retries; cached response replayed from `idempotency_keys`.

3. **Release / expire**: `UPDATE reservations SET status = … WHERE status = 'active'` — only the winning row returns stock (`reserved_quantity -= quantity`).

4. **Expiration worker**: Go goroutine every 5s marks overdue `active` reservations as `expired` and returns stock in one transaction.

This prevents over-reservation under 50+ concurrent goroutines (verified in integration tests).

## Prerequisites

- Docker & Docker Compose only (no local Go, Node, or PostgreSQL required on the host)

## Run the stack

```bash
docker compose up --build
```

| Service  | URL |
|----------|-----|
| Frontend | http://localhost:5173 |
| API      | http://localhost:8080/api/v1/inventory |
| OpenAPI  | http://localhost:8080/openapi.yaml |
| Health   | http://localhost:8080/health |

## Run tests (inside Docker)

```bash
# Go concurrency + idempotency integration tests
docker compose --profile test run --rm test

# React unit + component tests
docker compose --profile test run --rm frontend-test
```

### Test coverage (challenge rubric)

| Test | Location |
|------|----------|
| 50+ concurrent reserve, last item | `backend/test/integration/reservation_test.go` |
| 100 concurrent, 10 units | same |
| Parallel idempotency (reserve) | same |
| Double release idempotency | same |
| Timer logic | `frontend/src/utils/reservationTimer.test.ts` |
| Reserve happy path + insufficient stock UI | `frontend/src/components/InventoryList.test.tsx` |

## API quick examples

```bash
# Inventory
curl -s http://localhost:8080/api/v1/inventory | jq

# Reserve (replace ITEM_UUID)
curl -s -X POST http://localhost:8080/api/v1/reservations \
  -H "Content-Type: application/json" \
  -H "X-User-Id: demo-user" \
  -H "Idempotency-Key: $(uuidgen)" \
  -d '{"item_id":"11111111-1111-1111-1111-111111111105","quantity":1}'
```

## Spec Kit artifacts

All under `specs/001-inventory-reservation/` plus `.specify/memory/constitution.md`. See `spec-kit-notes.md` for workflow commands and pivots.

## Time spent

~4.5 hours (spec artifacts, backend, frontend, Docker, tests, documentation).

## Assumptions

- User identity via `X-User-Id` header (stored in browser `localStorage`).
- Frontend sync via **3-second polling** (no WebSocket).
- No separate “confirm” endpoint; TTL applies to all `active` reservations.
