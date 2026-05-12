# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/a11y/keyboard.spec.ts >> keyboard-navigation >> keyboard shortcuts toggle UI panels
- Location: tests/a11y/keyboard.spec.ts:63:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1   | /**
  2   |  * A11y Test: Keyboard Navigation
  3   |  *
  4   |  * Verifies that the UI is navigable via keyboard, with visible focus rings
  5   |  * on all interactive elements.
  6   |  */
  7   | import { test, expect } from '@playwright/test'
  8   | 
  9   | test.describe('keyboard-navigation', () => {
  10  |   test('Tab through UI and verify focus rings are visible', async ({ page }) => {
  11  |     await page.goto('/')
  12  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  13  | 
  14  |     // Focus should start on the first focusable element or the page
  15  |     // Press Tab multiple times and verify focus moves
  16  |     const focusableSelectors = [
  17  |       'button',
  18  |       'input',
  19  |       'a[href]',
  20  |       '[tabindex]:not([tabindex="-1"])',
  21  |       'textarea',
  22  |       'select',
  23  |     ]
  24  | 
  25  |     // Tab through elements and verify each gets focus
  26  |     for (let i = 0; i < 5; i++) {
  27  |       await page.keyboard.press('Tab')
  28  |       await page.waitForTimeout(100)
  29  | 
  30  |       // Check that something has focus (document.activeElement exists)
  31  |       const focusedTag = await page.evaluate(() => document.activeElement?.tagName ?? null)
  32  |       expect(focusedTag).not.toBeNull()
  33  |     }
  34  |   })
  35  | 
  36  |   test('focus rings are visible on focused elements', async ({ page }) => {
  37  |     await page.goto('/')
  38  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  39  | 
  40  |     // Tab to a focusable element
  41  |     await page.keyboard.press('Tab')
  42  |     await page.waitForTimeout(100)
  43  | 
  44  |     // Check that the focused element has a visible focus ring
  45  |     const hasFocusRing = await page.evaluate(() => {
  46  |       const el = document.activeElement
  47  |       if (!el) return false
  48  |       const style = window.getComputedStyle(el)
  49  |       // Check for outline or box-shadow on focus
  50  |       const outline = style.outlineStyle
  51  |       const boxShadow = style.boxShadow
  52  |       // Either outline or box-shadow indicates a focus ring
  53  |       return outline !== 'none' || boxShadow !== 'none' ||
  54  |         outlineWidth !== '0px'
  55  |     })
  56  | 
  57  |     // If the app has custom focus management, verify at least some visual indicator
  58  |     const focusedClasses = await page.evaluate(() => document.activeElement?.className ?? '')
  59  |     // Even if no explicit focus ring, the element should be focused in DOM
  60  |     expect(await page.evaluate(() => document.activeElement !== document.body)).toBe(true)
  61  |   })
  62  | 
  63  |   test('keyboard shortcuts toggle UI panels', async ({ page }) => {
> 64  |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  65  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  66  | 
  67  |     // Ctrl+K should toggle command palette
  68  |     await page.keyboard.press('Control+K')
  69  |     await page.waitForTimeout(500)
  70  |     // Command palette should be visible (check for common patterns)
  71  |     const hasPalette = await page.evaluate(() => {
  72  |       return document.querySelector('[class*="palette"]') !== null ||
  73  |              document.querySelector('[class*="Palette"]') !== null ||
  74  |              document.querySelector('.w-96.bg-gray-900') !== null
  75  |     })
  76  | 
  77  |     // Escape should close any overlay
  78  |     await page.keyboard.press('Escape')
  79  |     await page.waitForTimeout(300)
  80  | 
  81  |     // Ctrl+? should toggle keymap overlay
  82  |     await page.keyboard.press('Control+?')
  83  |     await page.waitForTimeout(500)
  84  |     const hasKeymap = await page.evaluate(() => {
  85  |       return document.querySelector('[class*="keymap"]') !== null ||
  86  |              document.querySelector('[class*="Keymap"]') !== null
  87  |     })
  88  | 
  89  |     // Escape again to close
  90  |     await page.keyboard.press('Escape')
  91  |     await page.waitForTimeout(300)
  92  |   })
  93  | 
  94  |   test('Escape key dismisses overlays', async ({ page }) => {
  95  |     await page.goto('/')
  96  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  97  | 
  98  |     // Open command palette
  99  |     await page.keyboard.press('Control+K')
  100 |     await page.waitForTimeout(500)
  101 | 
  102 |     // Press Escape to dismiss
  103 |     await page.keyboard.press('Escape')
  104 |     await page.waitForTimeout(300)
  105 | 
  106 |     // Verify the palette is closed by checking dialog state
  107 |     const dialogVisible = await page.locator('.w-96.bg-gray-900').isVisible().catch(() => false)
  108 |     // After escape, dialog should be hidden or removed
  109 |     expect(dialogVisible).toBe(false)
  110 |   })
  111 | })
  112 | 
```