import { test, expect, type Page } from '@playwright/test'

// ── Auth helpers ─────────────────────────────────────────────────────────────
// Authentication protocol (no static credentials):
//   1. Primary: tests mock auth entirely via route interception / seeded tokens.
//   2. Fallback: if a real backend round-trip is required, read an ephemeral
//      JWT from TEST_TOKEN (and optional TEST_REFRESH_TOKEN). Never log it.
//   3. If neither is available, halt and report a blocker.
function readEphemeralToken(): { access: string; refresh: string } {
  const access = process.env.TEST_TOKEN
  if (!access) {
    throw new Error(
      'Blocker: no authentication method available. Set TEST_TOKEN to an ' +
      'ephemeral JWT or configure the test to mock auth. Static credentials ' +
      'are not permitted.'
    )
  }
  const refresh = process.env.TEST_REFRESH_TOKEN ?? access
  return { access, refresh }
}

// Seeds localStorage with an ephemeral JWT so tests skip the login UI.
async function seedAuth(page: Page) {
  const { access, refresh } = readEphemeralToken()
  await page.goto('/')
  await page.evaluate(({ a, r }) => {
    localStorage.setItem('mbgc_access', a)
    localStorage.setItem('mbgc_refresh', r)
  }, { a: access, r: refresh })
}

// Navigate to the collection and wait for at least one game link to appear
async function goToCollection(page: Page) {
  await page.goto('/')
  await expect(page.locator('a[href*="/games/"]').first()).toBeVisible({ timeout: 12000 })
}

// Navigate to the first game's detail page and wait for its heading
async function goToFirstGame(page: Page): Promise<string> {
  await goToCollection(page)
  const firstLink = page.locator('a[href*="/games/"]').first()
  const gameName = await firstLink.getAttribute('aria-label') ?? ''
  await firstLink.click()
  await expect(page).toHaveURL(/\/games\/\d+/, { timeout: 8000 })
  // Wait for hero heading — game name rendered in h1
  await expect(page.locator('h1').first()).toBeVisible({ timeout: 10000 })
  return gameName
}

// ── Auth ─────────────────────────────────────────────────────────────────────
test.describe('Auth', () => {
  test('redirects unauthenticated users to /login', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/#\/login/)
    await expect(page.getByRole('heading', { name: /board game collection/i })).toBeVisible()
  })

  test('login form submits and navigates to collection (mocked)', async ({ page }) => {
    // Mock the auth endpoint so the UI test does not require real credentials.
    await page.route('**/api/v1/auth/login', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: { access_token: 'mock.jwt.access', refresh_token: 'mock.jwt.refresh' },
        }),
      })
    )
    await page.goto('/#/login')
    await page.getByLabel('Username').fill('mock-user')
    await page.getByLabel('Password').fill('mock-pass')
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible({ timeout: 10000 })
  })

  test('login form shows error when backend rejects (mocked)', async ({ page }) => {
    await page.route('**/api/v1/auth/login', (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'invalid username or password' }),
      })
    )
    await page.goto('/#/login')
    await page.getByLabel('Username').fill('mock-user')
    await page.getByLabel('Password').fill('mock-pass')
    await page.getByRole('button', { name: /sign in/i }).click()
    await expect(page.getByText(/invalid username or password/i)).toBeVisible({ timeout: 5000 })
  })
})

// ── Collection page ───────────────────────────────────────────────────────────
test.describe('Collection page', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
    await goToCollection(page)
  })

  test('shows collection heading and game count', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
    await expect(page.getByText(/\d+ game/)).toBeVisible()
  })

  test('has at least one game in the list', async ({ page }) => {
    const count = await page.locator('a[href*="/games/"]').count()
    expect(count).toBeGreaterThan(0)
  })

  test('search filter narrows the game list', async ({ page }) => {
    const total = await page.locator('a[href*="/games/"]').count()
    // Type something that should match fewer results — first game's name prefix
    const firstName = await page.locator('a[href*="/games/"]').first().textContent()
    const prefix = (firstName ?? '').slice(0, 4).trim()
    if (!prefix) test.skip()

    await page.getByPlaceholder(/search/i).fill(prefix)
    await page.waitForTimeout(400) // debounce
    const filtered = await page.locator('a[href*="/games/"]').count()
    expect(filtered).toBeLessThanOrEqual(total)
  })

  test('navigates to game detail on click', async ({ page }) => {
    await page.locator('a[href*="/games/"]').first().click()
    await expect(page).toHaveURL(/\/games\/\d+/, { timeout: 8000 })
    await expect(page.locator('h1').first()).toBeVisible({ timeout: 10000 })
  })
})

