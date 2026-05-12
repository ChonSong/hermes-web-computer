# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/workflows/pipeline.spec.ts >> Multi-Terminal Pipeline Workflow >> launch Terminal 1, run mkdir and echo commands
- Location: tests/workflows/pipeline.spec.ts:11:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1   | import { test, expect } from '@playwright/test'
  2   | 
  3   | test.describe('Multi-Terminal Pipeline Workflow', () => {
  4   |   test.beforeEach(async ({ page }) => {
> 5   |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  6   |     // Wait for WebSocket connection and initial terminal
  7   |     await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10_000 })
  8   |     await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 10_000 })
  9   |   })
  10  | 
  11  |   test('launch Terminal 1, run mkdir and echo commands', async ({ page }) => {
  12  |     // The initial terminal (root tile) is already present
  13  |     const terminal1 = page.locator('div.border-blue-500').first()
  14  |     await expect(terminal1).toBeVisible({ timeout: 10_000 })
  15  | 
  16  |     // Click to focus Terminal 1
  17  |     await terminal1.click()
  18  | 
  19  |     // Run mkdir command
  20  |     await page.keyboard.type('mkdir -p /tmp/pipeline-test')
  21  |     await page.keyboard.press('Enter')
  22  |     await page.waitForTimeout(1000)
  23  | 
  24  |     // Run echo command
  25  |     await page.keyboard.type('echo "pipeline-step-1-complete" > /tmp/pipeline-test/output.txt')
  26  |     await page.keyboard.press('Enter')
  27  |     await page.waitForTimeout(1000)
  28  | 
  29  |     // Verify the command output appears in the terminal
  30  |     // The xterm canvas renders text, so we check for the terminal container
  31  |     await expect(terminal1).toBeVisible()
  32  | 
  33  |     // Verify the directory was created by checking file tree
  34  |     await page.getByRole('button', { name: 'Files' }).click()
  35  |     await page.waitForTimeout(1500)
  36  | 
  37  |     // Navigate to tmp
  38  |     await page.getByRole('button', { name: '/' }).first().click()
  39  |     await page.waitForTimeout(1500)
  40  | 
  41  |     const tmpEntry = page.locator('li:has-text("tmp")').first()
  42  |     if (await tmpEntry.isVisible({ timeout: 5_000 })) {
  43  |       await tmpEntry.click()
  44  |       await page.waitForTimeout(1500)
  45  | 
  46  |       // Verify pipeline-test directory exists
  47  |       const pipelineDir = page.locator('li:has-text("pipeline-test")').first()
  48  |       await expect(pipelineDir).toBeVisible({ timeout: 5_000 })
  49  |     }
  50  |   })
  51  | 
  52  |   test('launch Terminal 2 (split), verify layout changes', async ({ page }) => {
  53  |     // Wait for initial terminal
  54  |     const terminal1 = page.locator('div.border-blue-500').first()
  55  |     await expect(terminal1).toBeVisible({ timeout: 10_000 })
  56  | 
  57  |     // Focus Terminal 1 and double-click to split (creates Terminal 2)
  58  |     await terminal1.click()
  59  |     await terminal1.dblclick()
  60  |     await page.waitForTimeout(1500)
  61  | 
  62  |     // Verify we now have 2 terminal tiles
  63  |     const tiles = page.locator('div.border-blue-500')
  64  |     await expect(tiles).toHaveCount(2, { timeout: 5_000 })
  65  | 
  66  |     // Focus Terminal 2 (the right/bottom one) and run a command
  67  |     const terminal2 = tiles.nth(1)
  68  |     await terminal2.click()
  69  |     await page.keyboard.type('echo "terminal-2-active"')
  70  |     await page.keyboard.press('Enter')
  71  |     await page.waitForTimeout(1000)
  72  | 
  73  |     // Verify Terminal 2 is visible and responsive
  74  |     await expect(terminal2).toBeVisible()
  75  |   })
  76  | 
  77  |   test('launch Terminal 3 (2x2 grid), then close one and verify reflow', async ({ page }) => {
  78  |     // Wait for initial terminal
  79  |     const rootTile = page.locator('div.border-blue-500').first()
  80  |     await expect(rootTile).toBeVisible({ timeout: 10_000 })
  81  | 
  82  |     // Split 1: Double-click root to create second terminal (horizontal split)
  83  |     await rootTile.click()
  84  |     await rootTile.dblclick()
  85  |     await page.waitForTimeout(1500)
  86  | 
  87  |     // Split 2: Focus the new terminal and split vertically
  88  |     const tiles2 = page.locator('div.border-blue-500')
  89  |     await expect(tiles2).toHaveCount(2, { timeout: 5_000 })
  90  | 
  91  |     // Focus the second tile and split it
  92  |     const tile2 = tiles2.nth(1)
  93  |     await tile2.click()
  94  |     await tile2.dblclick()
  95  |     await page.waitForTimeout(1500)
  96  | 
  97  |     // Now we should have 3 tiles
  98  |     const tiles3 = page.locator('div.border-blue-500')
  99  |     await expect(tiles3).toHaveCount(3, { timeout: 5_000 })
  100 | 
  101 |     // Run a command in each terminal to verify they work
  102 |     // Terminal 1
  103 |     await tiles3.nth(0).click()
  104 |     await page.keyboard.type('echo "t1"')
  105 |     await page.keyboard.press('Enter')
```