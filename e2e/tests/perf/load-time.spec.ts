/**
 * Perf Test: Load Time Metrics
 *
 * Measures key performance metrics:
 * - TTFB (Time to First Byte)
 * - DCL (DOMContentLoaded)
 * - TTI (Time to Interactive — approximated via Performance API)
 * - Bundle size (from response headers / resource timing)
 */
import { test, expect } from '@playwright/test'

test.describe('load-time-perf', () => {
  test('measures TTFB, DCL, and reports bundle size', async ({ page }) => {
    // Navigate to the page
    const navigationStart = Date.now()
    await page.goto('/')

    // Wait for the app to mount
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
    const totalLoadTime = Date.now() - navigationStart

    // Collect Performance API metrics from the page
    const perfMetrics = await page.evaluate(() => {
      const perf = performance
      const nav = perf.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
      const resources = perf.getEntriesByType('resource') as PerformanceResourceTiming[]

      // Find the main JS bundle
      const jsBundles = resources.filter(r => r.name.endsWith('.js') && r.responseEnd > 0)
      const totalJsSize = jsBundles.reduce((sum, r) => sum + (r.transferSize || r.decodedBodySize || 0), 0)

      // Find the main HTML document
      const htmlResource = resources.find(r => r.name.endsWith('/') || r.name.endsWith('.html'))

      return {
        // TTFB: Time from navigation start to first byte of response
        ttfb: nav ? nav.responseStart - nav.startTime : 0,

        // DCL: DOM Content Loaded
        dcl: nav ? nav.domContentLoadedEventEnd - nav.startTime : 0,

        // DOM Complete (approximation of TTI)
        domComplete: nav ? nav.domComplete - nav.startTime : 0,

        // Load event
        loadEvent: nav ? nav.loadEventEnd - nav.startTime : 0,

        // JS bundle size
        jsBundleCount: jsBundles.length,
        totalJsBytes: totalJsSize,
        totalJsKB: Math.round(totalJsSize / 1024),

        // HTML size
        htmlBytes: htmlResource?.decodedBodySize || 0,
        htmlKB: Math.round((htmlResource?.decodedBodySize || 0) / 1024),

        // Total transfer size
        totalTransferBytes: resources.reduce((sum, r) => sum + (r.transferSize || 0), 0),
        totalTransferKB: Math.round(resources.reduce((sum, r) => sum + (r.transferSize || 0), 0) / 1024),

        // Resource count
        resourceCount: resources.length,
      }
    })

    // Log performance report
    console.log('=== Performance Report ===')
    console.log(`TTFB: ${perfMetrics.ttfb.toFixed(0)}ms`)
    console.log(`DCL: ${perfMetrics.dcl.toFixed(0)}ms`)
    console.log(`DOM Complete: ${perfMetrics.domComplete.toFixed(0)}ms`)
    console.log(`Load Event: ${perfMetrics.loadEvent.toFixed(0)}ms`)
    console.log(`JS Bundles: ${perfMetrics.jsBundleCount} (${perfMetrics.totalJsKB} KB)`)
    console.log(`HTML Size: ${perfMetrics.htmlKB} KB`)
    console.log(`Total Transfer: ${perfMetrics.totalTransferKB} KB`)
    console.log(`Resources Loaded: ${perfMetrics.resourceCount}`)
    console.log(`Total Load Time (wall clock): ${totalLoadTime}ms`)

    // Assertions — set reasonable thresholds for a dev environment
    // TTFB should be under 2 seconds (dev server)
    expect(perfMetrics.ttfb).toBeLessThan(2000)

    // DCL should be under 5 seconds
    expect(perfMetrics.dcl).toBeLessThan(5000)

    // DOM Complete should be under 10 seconds
    expect(perfMetrics.domComplete).toBeLessThan(10000)

    // JS bundle should be under 2MB uncompressed
    expect(perfMetrics.totalJsKB).toBeLessThan(2048)
  })

  test('measures Time to Interactive via long task detection', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Wait for the page to settle
    await page.waitForTimeout(2000)

    // Check for long tasks (>50ms) that would delay interactivity
    const longTasks = await page.evaluate(() => {
      // PerformanceLongTaskTiming API may not be available in all browsers
      const entries = performance.getEntriesByType('longtask') as PerformanceEntry[]
      return entries.map(e => ({
        name: e.name,
        startTime: e.startTime,
        duration: e.duration,
      }))
    })

    console.log('Long tasks detected:', longTasks.length)
    if (longTasks.length > 0) {
      console.log('Long task details:', JSON.stringify(longTasks, null, 2))
    }

    // Verify page is interactive by checking if we can interact with elements
    const isInteractive = await page.evaluate(() => {
      const app = document.getElementById('app')
      return app && app.childElementCount > 0
    })
    expect(isInteractive).toBe(true)
  })

  test('reports initial paint and contentful paint timings', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    const paintTimings = await page.evaluate(() => {
      const entries = performance.getEntriesByType('paint') as PerformanceEntry[]
      const result: Record<string, number> = {}
      for (const entry of entries) {
        result[entry.name] = entry.startTime
      }
      return result
    })

    console.log('Paint timings:', JSON.stringify(paintTimings, null, 2))

    // First Paint should occur
    expect(paintTimings['first-paint'] ?? paintTimings['first-contentful-paint']).toBeDefined()
  })
})
