# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/workflows/chat-context.spec.ts >> Chat Context Workflow >> verify message ordering is preserved after multiple rapid sends
- Location: tests/workflows/chat-context.spec.ts:91:3

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
  3   | test.describe('Chat Context Workflow', () => {
  4   |   test.beforeEach(async ({ page }) => {
> 5   |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  6   |     // Wait for WebSocket connection
  7   |     await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10_000 })
  8   |     await expect(page.getByText('Connecting...')).not.toBeVisible({ timeout: 10_000 })
  9   |   })
  10  | 
  11  |   test('type message in right panel, press Enter, verify it appears in history', async ({ page }) => {
  12  |     // The right panel should already be visible with the Agent header and welcome message
  13  |     await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })
  14  | 
  15  |     // Verify the welcome message is present
  16  |     await expect(page.getByText("Hello! I'm your agent. How can I help you today?")).toBeVisible({ timeout: 5_000 })
  17  | 
  18  |     // Find the chat input in the right panel
  19  |     const chatInput = page.locator('input[placeholder="Type a message..."]')
  20  |     await expect(chatInput).toBeVisible()
  21  | 
  22  |     // Type a message
  23  |     await chatInput.click()
  24  |     await chatInput.fill('Hello, can you help me with a task?')
  25  | 
  26  |     // Press Enter to send
  27  |     await chatInput.press('Enter')
  28  |     await page.waitForTimeout(500)
  29  | 
  30  |     // Verify the message appears in the chat history (blue user bubble)
  31  |     await expect(
  32  |       page.locator('div.bg-blue-600:has-text("Hello, can you help me with a task?")')
  33  |     ).toBeVisible({ timeout: 5_000 })
  34  | 
  35  |     // Verify the input is cleared after sending
  36  |     await expect(chatInput).toHaveValue('')
  37  |   })
  38  | 
  39  |   test('send multiple messages and verify all appear in order', async ({ page }) => {
  40  |     // Wait for right panel
  41  |     await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })
  42  | 
  43  |     const chatInput = page.locator('input[placeholder="Type a message..."]')
  44  |     await expect(chatInput).toBeVisible()
  45  | 
  46  |     const messages = [
  47  |       'Step 1: Initialize the project',
  48  |       'Step 2: Write the tests',
  49  |       'Step 3: Deploy to production',
  50  |       'Step 4: Monitor the logs',
  51  |     ]
  52  | 
  53  |     // Send all messages
  54  |     for (const msg of messages) {
  55  |       await chatInput.click()
  56  |       await chatInput.fill(msg)
  57  |       await chatInput.press('Enter')
  58  |       await page.waitForTimeout(300)
  59  |     }
  60  | 
  61  |     // Verify all messages appear in the chat history in order
  62  |     const userBubbles = page.locator('div.bg-blue-600')
  63  |     await expect(userBubbles).toHaveCount(messages.length, { timeout: 5_000 })
  64  | 
  65  |     // Verify each message text is present
  66  |     for (const msg of messages) {
  67  |       await expect(page.locator(`div.bg-blue-600:has-text("${msg}")`)).toBeVisible({ timeout: 5_000 })
  68  |     }
  69  |   })
  70  | 
  71  |   test('send message via send button (not Enter key)', async ({ page }) => {
  72  |     await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })
  73  | 
  74  |     const chatInput = page.locator('input[placeholder="Type a message..."]')
  75  |     await expect(chatInput).toBeVisible()
  76  | 
  77  |     // Type a message
  78  |     await chatInput.fill('Testing the send button')
  79  | 
  80  |     // Click the send button (paper plane icon)
  81  |     const sendButton = page.locator('button:has(svg)').last()
  82  |     await sendButton.click()
  83  |     await page.waitForTimeout(500)
  84  | 
  85  |     // Verify the message appears
  86  |     await expect(
  87  |       page.locator('div.bg-blue-600:has-text("Testing the send button")')
  88  |     ).toBeVisible({ timeout: 5_000 })
  89  |   })
  90  | 
  91  |   test('verify message ordering is preserved after multiple rapid sends', async ({ page }) => {
  92  |     await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible({ timeout: 10_000 })
  93  | 
  94  |     const chatInput = page.locator('input[placeholder="Type a message..."]')
  95  | 
  96  |     // Send a rapid sequence of messages
  97  |     const rapidMessages = ['alpha', 'beta', 'gamma', 'delta', 'epsilon']
  98  | 
  99  |     for (const msg of rapidMessages) {
  100 |       await chatInput.fill(msg)
  101 |       await chatInput.press('Enter')
  102 |       await page.waitForTimeout(100) // minimal delay
  103 |     }
  104 | 
  105 |     // Wait for all to render
```