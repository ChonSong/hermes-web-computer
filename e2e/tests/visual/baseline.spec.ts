/**
 * Visual Test: Baseline Screenshots
 *
 * Generates baseline screenshots for all major views of the application.
 * Run with `--update-snapshots` to refresh baselines after intentional UI changes.
 *
 * Usage:
 *   npx playwright test e2e/tests/visual/baseline.spec.ts --update-snapshots
 */
import { test, expect } from '@playwright/test'

test.describe('visual-baseline', () => {
  test('captures baseline of disconnected state', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for the disconnected screen to render
    await expect(page.getByText('Disconnected', { exact: true })).toBeVisible({ timeout: 10_000 })

    // Take screenshot of disconnected state
    await expect(page).toHaveScreenshot('disconnected-state.png', {
      fullPage: false,
      maxDiffPixelRatio: 0,
    })
  })

  test('captures baseline of main connected layout', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for WS connection to establish and layout to render
    const disconnectedEl = page.getByText('Disconnected', { exact: true })
    await disconnectedEl.waitFor({ state: 'hidden', timeout: 15_000 }).catch(() => {
      // May already be hidden
    })

    // Ensure the three-panel layout is rendered
    await expect(page.locator('.bg-gray-950')).toBeVisible()

    // Take screenshot of the full layout
    await expect(page).toHaveScreenshot('main-layout.png', {
      fullPage: false,
      maxDiffPixelRatio: 0,
    })
  })

  test('captures baseline of command palette', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for connection
    await page.waitForTimeout(2000)

    // Open command palette
    await page.keyboard.press('Control+K')
    await page.waitForTimeout(500)

    // Take screenshot of command palette overlay
    await expect(page).toHaveScreenshot('command-palette.png', {
      fullPage: false,
      maxDiffPixelRatio: 0,
    })

    // Close palette
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  })

  test('captures baseline of keymap overlay', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for connection
    await page.waitForTimeout(2000)

    // Open keymap overlay
    await page.keyboard.press('Control+?')
    await page.waitForTimeout(500)

    // Take screenshot of keymap overlay
    await expect(page).toHaveScreenshot('keymap-overlay.png', {
      fullPage: false,
      maxDiffPixelRatio: 0,
    })

    // Close overlay
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  })

  test('captures baseline with left panel hidden', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for connection
    await page.waitForTimeout(2000)

    // Toggle left panel (Ctrl+B)
    await page.keyboard.press('Control+b')
    await page.waitForTimeout(500)

    // Take screenshot
    await expect(page).toHaveScreenshot('left-panel-hidden.png', {
      fullPage: false,
      maxDiffPixelRatio: 0,
    })

    // Restore
    await page.keyboard.press('Control+b')
    await page.waitForTimeout(500)
  })
})
