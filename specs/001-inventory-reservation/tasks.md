# Tasks: Inventory Reservation System

**Input**: spec.md, plan.md, data-model.md, contracts/openapi.yaml  
**Organization**: Phased by user story; `[P]` = parallelizable within phase

## Phase 1 — Spec Kit & infrastructure (blocking)

- [x] T001 Create constitution in `.specify/memory/constitution.md`
- [x] T002 [P] Write spec.md with edge cases and assumptions
- [x] T003 [P] Write research.md and data-model.md
- [x] T004 Write plan.md and OpenAPI contract
- [x] T005 Generate tasks.md (this file)
- [x] T006 [P] Docker Compose: postgres, api, frontend, test profile
- [x] T007 [P] SQL migrations + seed data

## Phase 2 — Backend foundation

- [x] T008 Go module, Chi server skeleton, health endpoint
- [x] T009 Database pool (pgx), migration runner on startup
- [x] T010 Repository layer: items, reservations, idempotency

## Phase 3 — US-1 Inventory dashboard API (P1)

- [x] T011 GET /api/v1/inventory handler + integration test
- [x] T012 Map items to JSON (name, total, reserved, available)

## Phase 4 — US-2 Atomic reservation (P1)

- [x] T013 POST /reservations with transactional stock check
- [x] T014 Idempotency-Key storage and conflict detection
- [x] T015 TestConcurrentLastItem (50+ goroutines)
- [x] T016 TestConcurrentTenUnits (100 goroutines, 10 stock)
- [x] T017 TestReserveIdempotencyParallel

## Phase 5 — US-3 TTL expiration (P1)

- [x] T018 Expiration worker goroutine (5s ticker)
- [x] T019 Expiration integration test with injected clock

## Phase 6 — US-4 & US-5 Release + idempotency (P1)

- [x] T020 DELETE /reservations/{id} with status-gated stock return
- [x] T021 TestReleaseIdempotency (double delete)
- [x] T022 GET /reservations for active user reservations

## Phase 7 — Frontend (US-1, US-6)

- [x] T023 [P] Vite + React + TS scaffold, API client
- [x] T024 [P] TanStack Query hooks with 3s polling
- [x] T025 InventoryList component (reserve action, errors, loading)
- [x] T026 ReservationPanel (active list, release, timer)
- [x] T027 reservationTimer unit tests
- [x] T028 Component tests: happy path + insufficient stock

## Phase 8 — Documentation & submission

- [x] T029 README.md (concurrency, tests, LLM rationale)
- [x] T030 spec-kit-notes.md (commands, pivots, assumptions)
- [x] T031 Verify full stack via `docker compose up`

## Dependencies

```text
Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6 → Phase 7 → Phase 8
T015-T017 require T013-T014
T028 requires T025-T027
```

## Implementation strategy

1. Complete Phases 1–6 before frontend (backend contract stable).
2. Run Go tests in Docker after Phase 6.
3. Frontend against running API container.
4. Final documentation pass.
