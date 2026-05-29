# Data Model

## Entity Relationship

```text
items 1 ──< reservations
items (standalone inventory)
idempotency_keys (standalone, links to reservation_id optionally)
```

## Tables

### items

| Column           | Type         | Constraints        |
|------------------|--------------|--------------------|
| id               | UUID         | PK, default gen    |
| name             | TEXT         | NOT NULL           |
| total_quantity   | INT          | NOT NULL, CHECK ≥ 0 |
| reserved_quantity| INT          | NOT NULL, CHECK ≥ 0, ≤ total |

**Invariant**: `reserved_quantity <= total_quantity` at all times.

### reservations

| Column     | Type        | Constraints |
|------------|-------------|-------------|
| id         | UUID        | PK          |
| item_id    | UUID        | FK → items  |
| user_id    | TEXT        | NOT NULL    |
| quantity   | INT         | NOT NULL, CHECK > 0 |
| status     | TEXT        | NOT NULL, CHECK IN ('active','released','expired') |
| expires_at | TIMESTAMPTZ | NOT NULL    |
| created_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |
| updated_at | TIMESTAMPTZ | NOT NULL DEFAULT now() |

**Indexes**: `(user_id, status)`, `(status, expires_at)` for expiration sweeps.

### idempotency_keys

| Column          | Type   | Constraints |
|-----------------|--------|-------------|
| key             | TEXT   | PK          |
| request_hash    | TEXT   | NOT NULL    |
| response_status | INT    | NOT NULL    |
| response_body   | JSONB  | NOT NULL    |
| reservation_id  | UUID   | NULL        |
| created_at      | TIMESTAMPTZ | DEFAULT now() |

## State transitions

```text
active ──(manual DELETE)──> released
active ──(TTL worker)────> expired
released ──(DELETE again)─> no-op (idempotent)
expired  ──(DELETE)──────> no-op (idempotent)
```

## Stock mutation rules

| Event              | items.reserved_quantity change |
|--------------------|--------------------------------|
| Reserve success    | +quantity                      |
| Release success    | -quantity (from active only)   |
| Expire success     | -quantity (from active only)   |
| Idempotent re-release | 0                           |
