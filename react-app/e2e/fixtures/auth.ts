import { test as base, type Page } from '@playwright/test'

/**
 * Authentication protocol for the E2E suite.
 *
 * 1. Primary — mock `/api/v1/auth/login` via `page.route()` in the test itself
 *    (see `helpers/mocks.ts`). Use this for anything that exercises the login
 *    UI or does not require a real authenticated session.
 * 2. Fallback — for flows that must hit the real backend, read an ephemeral
 *    JWT from `TEST_TOKEN` (optional `TEST_REFRESH_TOKEN`). This fixture
 *    seeds localStorage so the app boots authenticated.
 * 3. If neither is available, tests halt with a blocker.
 *
 * Static usernames/passwords are never allowed. The token is never logged.
 */
function readEphemeralToken(): { access: string; refresh: string } {
  const access = process.env.TEST_TOKEN
  if (!access) {
    throw new Error(
      'Blocker: TEST_TOKEN not set. Provide an ephemeral JWT or rewrite the ' +
      'test to mock auth. Static credentials are not permitted.'
    )
  }
  return { access, refresh: process.env.TEST_REFRESH_TOKEN ?? access }
}

export async function seedAuth(page: Page): Promise<void> {
  const { access, refresh } = readEphemeralToken()
  await page.goto('/')
  await page.evaluate(({ a, r }) => {
    localStorage.setItem('mbgc_access', a)
    localStorage.setItem('mbgc_refresh', r)
  }, { a: access, r: refresh })
}

type AuthFixtures = {
  authenticatedPage: Page
}

/**
 * Extended test runner. Use `test` from this module instead of `@playwright/test`
 * when a test needs to start already authenticated:
 *
 *   test('...', async ({ authenticatedPage }) => { ... })
 *
 * For unauthenticated / login-UI tests, use the plain `page` fixture.
 */
export const test = base.extend<AuthFixtures>({
  authenticatedPage: async ({ page }, use) => {
    await seedAuth(page)
    await use(page)
  },
})

export { expect } from '@playwright/test'
