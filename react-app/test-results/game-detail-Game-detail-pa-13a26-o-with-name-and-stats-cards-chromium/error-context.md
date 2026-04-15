# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: game-detail.spec.ts >> Game detail page >> renders hero with name and stats cards
- Location: e2e/tests/game-detail.spec.ts:5:3

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: getByText('Players')
Expected: visible
Error: strict mode violation: getByText('Players') resolved to 2 elements:
    1) <div class="text-[0.62rem] font-bold uppercase tracking-wider text-accent mt-1">Players</div> aka getByText('Players', { exact: true })
    2) <p class="text-[0.875rem] leading-relaxed text-ink line-clamp-3">Bohnanza is the first in the Bohnanza family of g…</p> aka getByText('Bohnanza is the first in the')

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for getByText('Players')

```

# Page snapshot

```yaml
- generic [ref=e3]:
  - banner [ref=e4]:
    - button "‹ Collection" [ref=e6] [cursor=pointer]:
      - generic [ref=e7]: ‹
      - generic [ref=e8]: Collection
  - main [ref=e9]:
    - generic [ref=e10]:
      - generic [ref=e11]:
        - img "Bohnanza" [ref=e12]
        - generic [ref=e14]:
          - heading "Bohnanza" [level=1] [ref=e15]
          - generic [ref=e16]:
            - generic [ref=e17]: "1997"
            - generic [ref=e18]: ★ 7.1
            - generic [ref=e19]: Light
            - generic [ref=e20]: 🗣 No language
      - generic [ref=e21]:
        - generic [ref=e22]:
          - generic [ref=e23]: 2–7
          - generic [ref=e24]: Players
          - generic [ref=e25]: count
        - generic [ref=e26]:
          - generic [ref=e27]: "45"
          - generic [ref=e28]: Playtime
          - generic [ref=e29]: minutes
        - generic [ref=e30]:
          - generic [ref=e31]: "1.7"
          - generic [ref=e32]: Complexity
          - generic [ref=e33]: / 5.0
      - generic [ref=e34]:
        - heading "About" [level=2] [ref=e35]
        - paragraph [ref=e36]: "Bohnanza is the first in the Bohnanza family of games and has been published in several different editions, including a 2023 version with flowers. This entry lists a few different major card sets for Bohnanza: the base game for 3-5 players, the expanded game with the same name for 2-7 players (that is, first expansion included), and Bohnanza Pocket. In the game, you plant, then harvest bean cards in order to earn coins. Each player starts with a hand of random bean cards, and each card has a number on it corresponding to the number of that type of beans in the deck. Unlike in most other card games, you can't rearrange the order of cards in hand, so you must use them in the order that you've picked them up from the deck — unless you can trade them to other players, which is the heart of the game. On a turn, you must plant the first one or two cards in your hand into the \"fields\" in front of you. Each field can hold only one type of bean, so if you must plant a type of bean that's not in one of your fields, then you must harvest a field to make room for the new arrival. This usually isn't good! Next, you reveal two cards from the deck, and you can then trade these cards as well as any card in your hand for cards from other players. You can even make future promises for cards received right now! After all the trading is complete — and all trades on a turn must involve the active player — then you end your turn by drawing cards from the deck and placing them at the back of your hand. When you harvest beans, you receive coins based on the number of bean cards in that field and the \"beanometer\" for that particular type of bean. Flip over 1-4 cards from that field to transform them into coins, then place the remainder of the cards in the discard pile. When the deck runs out, shuffle the discards, playing through the deck two more times. At the end of the game, everyone can harvest their fields, then whoever has earned the most coins wins. The original German edition supports 3-5 players. The English version from Rio Grande Games comes with the first edition of the first German expansion included in a slightly oversized box. One difference in the contents, however, is that bean #22's Weinbrandbohne (Brandy Bean) was replaced by the Wachsbohne, or Wax Bean. This edition includes rules for up to seven players, like the Erweiterungs-Set, but also adapts the two-player rules of Al Cabohne in order to allow two people to play Bohnanza."
        - button "Read more ↓" [ref=e37] [cursor=pointer]
      - generic [ref=e38]:
        - generic [ref=e39]:
          - generic [ref=e40]: Categories
          - generic [ref=e41]:
            - generic [ref=e42]: Card Game
            - generic [ref=e43]: Farming
            - generic [ref=e44]: Negotiation
        - generic [ref=e45]:
          - generic [ref=e46]: Mechanics
          - generic [ref=e47]:
            - generic [ref=e48]: Hand Management
            - generic [ref=e49]: Multi-Use Cards
            - generic [ref=e50]: Negotiation
            - generic [ref=e51]: Set Collection
            - generic [ref=e52]: Trading
      - generic [ref=e53]:
        - heading "Player Aids" [level=2] [ref=e54]
        - generic [ref=e55]:
          - textbox "Label (optional)" [ref=e56]
          - generic [ref=e57] [cursor=pointer]: + Upload
      - generic [ref=e58]:
        - generic [ref=e59]:
          - heading "Vibes" [level=2] [ref=e60]
          - button "Edit" [ref=e61] [cursor=pointer]
        - generic [ref=e62]: No vibes assigned.
      - generic [ref=e63]:
        - generic [ref=e64]:
          - generic [ref=e65]:
            - generic [ref=e66]: 📖
            - generic [ref=e67]: No rulebook link
          - button "✏️" [ref=e68] [cursor=pointer]
        - link "🎲 View on BoardGameGeek ↗" [ref=e69] [cursor=pointer]:
          - /url: https://boardgamegeek.com/boardgame/11
          - generic [ref=e70]: 🎲
          - generic [ref=e71]: View on BoardGameGeek
          - generic [ref=e72]: ↗
      - button "Delete game" [ref=e74] [cursor=pointer]
  - navigation [ref=e75]:
    - link "⊞ Collection" [ref=e76] [cursor=pointer]:
      - /url: "#/"
      - generic [ref=e77]: ⊞
      - generic [ref=e78]: Collection
    - link "✦ Vibes" [ref=e79] [cursor=pointer]:
      - /url: "#/vibes"
      - generic [ref=e80]: ✦
      - generic [ref=e81]: Vibes
    - link "⇩ Import" [ref=e82] [cursor=pointer]:
      - /url: "#/import"
      - generic [ref=e83]: ⇩
      - generic [ref=e84]: Import
    - link "⊙ Profile" [ref=e85] [cursor=pointer]:
      - /url: "#/profile"
      - generic [ref=e86]: ⊙
      - generic [ref=e87]: Profile
