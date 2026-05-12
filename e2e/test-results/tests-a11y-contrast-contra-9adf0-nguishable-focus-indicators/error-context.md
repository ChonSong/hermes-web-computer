# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/a11y/contrast.spec.ts >> contrast-wcag-aa >> interactive elements have distinguishable focus indicators
- Location: tests/a11y/contrast.spec.ts:141:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  42  | 
  43  |     const colors = await page.evaluate(() => {
  44  |       const app = document.querySelector('.bg-gray-950') || document.body
  45  |       const appStyle = window.getComputedStyle(app)
  46  |       return {
  47  |         bg: appStyle.backgroundColor,
  48  |       }
  49  |     })
  50  | 
  51  |     // Parse the background color
  52  |     const bgColorMatch = colors.bg.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/)
  53  |     expect(bgColorMatch).not.toBeNull()
  54  | 
  55  |     console.log('Background color:', colors.bg)
  56  |   })
  57  | 
  58  |   test('all visible text elements meet minimum contrast', async ({ page }) => {
  59  |     await page.goto('/')
  60  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  61  | 
  62  |     // Wait for content to settle
  63  |     await page.waitForTimeout(500)
  64  | 
  65  |     // Sample key text elements and their computed colors
  66  |     const textContrasts = await page.evaluate(
  67  |       (normalThreshold: number, largeThreshold: number) => {
  68  |         function parseColor(colorStr: string): [number, number, number] | null {
  69  |           const match = colorStr.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/)
  70  |           if (!match) return null
  71  |           return [parseInt(match[1]), parseInt(match[2]), parseInt(match[3])]
  72  |         }
  73  | 
  74  |         function luminance(r: number, g: number, b: number): number {
  75  |           const [rs, gs, bs] = [r, g, b].map(c => {
  76  |             const srgb = c / 255
  77  |             return srgb <= 0.03928 ? srgb / 12.92 : Math.pow((srgb + 0.055) / 1.055, 2.4)
  78  |           })
  79  |           return 0.2126 * rs + 0.7152 * gs + 0.0722 * bs
  80  |         }
  81  | 
  82  |         function ratio(l1: number, l2: number): number {
  83  |           return (Math.max(l1, l2) + 0.05) / (Math.min(l1, l2) + 0.05)
  84  |         }
  85  | 
  86  |         // Get all text-containing elements
  87  |         const elements = document.querySelectorAll('p, span, h1, h2, h3, button, a, label, div')
  88  |         const results: { text: string; bg: string; fg: string; pass: boolean; element: string }[] = []
  89  | 
  90  |         for (const el of Array.from(elements)) {
  91  |           // Skip elements with no direct text
  92  |           if (el.children.length > 0 && el.textContent?.trim() === '') continue
  93  |           if (!el.textContent?.trim()) continue
  94  | 
  95  |           // Skip if text is too long (we just need a sample)
  96  |           if (el.textContent.trim().length > 100) continue
  97  | 
  98  |           const style = window.getComputedStyle(el)
  99  |           const fg = parseColor(style.color)
  100 |           const bg = parseColor(style.backgroundColor)
  101 | 
  102 |           // Skip transparent or auto backgrounds
  103 |           if (!fg || !bg) continue
  104 | 
  105 |           const fgLum = luminance(...fg)
  106 |           const bgLum = luminance(...bg)
  107 |           const contrastR = ratio(fgLum, bgLum)
  108 | 
  109 |           const fontSize = parseFloat(style.fontSize)
  110 |           const isBold = style.fontWeight && parseInt(style.fontWeight) >= 700
  111 |           const isLargeText = fontSize >= 18 || (fontSize >= 14 && isBold)
  112 |           const threshold = isLargeText ? largeThreshold : normalThreshold
  113 | 
  114 |           results.push({
  115 |             text: el.textContent.trim().substring(0, 50),
  116 |             bg: style.backgroundColor,
  117 |             fg: style.color,
  118 |             pass: contrastR >= threshold,
  119 |             element: el.tagName.toLowerCase() + (el.className ? `.${el.className.split(' ').slice(0, 3).join('.')}` : ''),
  120 |           })
  121 |         }
  122 | 
  123 |         return results
  124 |       },
  125 |       NORMAL_TEXT_RATIO,
  126 |       LARGE_TEXT_RATIO
  127 |     )
  128 | 
  129 |     // Log results for review
  130 |     const failures = textContrasts.filter(r => !r.pass)
  131 |     if (failures.length > 0) {
  132 |       console.log('Contrast failures:', JSON.stringify(failures, null, 2))
  133 |     }
  134 | 
  135 |     // The app uses text-gray-100 on bg-gray-950 — this should pass
  136 |     // text-gray-100 (#f3f4f6) on bg-gray-950 (#030712) gives ~16.7:1 ratio
  137 |     const passing = textContrasts.filter(r => r.pass)
  138 |     console.log(`Contrast check: ${passing.length}/${textContrasts.length} elements pass WCAG AA`)
  139 |   })
  140 | 
  141 |   test('interactive elements have distinguishable focus indicators', async ({ page }) => {
> 142 |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  143 |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  144 | 
  145 |     // Tab to focus an element and check its focus style
  146 |     await page.keyboard.press('Tab')
  147 |     await page.waitForTimeout(200)
  148 | 
  149 |     const focusStyle = await page.evaluate(() => {
  150 |       const el = document.activeElement
  151 |       if (!el || el === document.body) return null
  152 |       const style = window.getComputedStyle(el)
  153 |       return {
  154 |         outline: style.outline,
  155 |         outlineWidth: style.outlineWidth,
  156 |         outlineStyle: style.outlineStyle,
  157 |         outlineColor: style.outlineColor,
  158 |         boxShadow: style.boxShadow,
  159 |         borderColor: style.borderColor,
  160 |       }
  161 |     })
  162 | 
  163 |     if (focusStyle) {
  164 |       console.log('Focus style:', JSON.stringify(focusStyle, null, 2))
  165 |     }
  166 |   })
  167 | })
  168 | 
```