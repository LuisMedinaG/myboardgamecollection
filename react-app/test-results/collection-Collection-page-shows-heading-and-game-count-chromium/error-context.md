# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: collection.spec.ts >> Collection page >> shows heading and game count
- Location: e2e/tests/collection.spec.ts:9:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText(/\d+ game/)
Expected: visible
Error: strict mode violation: getByText(/\d+ game/) resolved to 2 elements:
    1) <p>11 games · find your next play</p> aka getByText('games · find your next play')
    2) <span>11 games</span> aka getByText('11 games', { exact: true })

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByText(/\d+ game/)

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - banner [ref=e4]:
    - generic [ref=e5]:
      - link "🎲 My Collection" [ref=e7] [cursor=pointer]:
        - /url: "#/"
        - generic [ref=e8]: 🎲
        - generic [ref=e9]: My Collection
      - button "Sign out" [ref=e10] [cursor=pointer]
  - main [ref=e11]:
    - generic [ref=e12]:
      - generic [ref=e13]:
        - heading "Board Game Collection" [level=1] [ref=e14]
        - paragraph [ref=e15]: 11 games · find your next play
      - generic [ref=e16]:
        - generic [ref=e17]:
          - generic: 🔍
          - searchbox "Search games…" [ref=e18]
        - generic [ref=e19]:
          - combobox [ref=e20] [cursor=pointer]:
            - option "All categories" [selected]
            - option "Abstract Strategy"
            - option "Bluffing"
            - option "Card Game"
            - option "Children's Game"
            - option "City Building"
            - option "Comic Book / Strip"
            - option "Dice"
            - option "Economic"
            - option "Farming"
            - option "Maze"
            - option "Medieval"
            - option "Movies / TV / Radio theme"
            - option "Negotiation"
            - option "Number"
            - option "Party Game"
            - option "Puzzle"
            - option "Real-time"
          - combobox [ref=e21] [cursor=pointer]:
            - option "Any players" [selected]
            - option "Solo (1)"
            - option "Up to 2"
            - option "Exactly 2"
            - option "Up to 3"
            - option "Up to 4"
            - option "5+"
          - combobox [ref=e22] [cursor=pointer]:
            - option "Any duration" [selected]
            - option "< 30 min"
            - option "30–60 min"
            - option "> 60 min"
          - combobox [ref=e23] [cursor=pointer]:
            - option "Any complexity" [selected]
            - option "Light"
            - option "Medium"
            - option "Heavy"
      - generic [ref=e24]:
        - generic [ref=e25]: 11 games
        - generic [ref=e26]:
          - button "List view" [ref=e27] [cursor=pointer]: ☰
          - button "Grid view" [ref=e28] [cursor=pointer]: ⊞
      - generic [ref=e29]:
        - link "Bohnanza Bohnanza 2–7 players · 45 min Light ›" [ref=e30] [cursor=pointer]:
          - /url: "#/games/119"
          - img "Bohnanza" [ref=e31]
          - generic [ref=e32]:
            - generic [ref=e33]: Bohnanza
            - generic [ref=e34]: 2–7 players · 45 min
            - generic [ref=e36]: Light
          - generic [ref=e37]: ›
        - link "Can't Stop Can't Stop 2–4 players · 30 min Light ›" [ref=e38] [cursor=pointer]:
          - /url: "#/games/120"
          - img "Can't Stop" [ref=e39]
          - generic [ref=e40]:
            - generic [ref=e41]: Can't Stop
            - generic [ref=e42]: 2–4 players · 30 min
            - generic [ref=e44]: Light
          - generic [ref=e45]: ›
        - link "Chinatown Chinatown 3–5 players · 60 min Medium ›" [ref=e46] [cursor=pointer]:
          - /url: "#/games/122"
          - img "Chinatown" [ref=e47]
          - generic [ref=e48]:
            - generic [ref=e49]: Chinatown
            - generic [ref=e50]: 3–5 players · 60 min
            - generic [ref=e52]: Medium
          - generic [ref=e53]: ›
        - link "El Grande El Grande 2–5 players · 120 min Medium ›" [ref=e54] [cursor=pointer]:
          - /url: "#/games/124"
          - img "El Grande" [ref=e55]
          - generic [ref=e56]:
            - generic [ref=e57]: El Grande
            - generic [ref=e58]: 2–5 players · 120 min
            - generic [ref=e60]: Medium
          - generic [ref=e61]: ›
        - link "For Sale For Sale 3–6 players · 30 min Light ›" [ref=e62] [cursor=pointer]:
          - /url: "#/games/125"
          - img "For Sale" [ref=e63]
          - generic [ref=e64]:
            - generic [ref=e65]: For Sale
            - generic [ref=e66]: 3–6 players · 30 min
            - generic [ref=e68]: Light
          - generic [ref=e69]: ›
        - link "High Society High Society 3–5 players · 30 min Light ›" [ref=e70] [cursor=pointer]:
          - /url: "#/games/126"
          - img "High Society" [ref=e71]
          - generic [ref=e72]:
            - generic [ref=e73]: High Society
            - generic [ref=e74]: 3–5 players · 30 min
            - generic [ref=e76]: Light
          - generic [ref=e77]: ›
        - link "Perudo Perudo 2–10 players · 30 min Light ›" [ref=e78] [cursor=pointer]:
          - /url: "#/games/121"
          - img "Perudo" [ref=e79]
          - generic [ref=e80]:
            - generic [ref=e81]: Perudo
            - generic [ref=e82]: 2–10 players · 30 min
            - generic [ref=e84]: Light
          - generic [ref=e85]: ›
        - link "Ricochet Robots Ricochet Robots 1–99 players · 30 min Medium ›" [ref=e86] [cursor=pointer]:
          - /url: "#/games/123"
          - img "Ricochet Robots" [ref=e87]
          - generic [ref=e88]:
            - generic [ref=e89]: Ricochet Robots
            - generic [ref=e90]: 1–99 players · 30 min
            - generic [ref=e92]: Medium
          - generic [ref=e93]: ›
        - link "Schotten Totten Schotten Totten 2 players · 20 min Light ›" [ref=e94] [cursor=pointer]:
          - /url: "#/games/127"
          - img "Schotten Totten" [ref=e95]
          - generic [ref=e96]:
            - generic [ref=e97]: Schotten Totten
            - generic [ref=e98]: 2 players · 20 min
            - generic [ref=e100]: Light
          - generic [ref=e101]: ›
        - link "Take 5 Take 5 2–10 players · 45 min Light ›" [ref=e102] [cursor=pointer]:
          - /url: "#/games/128"
          - img "Take 5" [ref=e103]
          - generic [ref=e104]:
            - generic [ref=e105]: Take 5
            - generic [ref=e106]: 2–10 players · 45 min
            - generic [ref=e108]: Light
          - generic [ref=e109]: ›
        - link "UNO UNO 2–10 players · 30 min Light ›" [ref=e110] [cursor=pointer]:
          - /url: "#/games/129"
          - img "UNO" [ref=e111]
          - generic [ref=e112]:
            - generic [ref=e113]: UNO
            - generic [ref=e114]: 2–10 players · 30 min
            - generic [ref=e116]: Light
          - generic [ref=e117]: ›
  - navigation [ref=e118]:
    - link "⊞ Collection" [ref=e119] [cursor=pointer]:
      - /url: "#/"
      - generic [ref=e120]: ⊞
      - generic [ref=e121]: Collection
    - link "✦ Vibes" [ref=e122] [cursor=pointer]:
      - /url: "#/vibes"
      - generic [ref=e123]: ✦
      - generic [ref=e124]: Vibes
    - link "⇩ Import" [ref=e125] [cursor=pointer]:
      - /url: "#/import"
      - generic [ref=e126]: ⇩
      - generic [ref=e127]: Import
    - link "⊙ Profile" [ref=e128] [cursor=pointer]:
      - /url: "#/profile"
      - generic [ref=e129]: ⊙
      - generic [ref=e130]: Profile
