# Quickstart Validation Scenarios

Run all commands inside Docker (no host PostgreSQL required).

## Prerequisites

- Docker and Docker Compose installed
- Ports 5173, 8080, 5432 available

## Start stack

```bash
docker compose up --build
```

- Frontend: http://localhost:5173
- API: http://localhost:8080/api/v1/inventory
- OpenAPI: http://localhost:8080/openapi.yaml

## Manual scenarios

### 1. View inventory

```bash
curl -s http://localhost:8080/api/v1/inventory | jq
```

Expect items with `total_quantity`, `reserved_quantity`, `available_quantity`.

### 2. Reserve with idempotency

```bash
curl -s -X POST http://localhost:8080/api/v1/reservations \
  -H "Content-Type: application/json" \
  -H "X-User-Id: user-demo-1" \
  -H "Idempotency-Key: demo-key-001" \
  -d '{"item_id":"<ITEM_UUID>","quantity":1}' | jq
```

Repeat the same command — expect identical `id` and no double decrement.

### 3. Insufficient stock

Reserve more units than available — expect HTTP 409 with `INSUFFICIENT_STOCK`.

### 4. Release (idempotent)

```bash
curl -s -X DELETE http://localhost:8080/api/v1/reservations/<RESERVATION_ID> \
  -H "X-User-Id: user-demo-1"
```

Run twice — second call succeeds as no-op; available stock increases once.

### 5. TTL expiration

Create reservation, wait 65 seconds, verify it disappears from:

```bash
curl -s http://localhost:8080/api/v1/reservations -H "X-User-Id: user-demo-1" | jq
```

And inventory available count increases.

## Automated tests

```bash
# Go integration + concurrency tests
docker compose --profile test run --rm test

# Frontend unit/component tests
docker compose --profile test run --rm frontend-test
```

## Expected test outcomes

| Test | Expected |
|------|----------|
| ConcurrentLastItem | 1 success, 49+ conflicts |
| ConcurrentTenUnits | 10 success, 90 conflicts |
| ReserveIdempotencyParallel | 1 reservation, 1 stock decrement |
| ReleaseIdempotency | stock +quantity once |
