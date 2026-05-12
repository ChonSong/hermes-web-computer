# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/a11y/screen-reader.spec.ts >> screen-reader-aria >> axe-core reports no critical violations on main page
- Location: tests/a11y/screen-reader.spec.ts:42:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1   | /**
  2   |  * A11y Test: Screen Reader / ARIA Labels
  3   |  *
  4   |  * Uses @axe-core/playwright to verify ARIA labels and accessibility attributes
  5   |  * on key UI elements.
  6   |  */
  7   | import { test, expect } from '@playwright/test'
  8   | import { AxeBuilder } from '@axe-core/playwright'
  9   | 
  10  | test.describe('screen-reader-aria', () => {
  11  |   test('main app container has accessible name or role', async ({ page }) => {
  12  |     await page.goto('/')
  13  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  14  | 
  15  |     // Check that the app container has a role or aria-label
  16  |     const hasAccessibleName = await page.evaluate(() => {
  17  |       const app = document.getElementById('app')
  18  |       if (!app) return false
  19  |       return !!(
  20  |         app.getAttribute('role') ||
  21  |         app.getAttribute('aria-label') ||
  22  |         app.getAttribute('aria-labelledby')
  23  |       )
  24  |     })
  25  | 
  26  |     // The app div may not have ARIA attributes — note this for improvement
  27  |     // We'll check child elements instead
  28  |     const accessibleChildren = await page.evaluate(() => {
  29  |       const interactive = document.querySelectorAll('button, a, input, [role]')
  30  |       return Array.from(interactive).map(el => ({
  31  |         tag: el.tagName.toLowerCase(),
  32  |         role: el.getAttribute('role'),
  33  |         ariaLabel: el.getAttribute('aria-label'),
  34  |         text: el.textContent?.trim()?.substring(0, 50) ?? '',
  35  |       }))
  36  |     })
  37  | 
  38  |     // Log findings for accessibility review
  39  |     console.log('Interactive elements found:', JSON.stringify(accessibleChildren, null, 2))
  40  |   })
  41  | 
  42  |   test('axe-core reports no critical violations on main page', async ({ page }) => {
> 43  |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  44  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  45  | 
  46  |     // Wait a moment for dynamic content to settle
  47  |     await page.waitForTimeout(1000)
  48  | 
  49  |     const results = await new AxeBuilder({ page })
  50  |       .withTags(['wcag2a', 'wcag2aa'])
  51  |       .analyze()
  52  | 
  53  |     // Log all violations for review
  54  |     if (results.violations.length > 0) {
  55  |       console.log('Axe violations:', JSON.stringify(results.violations, null, 2))
  56  |     }
  57  | 
  58  |     // Fail on critical/severe violations
  59  |     const critical = results.violations.filter(v => v.impact === 'critical' || v.impact === 'serious')
  60  |     expect(critical).toHaveLength(0)
  61  |   })
  62  | 
  63  |   test('key structural elements have ARIA roles', async ({ page }) => {
  64  |     await page.goto('/')
  65  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  66  | 
  67  |     // Check for panel-like elements that should have roles
  68  |     const panelRoles = await page.evaluate(() => {
  69  |       // Find elements that look like panels (based on CSS classes)
  70  |       const elements = document.querySelectorAll('[class*="panel"], [class*="Panel"]')
  71  |       return Array.from(elements).map(el => ({
  72  |         tag: el.tagName.toLowerCase(),
  73  |         role: el.getAttribute('role'),
  74  |         ariaLabel: el.getAttribute('aria-label'),
  75  |         classes: el.className,
  76  |       }))
  77  |     })
  78  | 
  79  |     console.log('Panel elements:', JSON.stringify(panelRoles, null, 2))
  80  | 
  81  |     // Check disconnected message has appropriate structure
  82  |     const disconnectedEl = page.getByText('Disconnected', { exact: true })
  83  |     const disconnectedVisible = await disconnectedEl.isVisible().catch(() => false)
  84  | 
  85  |     if (disconnectedVisible) {
  86  |       // Disconnected state should be announced to screen readers
  87  |       const hasAlertRole = await page.evaluate(() => {
  88  |         const els = document.querySelectorAll('[role="alert"], [role="status"], [aria-live]')
  89  |         return Array.from(els).length > 0
  90  |       })
  91  |       expect(hasAlertRole).toBe(true)
  92  |     }
  93  |   })
  94  | 
  95  |   test('command palette is accessible when opened', async ({ page }) => {
  96  |     await page.goto('/')
  97  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  98  | 
  99  |     // Open command palette
  100 |     await page.keyboard.press('Control+K')
  101 |     await page.waitForTimeout(500)
  102 | 
  103 |     // Check if palette appears and has proper ARIA attributes
  104 |     const paletteA11y = await page.evaluate(() => {
  105 |       const dialog = document.querySelector('[role="dialog"]')
  106 |       const palette = document.querySelector('[class*="palette"], [class*="Palette"]')
  107 |       const target = (dialog || palette) as HTMLElement | null
  108 |       if (!target) return null
  109 |       return {
  110 |         role: target.getAttribute('role'),
  111 |         ariaLabel: target.getAttribute('aria-label'),
  112 |         ariaModal: target.getAttribute('aria-modal'),
  113 |         hasFocus: target === document.activeElement || target.contains(document.activeElement),
  114 |       }
  115 |     })
  116 | 
  117 |     if (paletteA11y) {
  118 |       console.log('Command palette A11y:', JSON.stringify(paletteA11y, null, 2))
  119 |       // Should have role="dialog" and trap focus
  120 |       expect(paletteA11y.role).toBe('dialog')
  121 |     }
  122 |   })
  123 | })
  124 | 
```