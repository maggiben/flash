# Spec Kit Notes

## Commands used (Spec Kit workflow simulation)

This project followed the Spec Kit sequence manually (equivalent slash commands):

| Step | Command equivalent | Artifact(s) produced |
|------|-------------------|----------------------|
| 0 | Constitution setup | `.specify/memory/constitution.md` |
| 1 | `/speckit.specify` | `specs/001-inventory-reservation/spec.md` |
| 2 | `/speckit.plan` | `plan.md`, `research.md`, `data-model.md`, `contracts/openapi.yaml`, `quickstart.md`, `checklist.md` |
| 3 | `/speckit.tasks` | `tasks.md` |
| 4 | `/speckit.implement` | `backend/`, `frontend/`, `docker-compose.yml` |

**Recommended git commit order for submission review:**

1. `docs: add constitution and spec kit artifacts`
2. `docs: add plan, research, data model, openapi contract`
3. `docs: add tasks and checklist`
4. `feat: add Go backend with postgres concurrency`
5. `feat: add React frontend with polling`
6. `chore: docker compose and seed data`
7. `test: concurrency, idempotency, and UI tests`

## Assumptions

1. **Auth**: `X-User-Id` header instead of login (challenge silent on auth).
2. **Design reference**: PDF mentions a visual reference not embedded; UI uses a dark flash-sale card layout.
3. **Confirm flow**: No `POST /confirm`; “unconfirmed” = `active` status until release, expiry, or future extension.
4. **Polling vs WebSocket**: Chose 3s polling per plan (satisfies “WebSockets or polling” rubric).
5. **Repo name**: Avoid company name from challenge PDF in public repos.

## Refinements during planning

- Added explicit edge-case table in `spec.md` (idempotency hash, expiration vs manual release race).
- Quantified concurrency acceptance criteria (50+ and 100 goroutines).

## Pivots

| Phase | Original idea | Pivot | Reason |
|-------|---------------|-------|--------|
| Plan | Redis for idempotency cache | PostgreSQL `idempotency_keys` table | Single datastore; simpler Docker; meets DB concurrency rubric |
| Plan | Serializable isolation | Read Committed + row locks + conditional UPDATE | Lower deadlock rate; sufficient with atomic UPDATE |
| Implement | Lazy expiration only | Background ticker (5s) + manual release | Stock must return without user action; manual release handles UI timer desync |

## Docker-only execution

All build, run, and test commands use Docker Compose — no host PostgreSQL or `go test` on bare metal required.

```bash
docker compose up --build
docker compose --profile test run --rm test
docker compose --profile test run --rm frontend-test
```

## Analyze checklist (self-review)

- [x] spec.md edge cases addressed before code
- [x] plan.md concurrency strategy matches implementation
- [x] tasks.md items map to files
- [x] OpenAPI matches handlers
- [x] README documents concurrency and LLM choice
