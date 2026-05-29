# Project Constitution — Flash Sale Reservation System

## Purpose

Govern all implementation decisions for the inventory reservation challenge. This document is immutable during a feature cycle; changes require explicit amendment with rationale in `spec-kit-notes.md`.

## Core Principles

1. **Architecture first** — No application code until `spec.md`, `plan.md`, and `tasks.md` exist and are reviewed.
2. **Correctness over speed** — Zero over-reservation under concurrent load is non-negotiable.
3. **Idempotent APIs** — Reserve and release must be safe under retries and double-clicks.
4. **Traceability** — Every implementation unit maps to a task in `tasks.md`.
5. **Container-only execution** — All runtime and tests run inside Docker; no host database or bare-metal services.

## Technology Stack (Fixed)

| Layer      | Choice                          |
|------------|---------------------------------|
| Backend    | Go 1.23+, Chi router            |
| Database   | PostgreSQL 16                   |
| Frontend   | React 19, Vite 6, TypeScript 5  |
| Testing    | Go `testing` + testify; Vitest + Testing Library |
| API docs   | OpenAPI 3.1                     |

## Concurrency Strategy (Mandatory)

- All stock mutations occur inside PostgreSQL transactions.
- Reservation increments use a single atomic `UPDATE … WHERE available >= quantity` with row lock via `FOR UPDATE` when reading item state in the same transaction.
- Expiration and manual release use status transitions with `UPDATE … WHERE status = 'active'` so each reservation affects stock at most once.
- Idempotency keys are persisted before side effects complete; duplicate keys return cached responses.

## API Conventions

- REST under `/api/v1`.
- `Idempotency-Key` header required on `POST /reservations`.
- `X-User-Id` header identifies the caller (no auth scope in MVP; documented assumption).
- JSON error bodies: `{ "error": { "code": "...", "message": "..." } }`.

## Frontend Conventions

- Component composition with explicit loading/error/success states.
- Poll inventory and reservations every 3 seconds (no WebSocket in MVP).
- Reservation countdown derived from server `expires_at` (not client-only timers).

## Testing Requirements

- Go: 50+ concurrent reserve for last unit; 100 concurrent for 10 units; idempotency for reserve and release.
- React: timer logic unit tests; component tests for happy path and insufficient-stock error.

## Out of Scope

- Payment / order confirmation flow (reservations expire if not confirmed; confirm endpoint not required).
- Multi-tenant authentication.
- Kubernetes deployment.
