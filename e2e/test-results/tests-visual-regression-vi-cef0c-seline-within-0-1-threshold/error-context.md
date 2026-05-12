# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/visual/regression.spec.ts >> visual-regression >> command palette matches baseline within 0.1% threshold
- Location: tests/visual/regression.spec.ts:36:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1  | /**
  2  |  * Visual Test: Regression Comparison
  3  |  *
  4  |  * Compares current screenshots against saved baselines and fails if
  5  |  * the visual diff exceeds 0.1% (maxDiffPixelRatio: 0.001).
  6  |  *
  7  |  * Prerequisite: Run baseline.spec.ts first to generate baseline screenshots.
  8  |  *
  9  |  * Usage:
  10 |  *   npx playwright test e2e/tests/visual/regression.spec.ts
  11 |  */
  12 | import { test, expect } from '@playwright/test'
  13 | 
  14 | const REGRESSION_THRESHOLD = 0.001 // 0.1% max pixel diff
  15 | 
  16 | test.describe('visual-regression', () => {
  17 |   test('main layout matches baseline within 0.1% threshold', async ({ page }) => {
  18 |     await page.goto('/')
  19 |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  20 | 
  21 |     // Wait for WS connection to establish
  22 |     const disconnectedEl = page.getByText('Disconnected', { exact: true })
  23 |     await disconnectedEl.waitFor({ state: 'hidden', timeout: 15_000 }).catch(() => {
  24 |       // May already be hidden
  25 |     })
  26 | 
  27 |     await expect(page.locator('.bg-gray-950')).toBeVisible()
  28 | 
  29 |     // Compare against baseline
  30 |     await expect(page).toHaveScreenshot('main-layout.png', {
  31 |       fullPage: false,
  32 |       maxDiffPixelRatio: REGRESSION_THRESHOLD,
  33 |     })
  34 |   })
  35 | 
  36 |   test('command palette matches baseline within 0.1% threshold', async ({ page }) => {
> 37 |     await page.goto('/')
     |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  38 |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  39 | 
  40 |     // Wait for connection
  41 |     await page.waitForTimeout(2000)
  42 | 
  43 |     // Open command palette
  44 |     await page.keyboard.press('Control+K')
  45 |     await page.waitForTimeout(500)
  46 | 
  47 |     // Compare against baseline
  48 |     await expect(page).toHaveScreenshot('command-palette.png', {
  49 |       fullPage: false,
  50 |       maxDiffPixelRatio: REGRESSION_THRESHOLD,
  51 |     })
  52 | 
  53 |     // Close palette
  54 |     await page.keyboard.press('Escape')
  55 |     await page.waitForTimeout(300)
  56 |   })
  57 | 
  58 |   test('disconnected state matches baseline within 0.1% threshold', async ({ page }) => {
  59 |     await page.goto('/')
  60 |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  61 | 
  62 |     // The disconnected state should appear briefly before WS connects
  63 |     // If already connected, we simulate by checking the rendered state
  64 |     const disconnectedVisible = await page.getByText('Disconnected', { exact: true }).isVisible().catch(() => false)
  65 | 
  66 |     if (disconnectedVisible) {
  67 |       await expect(page).toHaveScreenshot('disconnected-state.png', {
  68 |         fullPage: false,
  69 |         maxDiffPixelRatio: REGRESSION_THRESHOLD,
  70 |       })
  71 |     }
  72 |   })
  73 | 
  74 |   test('keymap overlay matches baseline within 0.1% threshold', async ({ page }) => {
  75 |     await page.goto('/')
  76 |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  77 | 
  78 |     // Wait for connection
  79 |     await page.waitForTimeout(2000)
  80 | 
  81 |     // Open keymap overlay
  82 |     await page.keyboard.press('Control+?')
  83 |     await page.waitForTimeout(500)
  84 | 
  85 |     // Compare against baseline
  86 |     await expect(page).toHaveScreenshot('keymap-overlay.png', {
  87 |       fullPage: false,
  88 |       maxDiffPixelRatio: REGRESSION_THRESHOLD,
  89 |     })
  90 | 
  91 |     // Close overlay
  92 |     await page.keyboard.press('Escape')
  93 |     await page.waitForTimeout(300)
  94 |   })
  95 | })
  96 | 
```