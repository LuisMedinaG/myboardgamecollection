# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: vibes.spec.ts >> Vibes page >> selecting a collection shows games or empty state
- Location: e2e/tests/vibes.spec.ts:13:3

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
  1  | import { expect, type Page } from '@playwright/test'
  2  | 
  3  | /** Go to the collection and wait for at least one game link. */
  4  | export async function goToCollection(page: Page): Promise<void> {
  5  |   await page.goto('/')
  6  |   await expect(page.locator('a[href*="/games/"]').first()).toBeVisible({ timeout: 12000 })
  7  | }
  8  | 
  9  | /** Go to the first game's detail page. Returns the game's aria-label. */
  10 | export async function goToFirstGame(page: Page): Promise<string> {
  11 |   await goToCollection(page)
  12 |   const firstLink = page.locator('a[href*="/games/"]').first()
  13 |   const name = (await firstLink.getAttribute('aria-label')) ?? ''
  14 |   await firstLink.click()
  15 |   await expect(page).toHaveURL(/\/games\/\d+/, { timeout: 8000 })
  16 |   await expect(page.locator('h1').first()).toBeVisible({ timeout: 10000 })
  17 |   return name
  18 | }
  19 | 
  20 | /** Go to the vibes page and wait for its heading. */
  21 | export async function goToVibes(page: Page): Promise<void> {
  22 |   await page.goto('/#/vibes')
> 23 |   await expect(page.getByRole('heading', { name: 'Browse by Vibe' })).toBeVisible({ timeout: 8000 })
     |                                                                       ^ Error: expect(locator).toBeVisible() failed
  24 | }
  25 | 
```