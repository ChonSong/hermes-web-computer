import { test, expect } from '@playwright/test'

test('three-column layout renders', async ({ page }) => {
  await page.goto('/')
  // Wait for connected state (no "Disconnected" visible)
  await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10000 })

  // Left panel visible with tabs
  await expect(page.getByText('📁 Files')).toBeVisible()
  await expect(page.getByText('🚀 Apps')).toBeVisible()

  // Middle panel renders a terminal tile (border-blue-500)
  await expect(page.locator('div.border-blue-500').first()).toBeVisible({ timeout: 10000 })

  // Right panel with agent chat
  await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible()
  await expect(page.getByRole('textbox', { name: 'Type a message...' })).toBeVisible()

  // Screenshot for vision analysis
  await page.screenshot({ path: 'e2e/test-results/layout-default.png', fullPage: true })
})
