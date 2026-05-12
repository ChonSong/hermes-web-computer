# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/01-layout.spec.ts >> three-column layout renders
- Location: tests/01-layout.spec.ts:3:1

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
  3  | test('three-column layout renders', async ({ page }) => {
> 4  |   await page.goto('/')
     |              ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  5  |   // Wait for connected state (no "Disconnected" visible)
  6  |   await expect(page.getByText('Disconnected')).not.toBeVisible({ timeout: 10000 })
  7  | 
  8  |   // Left panel visible with tabs
  9  |   await expect(page.getByText('📁 Files')).toBeVisible()
  10 |   await expect(page.getByText('🚀 Apps')).toBeVisible()
  11 | 
  12 |   // Middle panel renders a terminal tile (border-blue-500)
  13 |   await expect(page.locator('div.border-blue-500').first()).toBeVisible({ timeout: 10000 })
  14 | 
  15 |   // Right panel with agent chat
  16 |   await expect(page.getByRole('heading', { name: 'Agent' })).toBeVisible()
  17 |   await expect(page.getByRole('textbox', { name: 'Type a message...' })).toBeVisible()
  18 | 
  19 |   // Screenshot for vision analysis
  20 |   await page.screenshot({ path: 'e2e/test-results/layout-default.png', fullPage: true })
  21 | })
  22 | 
```