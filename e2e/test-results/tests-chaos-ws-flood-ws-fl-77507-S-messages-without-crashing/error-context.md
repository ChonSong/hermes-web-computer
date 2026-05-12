# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: tests/chaos/ws-flood.spec.ts >> ws-flood >> handles rapid WS messages without crashing
- Location: tests/chaos/ws-flood.spec.ts:10:3

# Error details

```
Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
Call log:
  - navigating to "/", waiting until "load"

```

# Test source

```ts
  1   | /**
  2   |  * Chaos Test: WebSocket Message Flood
  3   |  *
  4   |  * Opens a WebSocket connection to the server and sends a high volume of messages
  5   |  * to verify the frontend doesn't crash under message pressure.
  6   |  */
  7   | import { test, expect } from '@playwright/test'
  8   | 
  9   | test.describe('ws-flood', () => {
  10  |   test('handles rapid WS messages without crashing', async ({ page }) => {
> 11  |     await page.goto('/')
      |                ^ Error: page.goto: Protocol error (Page.navigate): Cannot navigate to invalid URL
  12  | 
  13  |     // Wait for the app container to be present
  14  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  15  | 
  16  |     // Open a WebSocket from page context and flood it
  17  |     await page.evaluate(async () => {
  18  |       return new Promise<number>((resolve) => {
  19  |         const socket = new WebSocket('ws://localhost:3005/ws')
  20  |         let received = 0
  21  |         const floodCount = 500
  22  | 
  23  |         socket.onopen = () => {
  24  |           // Send many messages rapidly
  25  |           for (let i = 0; i < floodCount; i++) {
  26  |             socket.send(JSON.stringify({
  27  |               protocol: 'ui',
  28  |               method: 'fs.list',
  29  |               params: { path: '/' },
  30  |               id: `flood_${i}`,
  31  |               ts: Date.now(),
  32  |             }))
  33  |           }
  34  |         }
  35  | 
  36  |         socket.onmessage = () => {
  37  |           received++
  38  |         }
  39  | 
  40  |         socket.onerror = () => {
  41  |           // Server may reject unknown messages — that's fine
  42  |           resolve(received)
  43  |         }
  44  | 
  45  |         // Wait for responses or timeout
  46  |         setTimeout(() => {
  47  |           socket.close()
  48  |           resolve(received)
  49  |         }, 5000)
  50  |       })
  51  |     })
  52  | 
  53  |     // Verify the page is still responsive after flood
  54  |     await expect(page.locator('#app')).toBeVisible()
  55  | 
  56  |     // No unhandled errors in console
  57  |     const errors: string[] = []
  58  |     page.on('pageerror', (err) => errors.push(err.message))
  59  |     expect(errors).toHaveLength(0)
  60  |   })
  61  | 
  62  |   test('handles large WS payloads without freezing', async ({ page }) => {
  63  |     await page.goto('/')
  64  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  65  | 
  66  |     // Send a few messages with very large payloads
  67  |     await page.evaluate(async () => {
  68  |       return new Promise<void>((resolve) => {
  69  |         const socket = new WebSocket('ws://localhost:3005/ws')
  70  |         const largePayload = 'x'.repeat(100_000) // 100KB payload
  71  | 
  72  |         socket.onopen = () => {
  73  |           for (let i = 0; i < 10; i++) {
  74  |             socket.send(JSON.stringify({
  75  |               protocol: 'ui',
  76  |               method: 'fs.write',
  77  |               params: { path: `/test_${i}`, content: largePayload },
  78  |               id: `large_${i}`,
  79  |               ts: Date.now(),
  80  |             }))
  81  |           }
  82  |         }
  83  | 
  84  |         setTimeout(() => {
  85  |           socket.close()
  86  |           resolve()
  87  |         }, 3000)
  88  |       })
  89  |     })
  90  | 
  91  |     // Page should still be interactive
  92  |     await expect(page.locator('#app')).toBeVisible()
  93  |   })
  94  | 
  95  |   test('survives concurrent WS connections from same page', async ({ page }) => {
  96  |     await page.goto('/')
  97  |     await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })
  98  | 
  99  |     // Open multiple WS connections simultaneously
  100 |     await page.evaluate(async () => {
  101 |       const sockets: WebSocket[] = []
  102 |       for (let i = 0; i < 5; i++) {
  103 |         const socket = new WebSocket('ws://localhost:3005/ws')
  104 |         sockets.push(socket)
  105 |         socket.send(JSON.stringify({
  106 |           protocol: 'ui',
  107 |           method: 'apps.list',
  108 |           id: `conn_${i}`,
  109 |           ts: Date.now(),
  110 |         }))
  111 |       }
```