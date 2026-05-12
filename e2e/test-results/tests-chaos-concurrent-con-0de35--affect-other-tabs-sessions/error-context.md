# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/chaos/concurrent.spec.ts >> concurrent-tabs >> closing one tab does not affect other tabs sessions
- Location: tests/chaos/concurrent.spec.ts:56:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1  | /**
  2  |  * Chaos Test: Concurrent Tabs
  3  |  *
  4  |  * Opens multiple browser tabs (pages in the same context) and verifies each
  5  |  * establishes its own independent session with the server.
  6  |  */
  7  | import { test, expect } from '@playwright/test'
  8  | 
  9  | test.describe('concurrent-tabs', () => {
  10 |   test('opens 3 tabs and each gets its own session', async ({ context }) => {
  11 |     // Open 3 pages in the same browser context (simulating 3 tabs)
  12 |     const pages = await Promise.all(
  13 |       Array.from({ length: 3 }, () => context.newPage())
  14 |     )
  15 | 
  16 |     // Navigate all pages and wait for them to connect
  17 |     await Promise.all(pages.map(async (p) => {
  18 |       await p.goto('/')
  19 |       await expect(p.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
  20 |     }))
  21 | 
  22 |     // Each page should be independent — verify they all render the dark background
  23 |     for (const page of pages) {
  24 |       await expect(page.locator('.bg-gray-950').first()).toBeVisible()
  25 |     }
  26 | 
  27 |     // Close all pages
  28 |     await Promise.all(pages.map(p => p.close()))
  29 |   })
  30 | 
  31 |   test('tabs do not interfere with each other WS connections', async ({ context }) => {
  32 |     const page1 = await context.newPage()
  33 |     const page2 = await context.newPage()
  34 | 
  35 |     await page1.goto('/')
  36 |     await expect(page1.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })
  37 | 
  38 |     await page2.goto('/')
  39 |     await expect(page2.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })
  40 | 
  41 |     // Both pages should show the main layout
  42 |     await expect(page1.locator('.bg-gray-950').first()).toBeVisible()
  43 |     await expect(page2.locator('.bg-gray-950').first()).toBeVisible()
  44 | 
  45 |     // Interact with page1 — open command palette
  46 |     await page1.keyboard.press('Control+K')
  47 |     await page1.waitForTimeout(1000)
  48 | 
  49 |     // Page2 should still show its normal state (not affected by page1 interaction)
  50 |     await expect(page2.locator('.bg-gray-950').first()).toBeVisible()
  51 | 
  52 |     await page1.close()
  53 |     await page2.close()
  54 |   })
  55 | 
  56 |   test('closing one tab does not affect other tabs sessions', async ({ context }) => {
  57 |     const page1 = await context.newPage()
  58 |     const page2 = await context.newPage()
  59 |     const page3 = await context.newPage()
  60 | 
> 61 |     await Promise.all([page1, page2, page3].map(p => p.goto('/')))
     |                                                        ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  62 | 
  63 |     // Wait for all to load
  64 |     await Promise.all([page1, page2, page3].map(p =>
  65 |       expect(p.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })
  66 |     ))
  67 | 
  68 |     // Close the middle tab
  69 |     await page2.close()
  70 | 
  71 |     // Remaining tabs should still function
  72 |     await expect(page1.locator('.bg-gray-950').first()).toBeVisible()
  73 |     await expect(page3.locator('.bg-gray-950').first()).toBeVisible()
  74 | 
  75 |     // Navigate in remaining tabs to confirm responsiveness
  76 |     await page1.reload()
  77 |     await expect(page1.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })
  78 | 
  79 |     await page1.close()
  80 |     await page3.close()
  81 |   })
  82 | })
  83 | 
```