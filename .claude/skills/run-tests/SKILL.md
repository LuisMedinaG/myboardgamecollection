---
name: run-tests
description: Run Go unit tests and/or Playwright E2E tests. Use when asked to run tests or verify correctness.
---

# Run Tests

## Go unit tests

```sh
make test        # all tests, coverage summary
make test-v      # verbose — shows each test name
make cover       # per-package coverage %
make cover-html  # HTML coverage report (opens browser)
```

Current coverage target: ~70% (Phase 2 in progress).
Current baseline: 88 tests, ~52% coverage (Phase 1 complete).

Packages under test:
- `shared/httpx` — auth middleware, JWT, rate limiting
- `shared/db` — schema migrations, idempotency
- `services/games` — filter logic (`filter_test.go`)

## Playwright E2E tests

Requires the Go backend running first:

```sh
# Terminal 1 — start backend
make dev-go

# Terminal 2 — run E2E suite
cd react-app
TEST_TOKEN=<ephemeral-jwt> bun run test:e2e
```

E2E suite covers: login → collection list → game detail → vibes/collections flow.

Auth protocol: tests mock `/api/v1/auth/login` where possible. When a real
session is required, pass an ephemeral JWT via `TEST_TOKEN` (optionally
`TEST_REFRESH_TOKEN`). Never use static usernames/passwords. Never log the
token. If neither mocking nor `TEST_TOKEN` is available, halt and report a
blocker.

## Before shipping

Run at minimum:

```sh
make test
```

For features that touch auth, JWT, or the DB schema, also run E2E.

## Interpreting failures

- `FAIL` in Go output — read the test name and error, fix the root cause
- `UNIQUE constraint` in tests — test isolation issue; each test should use a fresh DB or unique IDs
- Playwright `TimeoutError` — backend may not be running or the JWT expired mid-test
