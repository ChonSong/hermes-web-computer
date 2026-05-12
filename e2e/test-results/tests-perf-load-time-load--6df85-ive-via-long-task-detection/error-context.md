# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/perf/load-time.spec.ts >> load-time-perf >> measures Time to Interactive via long task detection
- Location: tests/perf/load-time.spec.ts:92:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1   | /**
  2   |  * Perf Test: Load Time Metrics
  3   |  *
  4   |  * Measures key performance metrics:
  5   |  * - TTFB (Time to First Byte)
  6   |  * - DCL (DOMContentLoaded)
  7   |  * - TTI (Time to Interactive — approximated via Performance API)
  8   |  * - Bundle size (from response headers / resource timing)
  9   |  */
  10  | import { test, expect } from '@playwright/test'
  11  | 
  12  | test.describe('load-time-perf', () => {
  13  |   test('measures TTFB, DCL, and reports bundle size', async ({ page }) => {
  14  |     // Navigate to the page
  15  |     const navigationStart = Date.now()
  16  |     await page.goto('/')
  17  | 
  18  |     // Wait for the app to mount
  19  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  20  |     const totalLoadTime = Date.now() - navigationStart
  21  | 
  22  |     // Collect Performance API metrics from the page
  23  |     const perfMetrics = await page.evaluate(() => {
  24  |       const perf = performance
  25  |       const nav = perf.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
  26  |       const resources = perf.getEntriesByType('resource') as PerformanceResourceTiming[]
  27  | 
  28  |       // Find the main JS bundle
  29  |       const jsBundles = resources.filter(r => r.name.endsWith('.js') && r.responseEnd > 0)
  30  |       const totalJsSize = jsBundles.reduce((sum, r) => sum + (r.transferSize || r.decodedBodySize || 0), 0)
  31  | 
  32  |       // Find the main HTML document
  33  |       const htmlResource = resources.find(r => r.name.endsWith('/') || r.name.endsWith('.html'))
  34  | 
  35  |       return {
  36  |         // TTFB: Time from navigation start to first byte of response
  37  |         ttfb: nav ? nav.responseStart - nav.startTime : 0,
  38  | 
  39  |         // DCL: DOM Content Loaded
  40  |         dcl: nav ? nav.domContentLoadedEventEnd - nav.startTime : 0,
  41  | 
  42  |         // DOM Complete (approximation of TTI)
  43  |         domComplete: nav ? nav.domComplete - nav.startTime : 0,
  44  | 
  45  |         // Load event
  46  |         loadEvent: nav ? nav.loadEventEnd - nav.startTime : 0,
  47  | 
  48  |         // JS bundle size
  49  |         jsBundleCount: jsBundles.length,
  50  |         totalJsBytes: totalJsSize,
  51  |         totalJsKB: Math.round(totalJsSize / 1024),
  52  | 
  53  |         // HTML size
  54  |         htmlBytes: htmlResource?.decodedBodySize || 0,
  55  |         htmlKB: Math.round((htmlResource?.decodedBodySize || 0) / 1024),
  56  | 
  57  |         // Total transfer size
  58  |         totalTransferBytes: resources.reduce((sum, r) => sum + (r.transferSize || 0), 0),
  59  |         totalTransferKB: Math.round(resources.reduce((sum, r) => sum + (r.transferSize || 0), 0) / 1024),
  60  | 
  61  |         // Resource count
  62  |         resourceCount: resources.length,
  63  |       }
  64  |     })
  65  | 
  66  |     // Log performance report
  67  |     console.log('=== Performance Report ===')
  68  |     console.log(`TTFB: ${perfMetrics.ttfb.toFixed(0)}ms`)
  69  |     console.log(`DCL: ${perfMetrics.dcl.toFixed(0)}ms`)
  70  |     console.log(`DOM Complete: ${perfMetrics.domComplete.toFixed(0)}ms`)
  71  |     console.log(`Load Event: ${perfMetrics.loadEvent.toFixed(0)}ms`)
  72  |     console.log(`JS Bundles: ${perfMetrics.jsBundleCount} (${perfMetrics.totalJsKB} KB)`)
  73  |     console.log(`HTML Size: ${perfMetrics.htmlKB} KB`)
  74  |     console.log(`Total Transfer: ${perfMetrics.totalTransferKB} KB`)
  75  |     console.log(`Resources Loaded: ${perfMetrics.resourceCount}`)
  76  |     console.log(`Total Load Time (wall clock): ${totalLoadTime}ms`)
  77  | 
  78  |     // Assertions — set reasonable thresholds for a dev environment
  79  |     // TTFB should be under 2 seconds (dev server)
  80  |     expect(perfMetrics.ttfb).toBeLessThan(2000)
  81  | 
  82  |     // DCL should be under 5 seconds
  83  |     expect(perfMetrics.dcl).toBeLessThan(5000)
  84  | 
  85  |     // DOM Complete should be under 10 seconds
  86  |     expect(perfMetrics.domComplete).toBeLessThan(10000)
  87  | 
  88  |     // JS bundle should be under 2MB uncompressed
  89  |     expect(perfMetrics.totalJsKB).toBeLessThan(2048)
  90  |   })
  91  | 
  92  |   test('measures Time to Interactive via long task detection', async ({ page }) => {
> 93  |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  94  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  95  | 
  96  |     // Wait for the page to settle
  97  |     await page.waitForTimeout(2000)
  98  | 
  99  |     // Check for long tasks (>50ms) that would delay interactivity
  100 |     const longTasks = await page.evaluate(() => {
  101 |       // PerformanceLongTaskTiming API may not be available in all browsers
  102 |       const entries = performance.getEntriesByType('longtask') as PerformanceEntry[]
  103 |       return entries.map(e => ({
  104 |         name: e.name,
  105 |         startTime: e.startTime,
  106 |         duration: e.duration,
  107 |       }))
  108 |     })
  109 | 
  110 |     console.log('Long tasks detected:', longTasks.length)
  111 |     if (longTasks.length > 0) {
  112 |       console.log('Long task details:', JSON.stringify(longTasks, null, 2))
  113 |     }
  114 | 
  115 |     // Verify page is interactive by checking if we can interact with elements
  116 |     const isInteractive = await page.evaluate(() => {
  117 |       const app = document.getElementById('app')
  118 |       return app && app.childElementCount > 0
  119 |     })
  120 |     expect(isInteractive).toBe(true)
  121 |   })
  122 | 
  123 |   test('reports initial paint and contentful paint timings', async ({ page }) => {
  124 |     await page.goto('/')
  125 |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  126 | 
  127 |     const paintTimings = await page.evaluate(() => {
  128 |       const entries = performance.getEntriesByType('paint') as PerformanceEntry[]
  129 |       const result: Record<string, number> = {}
  130 |       for (const entry of entries) {
  131 |         result[entry.name] = entry.startTime
  132 |       }
  133 |       return result
  134 |     })
  135 | 
  136 |     console.log('Paint timings:', JSON.stringify(paintTimings, null, 2))
  137 | 
  138 |     // First Paint should occur
  139 |     expect(paintTimings['first-paint'] ?? paintTimings['first-contentful-paint']).toBeDefined()
  140 |   })
  141 | })
  142 | 
```