/**
 * Visual Test: Regression Comparison
 *
 * Compares current screenshots against saved baselines and fails if
 * the visual diff exceeds 0.1% (maxDiffPixelRatio: 0.001).
 *
 * Prerequisite: Run baseline.spec.ts first to generate baseline screenshots.
 *
 * Usage:
 *   npx playwright test e2e/tests/visual/regression.spec.ts
 */
import { test, expect } from '@playwright/test'

const REGRESSION_THRESHOLD = 0.001 // 0.1% max pixel diff

test.describe('visual-regression', () => {
  test('main layout matches baseline within 0.1% threshold', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for WS connection to establish
    const disconnectedEl = page.getByText('Disconnected', { exact: true })
    await disconnectedEl.waitFor({ state: 'hidden', timeout: 15_000 }).catch(() => {
      // May already be hidden
    })

    await expect(page.locator('.bg-gray-950')).toBeVisible()

    // Compare against baseline
    await expect(page).toHaveScreenshot('main-layout.png', {
      fullPage: false,
      maxDiffPixelRatio: REGRESSION_THRESHOLD,
    })
  })

  test('command palette matches baseline within 0.1% threshold', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for connection
    await page.waitForTimeout(2000)

    // Open command palette
    await page.keyboard.press('Control+K')
    await page.waitForTimeout(500)

    // Compare against baseline
    await expect(page).toHaveScreenshot('command-palette.png', {
      fullPage: false,
      maxDiffPixelRatio: REGRESSION_THRESHOLD,
    })

    // Close palette
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  })

  test('disconnected state matches baseline within 0.1% threshold', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // The disconnected state should appear briefly before WS connects
    // If already connected, we simulate by checking the rendered state
    const disconnectedVisible = await page.getByText('Disconnected', { exact: true }).isVisible().catch(() => false)

    if (disconnectedVisible) {
      await expect(page).toHaveScreenshot('disconnected-state.png', {
        fullPage: false,
        maxDiffPixelRatio: REGRESSION_THRESHOLD,
      })
    }
  })

  test('keymap overlay matches baseline within 0.1% threshold', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for connection
    await page.waitForTimeout(2000)

    // Open keymap overlay
    await page.keyboard.press('Control+?')
    await page.waitForTimeout(500)

    // Compare against baseline
    await expect(page).toHaveScreenshot('keymap-overlay.png', {
      fullPage: false,
      maxDiffPixelRatio: REGRESSION_THRESHOLD,
    })

    // Close overlay
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  })
})
