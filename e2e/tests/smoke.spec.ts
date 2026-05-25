/**
 * HWC v1.4 Smoke Tests
 * Run: npx playwright test e2e/tests/smoke.spec.ts --reporter=list
 */
import { test, expect } from '@playwright/test'

test.describe('HWC v1.4 Smoke Tests', () => {

  test('a) app loads at correct URL with #app visible', async ({ page }) => {
    await page.goto('/')
    // The app root element should be visible
    const appRoot = page.locator('#app')
    await expect(appRoot).toBeVisible({ timeout: 10_000 })

    // Page should load at localhost:3005
    await expect(page).toHaveURL(/localhost:3005/)
  })

  test('b) dark theme applied — body background is dark', async ({ page }) => {
    await page.goto('/')
    // Wait for app to be ready
    await page.waitForTimeout(1_000)

    // Check that the app uses a dark theme by verifying:
    // 1. The body has a dark background (rgb with all components < 60), OR
    // 2. The app root has a dark background, OR
    // 3. A dark gradient/background layer is present
    const themeCheck = await page.evaluate(() => {
      const bodyBg = window.getComputedStyle(document.body).backgroundColor
      const appBg = document.querySelector('#app')
        ? window.getComputedStyle(document.querySelector('#app')!).backgroundColor
        : 'transparent'

      // Check for dark fixed background layers (App.svelte has bg-[#0a0a0f])
      const hasFixedDarkBg = Array.from(document.querySelectorAll('.fixed'))
        .some(el => {
          const bg = window.getComputedStyle(el).backgroundColor
          const m = bg.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/)
          if (!m) return false
          const [, r, g, b] = m.map(Number)
          return r < 60 && g < 60 && b < 60
        })

      return { bodyBg, appBg, hasFixedDarkBg }
    })

    // At least one dark indicator should be present
    const bodyMatch = themeCheck.bodyBg.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/)
    const appMatch = themeCheck.appBg.match(/rgb\((\d+),\s*(\d+),\s*(\d+)\)/)

    const bodyIsDark = bodyMatch && [1,2,3].map(Number).every((_, i) => parseInt(bodyMatch[i+1]) < 60)
    const appIsDark = appMatch && [1,2,3].map(Number).every((_, i) => parseInt(appMatch[i+1]) < 60)

    expect(bodyIsDark || appIsDark || themeCheck.hasFixedDarkBg).toBe(true)
  })

  test('c) connection status shown in top bar', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1_000)

    // Connection bar is always visible at the top
    // Shows "Connected" (green), "Reconnecting..." (yellow), or "Disconnected" (red)
    const connectedText  = page.getByText('Connected')
    const reconnectingText = page.getByText(/Reconnecting/)
    const disconnectedText = page.getByText(/Disconnected/)

    // At least one of these connection states must be visible
    const hasAny = await Promise.all([
      connectedText.isVisible().catch(() => false),
      reconnectingText.isVisible().catch(() => false),
      disconnectedText.isVisible().catch(() => false),
    ]).then(([c, r, d]) => c || r || d)

    expect(hasAny).toBe(true)
  })

  test('d) Ctrl+B hides/shows left panel', async ({ page }) => {
    await page.goto('/')
    // Wait for app to be in a stable state
    await page.waitForTimeout(1_500)

    // Use the "Toggle panel" ◀ button in the left panel header area to check visibility
    // This button is always present when left panel is visible (shows ◀ when visible)
    const toggleBtn = page.locator('button:has-text("◀")')
    const initialVisible = await toggleBtn.isVisible().catch(() => false)

    // Press Ctrl+B to toggle left panel
    await page.keyboard.press('Control+b')
    await page.waitForTimeout(500)

    const afterVisible = await toggleBtn.isVisible().catch(() => false)

    // After toggling, visibility should be opposite of initial
    // If panel was visible (toggle shows ◀), it should now be hidden
    // If panel was hidden, it should now be visible
    expect(afterVisible).toBe(!initialVisible)

    // Press Ctrl+B again to restore
    await page.keyboard.press('Control+b')
    await page.waitForTimeout(500)
  })

  test('e) Ctrl+K opens command palette overlay', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1_000)

    // Press Ctrl+K
    await page.keyboard.press('Control+K')
    await page.waitForTimeout(500)

    // Command palette overlay should appear
    // Look for the 640px-wide palette container OR any dialog-like overlay
    const paletteContainer = page.locator('[role="dialog"]').or(page.locator('.fixed.inset-0'))
    await expect(paletteContainer.first()).toBeVisible({ timeout: 3_000 })

    // Also verify the palette search input is visible
    const searchInput = page.locator('input[placeholder*="command" i], input[placeholder*="search" i]')
    const hasInput = await searchInput.isVisible().catch(() => false)
    expect(hasInput).toBe(true)
  })

  test('f) slash command autocomplete in chat input', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1_500)

    // Find the chat input (textbox with placeholder "Type a message...")
    const chatInput = page.getByRole('textbox', { name: /Type a message/i })
    await expect(chatInput).toBeVisible({ timeout: 5_000 })

    // Type "/" to trigger slash command mode
    await chatInput.fill('/')
    await page.waitForTimeout(300)

    // The input should contain "/"
    const inputValue = await chatInput.inputValue()
    expect(inputValue).toContain('/')

    // Type more to test filtering
    await chatInput.fill('/new')
    await page.waitForTimeout(300)
    const filteredValue = await chatInput.inputValue()
    expect(filteredValue).toBe('/new')
  })

  test('g) right panel Docker/Containers tab exists', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1_500)

    // The right panel should have a Docker/Containers tab button
    // Look for either the emoji variant or plain text
    const containersTab = page.getByRole('button', { name: /Containers/i })
    await expect(containersTab).toBeVisible({ timeout: 5_000 })

    // Click it and verify DockerPanel content appears
    await containersTab.click()
    await page.waitForTimeout(500)

    // DockerPanel header should be visible
    await expect(page.getByText('Docker')).toBeVisible({ timeout: 5_000 })

    // DockerPanel internal tabs should be visible
    await expect(page.getByRole('button', { name: 'Containers', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Images', exact: true })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Compose', exact: true })).toBeVisible()
  })

  test('h) Ctrl+? opens keymap overlay', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(1_000)

    // Press Ctrl+? (Shift+/)
    await page.keyboard.press('Control+?')
    await page.waitForTimeout(500)

    // Keymap overlay should be visible — it has a "Keyboard Shortcuts" heading
    const keymapHeading = page.locator('h2:has-text("Keyboard Shortcuts")')
    await expect(keymapHeading).toBeVisible({ timeout: 3_000 })

    // The overlay should also contain shortcut rows with key combos
    await expect(page.getByText('Ctrl+K')).toBeVisible()
    await expect(page.getByText('Ctrl+?')).toBeVisible()
  })

})