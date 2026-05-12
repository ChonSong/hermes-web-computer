# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/visual/baseline.spec.ts >> visual-baseline >> captures baseline of main connected layout
- Location: tests/visual/baseline.spec.ts:4:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test'
  2  | 
  3  | test.describe('visual-baseline', () => {
  4  |   test('captures baseline of main connected layout', async ({ page }) => {
> 5  |     await page.goto('/')
     |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  6  |     await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
  7  |     await expect(page.locator('.bg-gray-950').first()).toBeVisible()
  8  |     await expect(page).toHaveScreenshot('main-layout.png', { fullPage: false, maxDiffPixelRatio: 0 })
  9  |   })
  10 | 
  11 |   test('captures baseline of command palette', async ({ page }) => {
  12 |     await page.goto('/')
  13 |     await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
  14 |     await page.waitForTimeout(2000)
  15 |     await page.keyboard.press('Control+K')
  16 |     await page.waitForTimeout(500)
  17 |     await expect(page).toHaveScreenshot('command-palette.png', { fullPage: false, maxDiffPixelRatio: 0 })
  18 |     await page.keyboard.press('Escape')
  19 |     await page.waitForTimeout(300)
  20 |   })
  21 | 
  22 |   test('captures baseline of keymap overlay', async ({ page }) => {
  23 |     await page.goto('/')
  24 |     await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
  25 |     await page.waitForTimeout(2000)
  26 |     await page.keyboard.press('Control+?')
  27 |     await page.waitForTimeout(500)
  28 |     await expect(page).toHaveScreenshot('keymap-overlay.png', { fullPage: false, maxDiffPixelRatio: 0 })
  29 |     await page.keyboard.press('Escape')
  30 |     await page.waitForTimeout(300)
  31 |   })
  32 | 
  33 |   test('captures baseline with left panel hidden', async ({ page }) => {
  34 |     await page.goto('/')
  35 |     await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
  36 |     await page.waitForTimeout(2000)
  37 |     await page.keyboard.press('Control+b')
  38 |     await page.waitForTimeout(500)
  39 |     await expect(page).toHaveScreenshot('left-panel-hidden.png', { fullPage: false, maxDiffPixelRatio: 0 })
  40 |     await page.keyboard.press('Control+b')
  41 |     await page.waitForTimeout(500)
  42 |   })
  43 | })
  44 | 
```