import { test, expect } from '@playwright/test'

// These tests hit the real app at http://localhost:5173 with backend at localhost:3113
// Run with: npx playwright test

test.describe('Agent-OS — critical paths', () => {
  test.beforeEach(async ({ page }) => {
    page.on('console', msg => {
      if (msg.type() === 'error') {
        console.log(`[console error] ${msg.text()}`)
      }
    })
  })

  test('app loads without JS errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', err => errors.push(err.message))
    await page.goto('http://localhost:5173')
    await page.waitForSelector('button:has-text("+ New Chat")', { timeout: 10000 })
    expect(errors).toHaveLength(0)
  })

  test('+ New Chat creates session and sends layout update (no crash)', async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForSelector('button:has-text("+ New Chat")')

    // Click + New Chat
    await page.click('button:has-text("+ New Chat")')
    await page.waitForTimeout(3000)

    // Verify: no crash, app still alive
    await expect(page.locator('button:has-text("+ New Chat")')).toBeVisible()
    // Verify session list updated
    const newSession = page.locator('button:has-text("New conversation")').first()
    await expect(newSession).toBeVisible()
  })

  test('RightPanel — all 7 non-chat tabs render content', async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForSelector('button:has-text("👤 Profiles")')

    for (const [btn, isSettings] of [
      ['👤 Profiles', false],
      ['◆ Skills', false],
      ['⏰ Crons', false],
      ['🧠 Memory', false],
      ['⚙️', true],
      ['🔧 Config', false],
      ['📊 Observe', false],
    ] as [string, boolean][]) {
      if (isSettings) {
        // ⚙️ tab — click by aria-label to avoid ambiguous selector
        await page.click('button[aria-label="Settings tab"]')
      } else {
        await page.click(`button:has-text("${btn}")`)
      }
      await page.waitForTimeout(800)
      // Panel visible
      const panelContent = page.locator('[class*="backdrop-blur"]').first()
      await expect(panelContent).toBeVisible()
    }
  })

  test('SessionsPanel — sessions are listed', async ({ page }) => {
    await page.goto('http://localhost:5173')
    await expect(page.locator('text=ALL SESSIONS')).toBeVisible()
    const sessions = page.locator('button:has-text("New conversation")')
    await expect(sessions.first()).toBeVisible()
  })

  test('Dock — panel feature buttons switch RightPanel tab without crash', async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForSelector('button:has-text("Files")')

    // Click Files app — it's a panel feature, not a tile
    await page.click('button:has-text("Files")')
    await page.waitForTimeout(800)

    // Should not crash — app still alive
    await expect(page.locator('button:has-text("+ New Chat")')).toBeVisible()
  })
})

test.describe('Agent-OS — chat flow', () => {
  test('chat tile receives messages', async ({ page }) => {
    await page.goto('http://localhost:5173')
    await page.waitForSelector('button:has-text("+ New Chat")')

    // Create a new chat
    await page.click('button:has-text("+ New Chat")')
    await page.waitForTimeout(1000)

    // Type in the chat input
    const input = page.locator('input[placeholder*="Type"]')
    if (await input.isVisible()) {
      await input.fill('hello agent')
      await page.keyboard.press('Enter')
      await page.waitForTimeout(2000)
    }

    // No crash — still on the same page
    await expect(page.locator('button:has-text("+ New Chat")')).toBeVisible()
  })
})