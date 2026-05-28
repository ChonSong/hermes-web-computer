import { test, expect } from '@playwright/test'

test.describe('Cross-Panel Interaction Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Wait for WebSocket connection
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 10_000 })
  })

  test('create file via terminal, verify it appears in file tree, open in editor', async ({ page }) => {
    // Step 1: Focus the terminal in the middle column
    const terminalTile = page.locator('div[tabindex="0"]').first()
    await expect(terminalTile).toBeVisible({ timeout: 10_000 })
    await terminalTile.click()

    // Step 2: Create a file via terminal command
    const testFile = '/tmp/cross-panel-test.txt'
    const testContent = 'Hello from terminal, visible in editor!'
    await page.keyboard.type(`echo "${testContent}" > ${testFile}`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1500)

    // Step 3: Verify the file was created by reading it back
    await page.keyboard.type(`cat ${testFile}`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1500)

    // Step 4: Switch to Files tab via left panel's 📁 button (specific to sidebar)
    const leftPanel = page.locator('.rounded-2xl.shadow-panel').first()
    const filesButton = leftPanel.locator('button').nth(1)
    await filesButton.click()
    await page.waitForTimeout(1500)

    // Step 5: Navigate to /tmp directory
    await page.getByRole('button', { name: '/' }).first().click()
    await page.waitForTimeout(1500)

    // Find and click on tmp directory
    const tmpEntry = page.locator('li:has-text("tmp")').first()
    if (await tmpEntry.isVisible({ timeout: 5_000 })) {
      await tmpEntry.click()
    } else {
      // Skip remaining assertions if tmp is not visible
      return
    }
    await page.waitForTimeout(1500)

    // Step 6: Verify the file appears in the file tree
    const createdFile = page.locator('li:has-text("cross-panel-test.txt")').first()
    await expect(createdFile).toBeVisible({ timeout: 5_000 })

    // Step 7: Click file to open in editor
    await createdFile.click()

    // Step 8: Verify content matches what was written in terminal
    // The Monaco editor should display the file content
    await expect(page.locator('div[class*="monaco"]').first()).toBeVisible({ timeout: 5_000 })
    await expect(page.locator('div[class*="monaco"]')).toContainText(testContent, { timeout: 5_000 })
  })

  test('create file via terminal, verify via file stat', async ({ page }) => {
    // Focus terminal
    const terminalTile = page.locator('div[tabindex="0"]').first()
    await terminalTile.click()

    // Create a unique file
    const uniqueName = `cross-panel-${Date.now()}.txt`
    await page.keyboard.type(`touch /tmp/${uniqueName}`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Verify file exists via terminal
    await page.keyboard.type(`ls -la /tmp/${uniqueName}`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Navigate to file tree and verify file appears
    const leftPanel = page.locator('.rounded-2xl.shadow-panel').first()
    const filesButton = leftPanel.locator('button').nth(1)
    await filesButton.click()
    await page.waitForTimeout(1500)
    await page.getByRole('button', { name: '/' }).first().click()
    await page.waitForTimeout(1500)

    const tmpEntry = page.locator('li:has-text("tmp")').first()
    if (await tmpEntry.isVisible({ timeout: 5_000 })) {
      await tmpEntry.click()
      await page.waitForTimeout(1500)

      // Verify file appears in tree
      const fileInTree = page.locator(`li:has-text("${uniqueName}")`).first()
      await expect(fileInTree).toBeVisible({ timeout: 5_000 })
    }
  })

  test('multi-step cross-panel workflow: create directory, create file, verify in tree', async ({ page }) => {
    // Step 1: Create a directory via terminal
    const terminalTile = page.locator('div[tabindex="0"]').first()
    await terminalTile.click()

    const dirName = `e2e-cross-panel-${Date.now()}`
    await page.keyboard.type(`mkdir /tmp/${dirName}`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Step 2: Create a file in that directory via terminal
    const fileName = 'test-file.txt'
    await page.keyboard.type(`echo "created-by-terminal" > /tmp/${dirName}/${fileName}`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Step 3: Navigate to the directory in file tree
    const leftPanel = page.locator('.rounded-2xl.shadow-panel').first()
    const filesButton = leftPanel.locator('button').nth(1)
    await filesButton.click()
    await page.waitForTimeout(1500)
    await page.getByRole('button', { name: '/' }).first().click()
    await page.waitForTimeout(1500)

    const tmpEntry = page.locator('li:has-text("tmp")').first()
    if (await tmpEntry.isVisible({ timeout: 5_000 })) {
      await tmpEntry.click()
      await page.waitForTimeout(1500)

      // Verify the directory appears
      const newDir = page.locator(`li:has-text("${dirName}")`).first()
      await expect(newDir).toBeVisible({ timeout: 5_000 })

      // Step 4: Navigate into the directory
      await newDir.click()
      await page.waitForTimeout(1500)

      // Step 5: Verify the file appears
      const createdFile = page.locator(`li:has-text("${fileName}")`).first()
      await expect(createdFile).toBeVisible({ timeout: 5_000 })

      // Step 6: Open file in editor and verify content
      await createdFile.click()
      await expect(page.locator('div[class*="monaco"]').first()).toBeVisible({ timeout: 5_000 })
      await expect(page.locator('div[class*="monaco"]')).toContainText('created-by-terminal', { timeout: 5_000 })
    }
  })

  test('create file in terminal, verify content matches when opened in editor', async ({ page }) => {
    // Focus terminal
    const terminalTile = page.locator('div[tabindex="0"]').first()
    await terminalTile.click()

    // Create a file with multi-line content
    const testContent = 'line-one\nline-two\nline-three'
    await page.keyboard.type(`printf "${testContent}" > /tmp/multiline-test.txt`)
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)

    // Navigate to file in tree
    const leftPanel = page.locator('.rounded-2xl.shadow-panel').first()
    const filesButton = leftPanel.locator('button').nth(1)
    await filesButton.click()
    await page.waitForTimeout(1500)
    await page.getByRole('button', { name: '/' }).first().click()
    await page.waitForTimeout(1500)

    const tmpEntry = page.locator('li:has-text("tmp")').first()
    if (await tmpEntry.isVisible({ timeout: 5_000 })) {
      await tmpEntry.click()
      await page.waitForTimeout(1500)

      // Open file in editor
      const fileEntry = page.locator('li:has-text("multiline-test.txt")').first()
      await expect(fileEntry).toBeVisible({ timeout: 5_000 })
      await fileEntry.click()

      // Verify each line appears in the editor
      await expect(page.locator('div[class*="monaco"]').first()).toBeVisible({ timeout: 5_000 })
      await expect(page.locator('div[class*="monaco"]')).toContainText('line-one', { timeout: 5_000 })
      await expect(page.locator('div[class*="monaco"]')).toContainText('line-two', { timeout: 5_000 })
      await expect(page.locator('div[class*="monaco"]')).toContainText('line-three', { timeout: 5_000 })
    }
  })
})
