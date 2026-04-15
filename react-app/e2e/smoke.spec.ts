import { test, expect, type Page } from '@playwright/test'

// ── Auth helpers ────────────────────────────────────────────────────────────────
// Seeds localStorage with valid JWT tokens via the API so tests don't go
// through the login UI on every run. Requires the Go backend at :8080.

const TEST_USER = process.env.TEST_USERNAME ?? 'testuser'
const TEST_PASS = process.env.TEST_PASSWORD ?? 'testpassword'

async function seedAuth(page: Page) {
  // Hit the API directly via Playwright's request context
  const resp = await page.request.post('http://localhost:8080/api/v1/auth/login', {
    data: { username: TEST_USER, password: TEST_PASS },
  })
  if (!resp.ok()) {
    throw new Error(`Login failed (${resp.status()}): set TEST_USERNAME / TEST_PASSWORD env vars`)
  }
  const body = await resp.json()
  const { access_token, refresh_token } = body.data

  // Navigate to the app first so we can write to its localStorage origin
  await page.goto('/')
  await page.evaluate(({ access, refresh }) => {
    localStorage.setItem('mbgc_access', access)
    localStorage.setItem('mbgc_refresh', refresh)
  }, { access: access_token, refresh: refresh_token })
}

// ── Tests ───────────────────────────────────────────────────────────────────────

test.describe('Auth', () => {
  test('redirects unauthenticated users to /login', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/#\/login/)
    await expect(page.getByRole('heading', { name: /board game collection/i })).toBeVisible()
  })

  test('login form accepts credentials and navigates to collection', async ({ page }) => {
    await page.goto('/#/login')
    await page.getByLabel(/username/i).fill(TEST_USER)
    await page.getByLabel(/password/i).fill(TEST_PASS)
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible({ timeout: 8000 })
  })
})

test.describe('Collection page', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
    await page.goto('/')
    // Wait for the collection to load (skeleton disappears, games appear)
    await expect(page.locator('a[href*="/games/"]').first()).toBeVisible({ timeout: 10000 })
  })

  test('loads and shows game list', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
    const gameLinks = page.locator('a[href*="/games/"]')
    const count = await gameLinks.count()
    expect(count).toBeGreaterThan(0)
  })

  test('shows game count in subtitle', async ({ page }) => {
    await expect(page.getByText(/\d+ games · find your next play/)).toBeVisible()
  })

  test('filters games by search', async ({ page }) => {
    const allLinks = page.locator('a[href*="/games/"]')
    const total = await allLinks.count()

    await page.getByPlaceholder(/search/i).fill('Catan')
    await expect(page.getByText('1 game')).toBeVisible({ timeout: 5000 })
    const filtered = page.locator('a[href*="/games/"]')
    expect(await filtered.count()).toBeLessThan(total)
  })

  test('navigates to game detail page', async ({ page }) => {
    await page.locator('a[href*="/games/"]').first().click()
    await expect(page).toHaveURL(/\/games\/\d+/)
    await expect(page.getByRole('button', { name: /Collection/ })).toBeVisible()
  })
})

test.describe('Vibes page', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
    await page.goto('/#/vibes')
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible()
  })

  test('loads collection pills from API', async ({ page }) => {
    // At least one pill button should appear after collections load
    await expect(page.locator('button.pressable').first()).toBeVisible({ timeout: 8000 })
  })

  test('selecting a collection shows games', async ({ page }) => {
    const firstPill = page.locator('button.pressable').first()
    await firstPill.click()
    // Either game rows or an empty-state message should appear
    await expect(
      page.locator('a[href*="/games/"]').first().or(page.getByText(/No games found/))
    ).toBeVisible({ timeout: 8000 })
  })
})

test.describe('Tab navigation', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible({ timeout: 8000 })
  })

  test('tab bar switches between Collection and Vibes', async ({ page }) => {
    await page.getByRole('link', { name: /vibes/i }).click()
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible()

    await page.getByRole('link', { name: '⊞ Collection' }).click()
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
  })
})
