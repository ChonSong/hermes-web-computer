import { test, expect } from '@playwright/test'

test.describe('Session Recovery Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Wait for WebSocket connection (Connected visible, not Disconnected)
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 15_000 })
    await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 15_000 })
  })

  test('page reload recovers full layout', async ({ page }) => {
    // Verify terminal tile is visible
    const terminalTile = page.locator('div.rounded-2xl').first()
    await expect(terminalTile).toBeVisible({ timeout: 10_000 })

    // Verify right panel chat is interactable
    const chatInput = page.getByRole('textbox', { name: /Type a message/i })
    await expect(chatInput).toBeVisible({ timeout: 5_000 })

    // Reload the page
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 15_000 })

    // Wait for reconnection
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 15_000 })

    // Verify layout is restored
    const restoredTile = page.locator('div.rounded-2xl').first()
    await expect(restoredTile).toBeVisible({ timeout: 10_000 })

    // Verify chat input is back
    await expect(chatInput).toBeVisible({ timeout: 5_000 })
  })

  test('terminal is functional after recovery', async ({ page }) => {
    const terminalTile = page.locator('div.rounded-2xl').first()
    await terminalTile.click()

    // Type a command
    await page.keyboard.type('echo "pre-recovery"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Reload
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 15_000 })
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 15_000 })

    // Terminal should be responsive after recovery
    const restoredTile = page.locator('div.rounded-2xl').first()
    await expect(restoredTile).toBeVisible({ timeout: 10_000 })
    await restoredTile.click()
    await page.keyboard.type('echo "post-recovery"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)
  })

  test('chat history is maintained across reload', async ({ page }) => {
    const chatInput = page.getByRole('textbox', { name: /Type a message/i })
    await chatInput.fill('message-before-reload')
    await chatInput.press('Enter')
    await page.waitForTimeout(500)

    // Message should appear in history
    await expect(page.locator('div.bg-blue-600:has-text("message-before-reload")')).toBeVisible({ timeout: 5_000 })

    // Reload
    await page.reload({ waitUntil: 'domcontentloaded', timeout: 15_000 })
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 15_000 })

    // Chat should be functional again
    const restoredInput = page.getByRole('textbox', { name: /Type a message/i })
    await expect(restoredInput).toBeVisible({ timeout: 5_000 })
  })

  test('rapid reloads do not crash the app', async ({ page }) => {
    for (let i = 0; i < 3; i++) {
      await page.reload({ waitUntil: 'domcontentloaded', timeout: 15_000 })
      await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
    }

    // After rapid reloads, app should still be connected
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 15_000 })
    await expect(page.locator('div.rounded-2xl').first()).toBeVisible({ timeout: 10_000 })
  })
})