```

# Test source

```ts
  1  | import { test, expect } from '../fixtures/auth'
  2  | import { goToFirstGame } from '../helpers/nav'
  3  | 
  4  | test.describe('Game detail page', () => {
  5  |   test('renders hero with name and stats cards', async ({ authenticatedPage: page }) => {
  6  |     await goToFirstGame(page)
  7  | 
  8  |     const heading = page.locator('h1').first()
  9  |     await expect(heading).toBeVisible()
  10 |     const name = await heading.textContent()
  11 |     expect(name?.trim().length).toBeGreaterThan(0)
  12 | 
> 13 |     await expect(page.getByText('Players')).toBeVisible()
     |                                             ^ Error: expect(locator).toBeVisible() failed
  14 |     await expect(page.getByText('Playtime')).toBeVisible()
  15 |     await expect(page.getByText('Complexity')).toBeVisible()
  16 |   })
  17 | 
  18 |   test('shows BGG link', async ({ authenticatedPage: page }) => {
  19 |     await goToFirstGame(page)
  20 |     await expect(page.getByRole('link', { name: /boardgamegeek/i })).toBeVisible()
  21 |   })
  22 | 
  23 |   test('player aids section has upload button', async ({ authenticatedPage: page }) => {
  24 |     await goToFirstGame(page)
  25 |     await expect(page.getByText('Player Aids')).toBeVisible()
  26 |     await expect(page.getByText('+ Upload')).toBeVisible()
  27 |   })
  28 | 
  29 |   test('vibes section has edit button', async ({ authenticatedPage: page }) => {
  30 |     await goToFirstGame(page)
  31 |     await expect(page.getByText('Vibes')).toBeVisible()
  32 |     await expect(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  33 |   })
  34 | 
  35 |   test('vibes edit shows save/cancel then cancels cleanly', async ({ authenticatedPage: page }) => {
  36 |     await goToFirstGame(page)
  37 |     await page.getByRole('button', { name: 'Edit' }).click()
  38 |     await expect(page.getByRole('button', { name: 'Save' })).toBeVisible()
  39 |     await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
  40 |     await page.getByRole('button', { name: 'Cancel' }).click()
  41 |     await expect(page.getByRole('button', { name: 'Edit' })).toBeVisible()
  42 |   })
  43 | 
  44 |   test('rulebook row shows edit pencil', async ({ authenticatedPage: page }) => {
  45 |     await goToFirstGame(page)
  46 |     await expect(page.getByText(/rulebook/i).first()).toBeVisible()
  47 |     await expect(page.getByTitle('Edit rulebook URL')).toBeVisible()
  48 |   })
  49 | 
  50 |   test('rules URL editor validates non-Drive URLs', async ({ authenticatedPage: page }) => {
  51 |     await goToFirstGame(page)
  52 |     await page.getByTitle('Edit rulebook URL').click()
  53 |     await expect(page.getByPlaceholder(/drive\.google\.com/i)).toBeVisible()
  54 | 
  55 |     await page.getByPlaceholder(/drive\.google\.com/i).fill('https://example.com/rules.pdf')
  56 |     await page.getByRole('button', { name: 'Save' }).click()
  57 |     await expect(page.getByText(/google drive/i)).toBeVisible()
  58 | 
  59 |     await page.getByRole('button', { name: 'Cancel' }).click()
  60 |     await expect(page.getByTitle('Edit rulebook URL')).toBeVisible()
  61 |   })
  62 | 
  63 |   test('delete confirmation can be cancelled', async ({ authenticatedPage: page }) => {
  64 |     await goToFirstGame(page)
  65 |     await page.getByRole('button', { name: 'Delete game' }).click()
  66 |     await expect(page.getByRole('button', { name: /yes, delete/i })).toBeVisible()
  67 |     await page.getByRole('button', { name: 'Cancel' }).last().click()
  68 |     await expect(page.getByRole('button', { name: 'Delete game' })).toBeVisible()
  69 |     await expect(page.getByRole('button', { name: /yes, delete/i })).not.toBeVisible()
  70 |   })
  71 | 
  72 |   test('back navigation returns to collection', async ({ authenticatedPage: page }) => {
  73 |     await goToFirstGame(page)
  74 |     await page.goBack()
  75 |     await expect(page).toHaveURL(/\/#\/$|#\/$|^\/$|localhost:5173\/$/)
  76 |   })
  77 | })
  78 | 
```