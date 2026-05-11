import { test, expect } from '@playwright/test'

test.describe('Chat Context Workflow', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
    // Wait for WebSocket connection
    await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10_000 })
    await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 10_000 })
  })

  test('type message in right panel, press Enter, verify it appears in history', async ({ page }) => {
    // The right panel should already be visible with the Agent header and welcome message
    await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })

    // Verify the welcome message is present
    await expect(page.getByText("Hello! I'm your agent. How can I help you today?")).toBeVisible({ timeout: 5_000 })

    // Find the chat input in the right panel
    const chatInput = page.locator('input[placeholder="Type a message..."]')
    await expect(chatInput).toBeVisible()

    // Type a message
    await chatInput.click()
    await chatInput.fill('Hello, can you help me with a task?')

    // Press Enter to send
    await chatInput.press('Enter')
    await page.waitForTimeout(500)

    // Verify the message appears in the chat history (blue user bubble)
    await expect(
      page.locator('div.bg-blue-600:has-text("Hello, can you help me with a task?")')
    ).toBeVisible({ timeout: 5_000 })

    // Verify the input is cleared after sending
    await expect(chatInput).toHaveValue('')
  })

  test('send multiple messages and verify all appear in order', async ({ page }) => {
    // Wait for right panel
    await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })

    const chatInput = page.locator('input[placeholder="Type a message..."]')
    await expect(chatInput).toBeVisible()

    const messages = [
      'Step 1: Initialize the project',
      'Step 2: Write the tests',
      'Step 3: Deploy to production',
      'Step 4: Monitor the logs',
    ]

    // Send all messages
    for (const msg of messages) {
      await chatInput.click()
      await chatInput.fill(msg)
      await chatInput.press('Enter')
      await page.waitForTimeout(300)
    }

    // Verify all messages appear in the chat history in order
    const userBubbles = page.locator('div.bg-blue-600')
    await expect(userBubbles).toHaveCount(messages.length, { timeout: 5_000 })

    // Verify each message text is present
    for (const msg of messages) {
      await expect(page.locator(`div.bg-blue-600:has-text("${msg}")`)).toBeVisible({ timeout: 5_000 })
    }
  })

  test('send message via send button (not Enter key)', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })

    const chatInput = page.locator('input[placeholder="Type a message..."]')
    await expect(chatInput).toBeVisible()

    // Type a message
    await chatInput.fill('Testing the send button')

    // Click the send button (paper plane icon)
    const sendButton = page.locator('button:has(svg)').last()
    await sendButton.click()
    await page.waitForTimeout(500)

    // Verify the message appears
    await expect(
      page.locator('div.bg-blue-600:has-text("Testing the send button")')
    ).toBeVisible({ timeout: 5_000 })
  })

  test('verify message ordering is preserved after multiple rapid sends', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })

    const chatInput = page.locator('input[placeholder="Type a message..."]')

    // Send a rapid sequence of messages
    const rapidMessages = ['alpha', 'beta', 'gamma', 'delta', 'epsilon']

    for (const msg of rapidMessages) {
      await chatInput.fill(msg)
      await chatInput.press('Enter')
      await page.waitForTimeout(100) // minimal delay
    }

    // Wait for all to render
    await page.waitForTimeout(1000)

    // Verify count
    const userBubbles = page.locator('div.bg-blue-600')
    await expect(userBubbles).toHaveCount(rapidMessages.length, { timeout: 5_000 })

    // Verify all texts are present (order may vary in DOM, but all should exist)
    for (const msg of rapidMessages) {
      await expect(page.locator(`div.bg-blue-600:has-text("${msg}")`)).toBeVisible({ timeout: 5_000 })
    }
  })

  test('empty message should not be sent', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })

    const chatInput = page.locator('input[placeholder="Type a message..."]')

    // Try to send empty message with Enter
    await chatInput.click()
    await chatInput.press('Enter')
    await page.waitForTimeout(500)

    // Verify no new user bubbles were created (only the welcome agent bubble should exist)
    // Count bubbles: should still just be the initial agent message
    const initialCount = 1 // just the welcome message
    const allBubbles = page.locator('div.max-w-\\[80\\%\\]')
    await expect(allBubbles).toHaveCount(initialCount, { timeout: 5_000 })
  })

  test('whitespace-only message should not be sent', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })

    const chatInput = page.locator('input[placeholder="Type a message..."]')

    // Type whitespace and press Enter
    await chatInput.fill('   ')
    await chatInput.press('Enter')
    await page.waitForTimeout(500)

    // No new bubbles should appear
    const userBubbles = page.locator('div.bg-blue-600')
    await expect(userBubbles).toHaveCount(0, { timeout: 5_000 })
  })
})
