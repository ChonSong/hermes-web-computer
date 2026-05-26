import { test, expect } from '@playwright/test'

test('three-column layout renders', async ({ page }) => {
  await page.goto('/')
  // Wait for connected state (no "Disconnected" visible)
  await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10000 })

  // Left panel visible with tabs (icon-only in v1.4)
  await expect(page.getByRole('button', { name: '📁' })).toBeVisible()
  await expect(page.getByRole('button', { name: '🚀' })).toBeVisible()

  // Middle panel renders a terminal tile (rounded-2xl, solid dark theme)
  await expect(page.locator('div.rounded-2xl').first()).toBeVisible({ timeout: 10000 })

  // Right panel with agent chat (tabs visible in v1.4)
  await expect(page.getByRole('button', { name: '💬 Chat' })).toBeVisible()
  await expect(page.getByRole('textbox', { name: 'Type a message...' })).toBeVisible()

  // Screenshot for vision analysis
  await page.screenshot({ path: 'e2e/test-results/layout-default.png', fullPage: true })
})
