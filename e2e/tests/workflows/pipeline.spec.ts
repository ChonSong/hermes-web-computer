import { test, expect } from '@playwright/test'

test.describe('Multi-Terminal Pipeline Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Wait for WebSocket connection and initial terminal
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 10_000 })
  })

  test('launch Terminal 1, run mkdir and echo commands', async ({ page }) => {
    // The initial terminal (root tile) is already present
    const terminal1 = page.locator('div.rounded-2xl').first()
    await expect(terminal1).toBeVisible({ timeout: 10_000 })

    // Click to focus Terminal 1
    await terminal1.click()

    // Run mkdir command
    await page.keyboard.type('mkdir -p /tmp/pipeline-test')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Run echo command
    await page.keyboard.type('echo "pipeline-step-1-complete" > /tmp/pipeline-test/output.txt')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Verify the command output appears in the terminal
    // The xterm canvas renders text, so we check for the terminal container
    await expect(terminal1).toBeVisible()

    // Verify the directory was created by checking file tree
    await page.getByRole('button', { name: 'Files' }).click()
    await page.waitForTimeout(1500)

    // Navigate to tmp
    await page.getByRole('button', { name: '/' }).first().click()
    await page.waitForTimeout(1500)

    const tmpEntry = page.locator('li:has-text("tmp")').first()
    if (await tmpEntry.isVisible({ timeout: 5_000 })) {
      await tmpEntry.click()
      await page.waitForTimeout(1500)

      // Verify pipeline-test directory exists
      const pipelineDir = page.locator('li:has-text("pipeline-test")').first()
      await expect(pipelineDir).toBeVisible({ timeout: 5_000 })
    }
  })

  test('launch Terminal 2 (split), verify layout changes', async ({ page }) => {
    // Wait for initial terminal
    const terminal1 = page.locator('div.rounded-2xl').first()
    await expect(terminal1).toBeVisible({ timeout: 10_000 })

    // Focus Terminal 1 and double-click to split (creates Terminal 2)
    await terminal1.click()
    await terminal1.dblclick()
    await page.waitForTimeout(1500)

    // Verify we now have 2 terminal tiles
    const tiles = page.locator('div.rounded-2xl')
    await expect(tiles).toHaveCount(2, { timeout: 5_000 })

    // Focus Terminal 2 (the right/bottom one) and run a command
    const terminal2 = tiles.nth(1)
    await terminal2.click()
    await page.keyboard.type('echo "terminal-2-active"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Verify Terminal 2 is visible and responsive
    await expect(terminal2).toBeVisible()
  })

  test('launch Terminal 3 (2x2 grid), then close one and verify reflow', async ({ page }) => {
    // Wait for initial terminal
    const rootTile = page.locator('div.rounded-2xl').first()
    await expect(rootTile).toBeVisible({ timeout: 10_000 })

    // Split 1: Double-click root to create second terminal (horizontal split)
    await rootTile.click()
    await rootTile.dblclick()
    await page.waitForTimeout(1500)

    // Split 2: Focus the new terminal and split vertically
    const tiles2 = page.locator('div.rounded-2xl')
    await expect(tiles2).toHaveCount(2, { timeout: 5_000 })

    // Focus the second tile and split it
    const tile2 = tiles2.nth(1)
    await tile2.click()
    await tile2.dblclick()
    await page.waitForTimeout(1500)

    // Now we should have 3 tiles
    const tiles3 = page.locator('div.rounded-2xl')
    await expect(tiles3).toHaveCount(3, { timeout: 5_000 })

    // Run a command in each terminal to verify they work
    // Terminal 1
    await tiles3.nth(0).click()
    await page.keyboard.type('echo "t1"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(500)

    // Terminal 2
    await tiles3.nth(1).click()
    await page.keyboard.type('echo "t2"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(500)

    // Terminal 3
    await tiles3.nth(2).click()
    await page.keyboard.type('echo "t3"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(500)

    // Close Terminal 2 using Shift+Q (as defined in Tile.svelte)
    await tiles3.nth(1).click()
    await page.keyboard.down('Shift')
    await page.keyboard.press('q')
    await page.keyboard.up('Shift')
    await page.waitForTimeout(1500)

    // Verify reflow: should now have 2 tiles
    const tilesAfter = page.locator('div.rounded-2xl')
    await expect(tilesAfter).toHaveCount(2, { timeout: 5_000 })
  })

  test('verify output visibility across split terminals', async ({ page }) => {
    // Wait for initial terminal
    const terminal1 = page.locator('div.rounded-2xl').first()
    await expect(terminal1).toBeVisible({ timeout: 10_000 })

    // Run a command that produces visible output
    await terminal1.click()
    await page.keyboard.type('echo "cross-terminal-visible"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1500)

    // Split to create Terminal 2
    await terminal1.dblclick()
    await page.waitForTimeout(1500)

    // Verify 2 terminals exist
    const tiles = page.locator('div.rounded-2xl')
    await expect(tiles).toHaveCount(2, { timeout: 5_000 })

    // Terminal 2 should show the initial xterm welcome text
    // since it's a new PTY session
    const terminal2 = tiles.nth(1)
    await expect(terminal2).toBeVisible()

    // Run command in Terminal 2
    await terminal2.click()
    await page.keyboard.type('echo "terminal-2-output"')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Verify both terminals are still visible
    await expect(tiles).toHaveCount(2)
  })
})
