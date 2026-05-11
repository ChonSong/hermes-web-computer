import { test, expect } from '@playwright/test'

test.describe('visual-baseline', () => {
  test('captures baseline of main connected layout', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('.bg-gray-950').first()).toBeVisible()
    await expect(page).toHaveScreenshot('main-layout.png', { fullPage: false, maxDiffPixelRatio: 0 })
  })

  test('captures baseline of command palette', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
    await page.waitForTimeout(2000)
    await page.keyboard.press('Control+K')
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('command-palette.png', { fullPage: false, maxDiffPixelRatio: 0 })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  })

  test('captures baseline of keymap overlay', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
    await page.waitForTimeout(2000)
    await page.keyboard.press('Control+?')
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('keymap-overlay.png', { fullPage: false, maxDiffPixelRatio: 0 })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)
  })

  test('captures baseline with left panel hidden', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('div.h-screen').first()).toBeVisible({ timeout: 10_000 })
    await page.waitForTimeout(2000)
    await page.keyboard.press('Control+b')
    await page.waitForTimeout(500)
    await expect(page).toHaveScreenshot('left-panel-hidden.png', { fullPage: false, maxDiffPixelRatio: 0 })
    await page.keyboard.press('Control+b')
    await page.waitForTimeout(500)
  })
})