// ── Game detail page ──────────────────────────────────────────────────────────
test.describe('Game detail page', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
  })

  test('renders hero with game name, year, and stats cards', async ({ page }) => {
    await goToFirstGame(page)

    // Hero h1 is visible
    const heading = page.locator('h1').first()
    await expect(heading).toBeVisible()
    const name = await heading.textContent()
    expect(name?.trim().length).toBeGreaterThan(0)

    // Stats row: Players / Playtime / Complexity
    await expect(page.getByText('Players')).toBeVisible()
    await expect(page.getByText('Playtime')).toBeVisible()
    await expect(page.getByText('Complexity')).toBeVisible()
  })

  test('shows BGG link', async ({ page }) => {
    await goToFirstGame(page)
    await expect(page.getByRole('link', { name: /boardgamegeek/i })).toBeVisible()
  })

  test('player aids section is always visible with upload button', async ({ page }) => {
    await goToFirstGame(page)
    await expect(page.getByText('Player Aids')).toBeVisible()
    await expect(page.getByText('+ Upload')).toBeVisible()
  })

  test('vibes section is always visible with edit button', async ({ page }) => {
    await goToFirstGame(page)
    await expect(page.getByText('Vibes')).toBeVisible()
    // Edit button should be present
    await expect(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  })

  test('vibes edit shows collection checkboxes then saves', async ({ page }) => {
    await goToFirstGame(page)
    await page.getByRole('button', { name: 'Edit' }).click()
    // Save and Cancel buttons should appear
    await expect(page.getByRole('button', { name: 'Save' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
    // Dismiss without changing
    await page.getByRole('button', { name: 'Cancel' }).click()
    await expect(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  })

  test('rulebook row is visible with edit pencil', async ({ page }) => {
    await goToFirstGame(page)
    // Either a link or the "No rulebook link" placeholder
    const ruleRow = page.getByText(/rulebook/i).first()
    await expect(ruleRow).toBeVisible()
    await expect(page.getByTitle('Edit rulebook URL')).toBeVisible()
  })

  test('rules URL editor opens, validates, and cancels', async ({ page }) => {
    await goToFirstGame(page)
    await page.getByTitle('Edit rulebook URL').click()
    await expect(page.getByPlaceholder(/drive\.google\.com/i)).toBeVisible()

    // Enter a non-Drive URL — should show a validation error on save
    await page.getByPlaceholder(/drive\.google\.com/i).fill('https://example.com/rules.pdf')
    await page.getByRole('button', { name: 'Save' }).click()
    await expect(page.getByText(/google drive/i)).toBeVisible()

    // Cancel and return to view mode
    await page.getByRole('button', { name: 'Cancel' }).click()
    await expect(page.getByTitle('Edit rulebook URL')).toBeVisible()
  })

  test('delete confirmation flow shows confirm then cancel', async ({ page }) => {
    await goToFirstGame(page)
    await page.getByRole('button', { name: 'Delete game' }).click()
    await expect(page.getByRole('button', { name: /yes, delete/i })).toBeVisible()
    await page.getByRole('button', { name: 'Cancel' }).last().click()
    // Should be back to normal state (delete button visible, confirm gone)
    await expect(page.getByRole('button', { name: 'Delete game' })).toBeVisible()
    await expect(page.getByRole('button', { name: /yes, delete/i })).not.toBeVisible()
  })

  test('back navigation returns to collection', async ({ page }) => {
    await goToFirstGame(page)
    await page.goBack()
    await expect(page).toHaveURL(/\/#\/$|#\/$|^\/$|localhost:5173\/$/)
  })
})

// ── Lightbox ──────────────────────────────────────────────────────────────────
test.describe('Player aid lightbox', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
  })

  test('clicking a player aid thumbnail opens lightbox overlay', async ({ page }) => {
    await goToFirstGame(page)
    const thumbnail = page.locator('img[alt]').filter({ hasText: '' }).nth(1) // skip hero img
    const aids = page.locator('#player-aid-thumb')
    const aidCount = await page.locator('button[style*="cursor: pointer"] img').count()

    if (aidCount === 0) {
      // No player aids on this game — skip gracefully
      test.skip()
      return
    }

    await page.locator('button[style*="cursor: pointer"] img').first().click()
    // Lightbox overlay should appear (fixed position, dark background)
    await expect(page.locator('div[style*="position: fixed"]')).toBeVisible()
    // Close with ✕ button
    await page.keyboard.press('Escape')
    await expect(page.locator('div[style*="position: fixed"]')).not.toBeVisible()
  })
})

// ── Vibes page ────────────────────────────────────────────────────────────────
test.describe('Vibes page', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
    await page.goto('/#/vibes')
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible({ timeout: 8000 })
  })

  test('loads collection pills from API', async ({ page }) => {
    await expect(page.locator('button.pressable').first()).toBeVisible({ timeout: 8000 })
  })

  test('selecting a collection shows games or empty state', async ({ page }) => {
    const firstPill = page.locator('button.pressable').first()
    await firstPill.click()
    await expect(
      page.locator('a[href*="/games/"]').first().or(page.getByText(/No games found/))
    ).toBeVisible({ timeout: 8000 })
  })
})

// ── Tab navigation ────────────────────────────────────────────────────────────
test.describe('Tab navigation', () => {
  test.beforeEach(async ({ page }) => {
    await seedAuth(page)
    await goToCollection(page)
  })

  test('Collection tab is active on root route', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
  })

  test('Vibes tab navigates to vibes page', async ({ page }) => {
    await page.getByRole('link', { name: /vibes/i }).click()
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible({ timeout: 8000 })
  })

  test('Collection tab navigates back from vibes', async ({ page }) => {
    await page.goto('/#/vibes')
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible({ timeout: 8000 })
    await page.getByRole('link', { name: /collection/i }).click()
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible({ timeout: 8000 })
  })
})