```

# Test source

```ts
  1  | import { test, expect } from '../fixtures/auth'
  2  | import { goToCollection } from '../helpers/nav'
  3  | 
  4  | test.describe('Collection page', () => {
  5  |   test.beforeEach(async ({ authenticatedPage }) => {
  6  |     await goToCollection(authenticatedPage)
  7  |   })
  8  | 
  9  |   test('shows heading and game count', async ({ authenticatedPage: page }) => {
  10 |     await expect(page.getByRole('heading', { name: 'Board Game Collection' })).toBeVisible()
> 11 |     await expect(page.getByText(/\d+ game/)).toBeVisible()
     |                                              ^ Error: expect(locator).toBeVisible() failed
  12 |   })
  13 | 
  14 |   test('has at least one game in the list', async ({ authenticatedPage: page }) => {
  15 |     const count = await page.locator('a[href*="/games/"]').count()
  16 |     expect(count).toBeGreaterThan(0)
  17 |   })
  18 | 
  19 |   test('search filter narrows the game list', async ({ authenticatedPage: page }) => {
  20 |     const total = await page.locator('a[href*="/games/"]').count()
  21 |     const firstName = await page.locator('a[href*="/games/"]').first().textContent()
  22 |     const prefix = (firstName ?? '').slice(0, 4).trim()
  23 |     if (!prefix) test.skip()
  24 | 
  25 |     await page.getByPlaceholder(/search/i).fill(prefix)
  26 |     await page.waitForTimeout(400) // debounce
  27 |     const filtered = await page.locator('a[href*="/games/"]').count()
  28 |     expect(filtered).toBeLessThanOrEqual(total)
  29 |   })
  30 | 
  31 |   test('navigates to game detail on click', async ({ authenticatedPage: page }) => {
  32 |     await page.locator('a[href*="/games/"]').first().click()
  33 |     await expect(page).toHaveURL(/\/games\/\d+/, { timeout: 8000 })
  34 |     await expect(page.locator('h1').first()).toBeVisible({ timeout: 10000 })
  35 |   })
  36 | })
  37 | 
```