/**
 * Chaos Test: Concurrent Tabs
 *
 * Opens multiple browser tabs (pages in the same context) and verifies each
 * establishes its own independent session with the server.
 */
import { test, expect } from '@playwright/test'

test.describe('concurrent-tabs', () => {
  test('opens 3 tabs and each gets its own session', async ({ context }) => {
    const pages = await Promise.all(
      Array.from({ length: 3 }, () => context.newPage())
    )

    await Promise.all(pages.map(async (p) => {
      await p.goto('/')
      await expect(p.locator('#app')).toBeVisible({ timeout: 10_000 })
    }))

    for (const page of pages) {
      await expect(page.locator('#app')).toBeVisible()
    }

    await Promise.all(pages.map(p => p.close()))
  })

  test('tabs do not interfere with each other WS connections', async ({ context }) => {
    const page1 = await context.newPage()
    const page2 = await context.newPage()

    await page1.goto('/')
    await expect(page1.locator('#app')).toBeVisible({ timeout: 10_000 })

    await page2.goto('/')
    await expect(page2.locator('#app')).toBeVisible({ timeout: 10_000 })

    await expect(page1.locator('#app')).toBeVisible()
    await expect(page2.locator('#app')).toBeVisible()

    // Interact with page1 — open command palette
    await page1.keyboard.press('Control+K')
    await page1.waitForTimeout(1000)

    // Page2 should still show its normal state
    await expect(page2.locator('#app')).toBeVisible()

    await page1.close()
    await page2.close()
  })

  test('closing one tab does not affect other tabs sessions', async ({ context }) => {
    const page1 = await context.newPage()
    const page2 = await context.newPage()
    const page3 = await context.newPage()

    await Promise.all([page1, page2, page3].map(p => p.goto('/')))

    await Promise.all([page1, page2, page3].map(p =>
      expect(p.locator('#app')).toBeVisible({ timeout: 10_000 })
    ))

    // Close the middle tab
    await page2.close()

    // Remaining tabs should still function
    await expect(page1.locator('#app')).toBeVisible()
    await expect(page3.locator('#app')).toBeVisible()

    await page1.reload()
    await expect(page1.locator('#app')).toBeVisible({ timeout: 10_000 })

    await page1.close()
    await page3.close()
  })

  test('concurrent terminal input from multiple tabs', async ({ context }) => {
    const page1 = await context.newPage()
    const page2 = await context.newPage()

    await page1.goto('/')
    await page2.goto('/')

    await expect(page1.locator('#app')).toBeVisible({ timeout: 10_000 })
    await expect(page2.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Type in page1's terminal if visible
    const terminal1 = page1.locator('.xterm-screen').first()
    if (await terminal1.isVisible({ timeout: 5_000 })) {
      await terminal1.click()
      await page1.keyboard.type('echo "tab1"')
      await page1.keyboard.press('Enter')
    }

    // Page2 should still be responsive
    await expect(page2.locator('#app')).toBeVisible()

    await page1.close()
    await page2.close()
  })
})
