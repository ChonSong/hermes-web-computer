# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/02-resize.spec.ts >> column resizing works
- Location: tests/02-resize.spec.ts:3:1

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test'
  2  | 
  3  | test('column resizing works', async ({ page }) => {
> 4  |   await page.goto('/')
     |              ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  5  |   await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10000 })
  6  |   
  7  |   // Get initial left panel width
  8  |   const leftPanel = page.locator('div').filter({ hasText: '📁 Files' }).first()
  9  |   const initialBox = await leftPanel.boundingBox()
  10 |   
  11 |   // Drag left resize handle 100px wider
  12 |   // ResizeHandle uses cursor-ew-resize class — first handle is the left one
  13 |   const leftHandle = page.locator('.cursor-ew-resize').first()
  14 |   const handleBox = await leftHandle.boundingBox()
  15 |   
  16 |   if (handleBox) {
  17 |     await page.mouse.move(handleBox.x + handleBox.width / 2, handleBox.y + handleBox.height / 2)
  18 |     await page.mouse.down()
  19 |     await page.mouse.move(handleBox.x + 100, handleBox.y + handleBox.height / 2)
  20 |     await page.mouse.up()
  21 |     
  22 |     // Verify width changed
  23 |     const newBox = await leftPanel.boundingBox()
  24 |     expect(newBox!.width).toBeGreaterThan(initialBox!.width - 5) // tolerance
  25 |   }
  26 |   
  27 |   // Screenshot
  28 |   await page.screenshot({ path: 'e2e/test-results/layout-resized.png' })
  29 | })
  30 | 
```