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

### Latest test run (performance)

Captured on Docker Desktop (Apple Silicon), with Postgres already warm from a prior compose run.

**Backend** — `docker compose --profile test run --rm test`

```
=== RUN   TestConcurrentLastItem
--- PASS: TestConcurrentLastItem (0.03s)
=== RUN   TestConcurrentTenUnits
--- PASS: TestConcurrentTenUnits (0.03s)
=== RUN   TestReserveIdempotencyParallel
--- PASS: TestReserveIdempotencyParallel (0.02s)
=== RUN   TestReleaseIdempotency
--- PASS: TestReleaseIdempotency (0.02s)
=== RUN   TestExpirationReturnsStock
--- PASS: TestExpirationReturnsStock (0.02s)
PASS
ok  	github.com/flash-reservation/backend/test/integration	0.118s
```

| Test | Load | Result | Time |
|------|------|--------|------|
| `TestConcurrentLastItem` | 55 goroutines, 1 unit in stock | 1 success, no over-sell | **30 ms** |
| `TestConcurrentTenUnits` | 100 goroutines, 10 units in stock | 10 success / 90 rejections | **30 ms** |
| `TestReserveIdempotencyParallel` | 2 parallel POSTs, same key | 1 reservation, 1 stock decrement | **20 ms** |
| `TestReleaseIdempotency` | 2 DELETEs on same reservation | Stock returned once | **20 ms** |
| `TestExpirationReturnsStock` | TTL + expiration worker | Reserved → 0 after expire | **20 ms** |
| **Package total** | 5 tests vs real PostgreSQL | **5/5 PASS** | **118 ms** |

**Frontend** — `docker compose --profile test run --rm frontend-test`

```
 RUN  v3.2.4 /app

 ✓ src/utils/reservationTimer.test.ts (4 tests) 2ms
 ✓ src/components/InventoryList.test.tsx (2 tests) 138ms

 Test Files  2 passed (2)
      Tests  6 passed (6)
   Duration  889ms (transform 119ms, setup 77ms, collect 189ms, tests 139ms, environment 654ms, prepare 161ms)
```

| Suite | Tests | Test execution | Total (incl. jsdom) |
|-------|-------|----------------|---------------------|
| `reservationTimer.test.ts` | 4 | **2 ms** | — |
| `InventoryList.test.tsx` | 2 (happy path + insufficient stock) | **138 ms** | — |
| **Vitest run** | **6/6 PASS** | **139 ms** | **889 ms** |

Wall-clock for each compose command is typically **2–7 s** (image + container startup); the numbers above are in-container test time only.

### Test coverage (challenge rubric)

| Test | Location |
|------|----------|
| 50+ concurrent reserve, last item | `backend/test/integration/reservation_test.go` → `TestConcurrentLastItem` |
| 100 concurrent, 10 units | same → `TestConcurrentTenUnits` |
| Parallel idempotency (reserve) | same → `TestReserveIdempotencyParallel` |
| Double release idempotency | same → `TestReleaseIdempotency` |
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

| Phase | Approximate time |
|-------|------------------|
| Initial challenge (Spec Kit, backend, frontend, Docker, tests, docs) | ~4–5 hours |
| Follow-up (reservation confirm feature, chat/docs export, pushes) | ~30–60 minutes |
| **Reasonable total** | **~5–6 hours** |

Git activity on 2026-05-29 spans ~50 minutes of commit timestamps; the total above includes Docker builds, test runs, review, and conversation time not reflected in commits.

## Assumptions

- User identity via `X-User-Id` header (stored in browser `localStorage`).
- Frontend sync via **3-second polling** (no WebSocket).
- Reservation confirm is on `main` (`POST /api/v1/reservations/{id}/confirm`); only unconfirmed `active` holds auto-expire after 60s.
