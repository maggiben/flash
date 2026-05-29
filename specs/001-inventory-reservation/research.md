# Research: Inventory Reservation System

## Concurrency approaches evaluated

| Approach | Pros | Cons | Decision |
|----------|------|------|----------|
| Application mutex (Go sync.Mutex) | Simple | Fails across multiple instances; not durable | Rejected |
| Redis DECR | Fast | Extra infra; not required by spec | Rejected |
| PostgreSQL conditional UPDATE | ACID, single source of truth, works multi-instance | Requires careful transaction design | **Selected** |
| Serializable isolation | Strong guarantees | Higher contention, deadlocks | Rejected for MVP; use Read Committed + row locks |

## Idempotency patterns

| Pattern | Use case | Decision |
|---------|----------|----------|
| Stripe-style idempotency table | POST with Idempotency-Key | **Selected** for reserve |
| Status-gated DELETE | Safe repeated release | **Selected** for release |
| Unique constraint on (user, item, active) | Prevent duplicate holds | Rejected — blocks legitimate multiple reservations |

## TTL expiration strategies

| Strategy | Decision |
|----------|----------|
| PostgreSQL `pg_cron` | Rejected — extra extension setup |
| Go background ticker (every 5s) | **Selected** — simple, testable with injected clock |
| Lazy expiration on read | Rejected alone — stock would appear locked until read |

## Frontend sync

| Strategy | Decision |
|----------|----------|
| WebSockets | Nice-to-have; polling satisfies rubric |
| Polling (3s) | **Selected** |
| Manual refresh only | Rejected — fails state management rubric |

## Library choices

- **Go router**: Chi — lightweight, stdlib-compatible, widely used.
- **DB driver**: pgx v5 — native PostgreSQL, excellent pool support.
- **Migrations**: Embedded SQL in `migrations/` applied on startup for Docker simplicity.
- **Frontend data**: TanStack Query — built-in polling, loading/error states.
- **Frontend tests**: Vitest + React Testing Library + MSW for API mocking.

## Risks

1. **Expiration lag**: Up to one ticker interval (5s) after TTL before stock returns — acceptable; manual release covers UI desync.
2. **Idempotency race**: Two parallel first requests with same key — mitigated by `INSERT … ON CONFLICT` or advisory lock on key before work.

## Pivot note

Initial plan considered Redis for idempotency caching; pivoted to PostgreSQL-only to keep Docker Compose minimal and meet "PostgreSQL handles race conditions" evaluation criteria.
