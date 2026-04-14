import { test, expect } from '@playwright/test'

test.describe('Collection page', () => {
  test('loads and shows game list', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
    // At least one game row should render
    const gameLinks = page.locator('a[href*="/games/"]')
    await expect(gameLinks.first()).toBeVisible()
    const count = await gameLinks.count()
    expect(count).toBeGreaterThan(0)
  })

  test('shows game count in subtitle', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByText(/\d+ games · find your next play/)).toBeVisible()
  })

  test('filters games by search', async ({ page }) => {
    await page.goto('/')
    const allLinks = page.locator('a[href*="/games/"]')
    const total = await allLinks.count()

    await page.getByPlaceholder(/search/i).fill('Catan')
    // Result count should update
    await expect(page.getByText('1 game')).toBeVisible()
    const filtered = page.locator('a[href*="/games/"]')
    expect(await filtered.count()).toBeLessThan(total)
  })

  test('navigates to game detail page', async ({ page }) => {
    await page.goto('/')
    const first = page.locator('a[href*="/games/"]').first()
    const name = await first.locator('div[style*="font-weight: 600"]').innerText().catch(() => '')
    await first.click()
    await expect(page).toHaveURL(/\/games\/\d+/)
    // Back button should appear (the ‹ Collection button, not the tab)
    await expect(page.getByRole('button', { name: /Collection/ })).toBeVisible()
  })
})

test.describe('Vibes page', () => {
  test('loads vibe pills', async ({ page }) => {
    await page.goto('/#/vibes')
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Social' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Competitive' })).toBeVisible()
  })

  test('selecting a vibe filters games', async ({ page }) => {
    await page.goto('/#/vibes')
    await page.getByRole('button', { name: 'Social' }).click()
    await expect(page.getByText(/game.*"Social" vibe/)).toBeVisible()
  })
})

test.describe('Tab navigation', () => {
  test('tab bar switches between Collection and Vibes', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()

    await page.getByRole('link', { name: /vibes/i }).click()
    await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible()

    // Click the tab bar link (the ⊞ Collection tab, not the header title)
    await page.getByRole('link', { name: '⊞ Collection' }).click()
    await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
  })
})
