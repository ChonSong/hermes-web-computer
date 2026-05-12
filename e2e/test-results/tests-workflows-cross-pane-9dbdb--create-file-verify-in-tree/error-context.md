# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/workflows/cross-panel.spec.ts >> Cross-Panel Interaction Workflow >> multi-step cross-panel workflow: create directory, create file, verify in tree
- Location: tests/workflows/cross-panel.spec.ts:93:3

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
  3   | test.describe('Cross-Panel Interaction Workflow', () => {
  4   |   test.beforeEach(async ({ page }) => {
> 5   |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  6   |     // Wait for WebSocket connection
  7   |     await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10_000 })
  8   |     await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 10_000 })
  9   |   })
  10  | 
  11  |   test('create file via terminal, verify it appears in file tree, open in editor', async ({ page }) => {
  12  |     // Step 1: Focus the terminal in the middle column
  13  |     const terminalTile = page.locator('div[tabindex="0"]').first()
  14  |     await expect(terminalTile).toBeVisible({ timeout: 10_000 })
  15  |     await terminalTile.click()
  16  | 
  17  |     // Step 2: Create a file via terminal command
  18  |     const testFile = '/tmp/cross-panel-test.txt'
  19  |     const testContent = 'Hello from terminal, visible in editor!'
  20  |     await page.keyboard.type(`echo "${testContent}" > ${testFile}`)
  21  |     await page.keyboard.press('Enter')
  22  |     await page.waitForTimeout(1500)
  23  | 
  24  |     // Step 3: Verify the file was created by reading it back
  25  |     await page.keyboard.type(`cat ${testFile}`)
  26  |     await page.keyboard.press('Enter')
  27  |     await page.waitForTimeout(1500)
  28  | 
  29  |     // Step 4: Switch to Files tab
  30  |     await page.getByRole('button', { name: 'Files' }).click()
  31  |     await page.waitForTimeout(1500)
  32  | 
  33  |     // Step 5: Navigate to /tmp directory
  34  |     await page.getByRole('button', { name: '/' }).first().click()
  35  |     await page.waitForTimeout(1500)
  36  | 
  37  |     // Find and click on tmp directory
  38  |     const tmpEntry = page.locator('li:has-text("tmp")').first()
  39  |     if (await tmpEntry.isVisible({ timeout: 5_000 })) {
  40  |       await tmpEntry.click()
  41  |     } else {
  42  |       // Skip remaining assertions if tmp is not visible
  43  |       return
  44  |     }
  45  |     await page.waitForTimeout(1500)
  46  | 
  47  |     // Step 6: Verify the file appears in the file tree
  48  |     const createdFile = page.locator('li:has-text("cross-panel-test.txt")').first()
  49  |     await expect(createdFile).toBeVisible({ timeout: 5_000 })
  50  | 
  51  |     // Step 7: Click file to open in editor
  52  |     await createdFile.click()
  53  | 
  54  |     // Step 8: Verify content matches what was written in terminal
  55  |     // The Monaco editor should display the file content
  56  |     await expect(page.locator('div[class*="monaco"]').first()).toBeVisible({ timeout: 5_000 })
  57  |     await expect(page.locator('div[class*="monaco"]')).toContainText(testContent, { timeout: 5_000 })
  58  |   })
  59  | 
  60  |   test('create file via terminal, verify via file stat', async ({ page }) => {
  61  |     // Focus terminal
  62  |     const terminalTile = page.locator('div[tabindex="0"]').first()
  63  |     await terminalTile.click()
  64  | 
  65  |     // Create a unique file
  66  |     const uniqueName = `cross-panel-${Date.now()}.txt`
  67  |     await page.keyboard.type(`touch /tmp/${uniqueName}`)
  68  |     await page.keyboard.press('Enter')
  69  |     await page.waitForTimeout(1000)
  70  | 
  71  |     // Verify file exists via terminal
  72  |     await page.keyboard.type(`ls -la /tmp/${uniqueName}`)
  73  |     await page.keyboard.press('Enter')
  74  |     await page.waitForTimeout(1000)
  75  | 
  76  |     // Navigate to file tree and verify file appears
  77  |     await page.getByRole('button', { name: 'Files' }).click()
  78  |     await page.waitForTimeout(1500)
  79  |     await page.getByRole('button', { name: '/' }).first().click()
  80  |     await page.waitForTimeout(1500)
  81  | 
  82  |     const tmpEntry = page.locator('li:has-text("tmp")').first()
  83  |     if (await tmpEntry.isVisible({ timeout: 5_000 })) {
  84  |       await tmpEntry.click()
  85  |       await page.waitForTimeout(1500)
  86  | 
  87  |       // Verify file appears in tree
  88  |       const fileInTree = page.locator(`li:has-text("${uniqueName}")`).first()
  89  |       await expect(fileInTree).toBeVisible({ timeout: 5_000 })
  90  |     }
  91  |   })
  92  | 
  93  |   test('multi-step cross-panel workflow: create directory, create file, verify in tree', async ({ page }) => {
  94  |     // Step 1: Create a directory via terminal
  95  |     const terminalTile = page.locator('div[tabindex="0"]').first()
  96  |     await terminalTile.click()
  97  | 
  98  |     const dirName = `e2e-cross-panel-${Date.now()}`
  99  |     await page.keyboard.type(`mkdir /tmp/${dirName}`)
  100 |     await page.keyboard.press('Enter')
  101 |     await page.waitForTimeout(1000)
  102 | 
  103 |     // Step 2: Create a file in that directory via terminal
  104 |     const fileName = 'test-file.txt'
  105 |     await page.keyboard.type(`echo "created-by-terminal" > /tmp/${dirName}/${fileName}`)
```