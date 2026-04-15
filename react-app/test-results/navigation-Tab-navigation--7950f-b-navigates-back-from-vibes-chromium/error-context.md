# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: navigation.spec.ts >> Tab navigation >> Collection tab navigates back from vibes
- Location: e2e/tests/navigation.spec.ts:17:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByRole('heading', { name: 'Browse by Vibe' })
Expected: visible
Timeout: 8000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 8000ms
  - waiting for getByRole('heading', { name: 'Browse by Vibe' })

```

# Page snapshot

```yaml
- generic [ref=e4]:
  - generic [ref=e5]:
    - generic [ref=e6]: 🎲
    - heading "My Board Game Collection" [level=1] [ref=e7]
    - paragraph [ref=e8]: Sign in to your account
  - generic [ref=e9]:
    - generic [ref=e10]:
      - generic [ref=e11]: Username
      - textbox "Username" [ref=e12]
    - generic [ref=e13]:
      - generic [ref=e14]: Password
      - textbox "Password" [ref=e15]
    - button "Sign in" [ref=e16] [cursor=pointer]
```

# Test source

```ts
  1  | import { test, expect } from '../fixtures/auth'
  2  | import { goToCollection } from '../helpers/nav'
  3  | 
  4  | test.describe('Tab navigation', () => {
  5  |   test('Collection tab active on root', async ({ authenticatedPage: page }) => {
  6  |     await goToCollection(page)
  7  |     await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
  8  |   })
  9  | 
  10 |   test('Vibes tab navigates to vibes page', async ({ authenticatedPage: page }) => {
  11 |     await goToCollection(page)
  12 |     await page.getByRole('link', { name: /vibes/i }).click()
  13 |     await expect(page.getByRole('heading', { name: 'Browse by Vibe' }))
  14 |       .toBeVisible({ timeout: 8000 })
  15 |   })
  16 | 
  17 |   test('Collection tab navigates back from vibes', async ({ authenticatedPage: page }) => {
  18 |     await page.goto('/#/vibes')
  19 |     await expect(page.getByRole('heading', { name: 'Browse by Vibe' }))
> 20 |       .toBeVisible({ timeout: 8000 })
     |        ^ Error: expect(locator).toBeVisible() failed
  21 |     await page.getByRole('link', { name: /collection/i }).click()
  22 |     await expect(page.getByRole('heading', { name: 'Board Game Collection' }))
  23 |       .toBeVisible({ timeout: 8000 })
  24 |   })
  25 | })
  26 | 
```