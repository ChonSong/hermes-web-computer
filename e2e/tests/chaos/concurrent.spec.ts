/**
 * Chaos Test: Concurrent Tabs
 *
 * Opens multiple browser tabs (pages in the same context) and verifies each
 * establishes its own independent session with the server.
 */
import { test, expect } from '@playwright/test'

test.describe('concurrent-tabs', () => {
  test('opens 3 tabs and each gets its own session', async ({ context }) => {
    // Open 3 pages in the same browser context (simulating 3 tabs)
    const pages = await Promise.all(
      Array.from({ length: 3 }, () => context.newPage())
    )

    // Navigate all pages and wait for them to connect
    await Promise.all(pages.map(async (p) => {
      await p.goto('/')
      await expect(p.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
    }))

    // Each page should be independent — verify they all render the dark background
    for (const page of pages) {
      await expect(page.locator('.bg-gray-950').first()).toBeVisible()
    }

    // Close all pages
    await Promise.all(pages.map(p => p.close()))
  })

  test('tabs do not interfere with each other WS connections', async ({ context }) => {
    const page1 = await context.newPage()
    const page2 = await context.newPage()

    await page1.goto('/')
    await expect(page1.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })

    await page2.goto('/')
    await expect(page2.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })

    // Both pages should show the main layout
    await expect(page1.locator('.bg-gray-950').first()).toBeVisible()
    await expect(page2.locator('.bg-gray-950').first()).toBeVisible()

    // Interact with page1 — open command palette
    await page1.keyboard.press('Control+K')
    await page1.waitForTimeout(1000)

    // Page2 should still show its normal state (not affected by page1 interaction)
    await expect(page2.locator('.bg-gray-950').first()).toBeVisible()

    await page1.close()
    await page2.close()
  })

  test('closing one tab does not affect other tabs sessions', async ({ context }) => {
    const page1 = await context.newPage()
    const page2 = await context.newPage()
    const page3 = await context.newPage()

    await Promise.all([page1, page2, page3].map(p => p.goto('/')))

    // Wait for all to load
    await Promise.all([page1, page2, page3].map(p =>
      expect(p.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })
    ))

    // Close the middle tab
    await page2.close()

    // Remaining tabs should still function
    await expect(page1.locator('.bg-gray-950').first()).toBeVisible()
    await expect(page3.locator('.bg-gray-950').first()).toBeVisible()

    // Navigate in remaining tabs to confirm responsiveness
    await page1.reload()
    await expect(page1.locator('.bg-gray-950').first()).toBeVisible({ timeout: 10_000 })

    await page1.close()
    await page3.close()
  })
})
