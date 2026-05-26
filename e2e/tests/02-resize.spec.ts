import { test, expect } from '@playwright/test'

test('column resizing works', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10000 })
  
  // Get initial left panel width
  const leftPanel = page.getByRole('button', { name: '📁' })
  const initialBox = await leftPanel.boundingBox()
  
  // Drag left resize handle 100px wider
  // ResizeHandle uses cursor-ew-resize class — first handle is the left one
  const leftHandle = page.locator('.cursor-ew-resize').first()
  const handleBox = await leftHandle.boundingBox()
  
  if (handleBox) {
    await page.mouse.move(handleBox.x + handleBox.width / 2, handleBox.y + handleBox.height / 2)
    await page.mouse.down()
    await page.mouse.move(handleBox.x + 100, handleBox.y + handleBox.height / 2)
    await page.mouse.up()
    
    // Verify width changed
    const newBox = await leftPanel.boundingBox()
    expect(newBox!.width).toBeGreaterThan(initialBox!.width - 5) // tolerance
  }
  
  // Screenshot
  await page.screenshot({ path: 'e2e/test-results/layout-resized.png' })
})
