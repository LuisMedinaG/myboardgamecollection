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
# Terminal 1 — start backend (creates test user automatically if TEST_USER set)
TEST_USER=testuser TEST_PASSWORD=testpass123 make dev-go

# Terminal 2 — run E2E suite (auto-logins if no TEST_TOKEN provided)
cd react-app
bun run test:e2e
```

E2E suite covers: login → collection list → game detail → vibes/collections flow.

Auth protocol: tests mock `/api/v1/auth/login` where possible. When a real
session is required, use TEST_TOKEN if provided, otherwise auto-login with
TEST_USER/TEST_PASSWORD (defaults: testuser/testpass123). Never use static
usernames/passwords in TEST_TOKEN. The token is never logged.

To generate a TEST_TOKEN manually:

```sh
TEST_USER=testuser TEST_PASSWORD=testpass123 make test-token
```

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
