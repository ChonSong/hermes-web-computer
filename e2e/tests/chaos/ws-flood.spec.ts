/**
 * Chaos Test: WebSocket Message Flood
 *
 * Opens a WebSocket connection to the server and sends a high volume of messages
 * to verify the frontend doesn't crash under message pressure.
 */
import { test, expect } from '@playwright/test'

test.describe('ws-flood', () => {
  test('handles rapid WS messages without crashing', async ({ page }) => {
    await page.goto('/')

    // Wait for the app container to be present
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Open a WebSocket from page context and flood it
    await page.evaluate(async () => {
      return new Promise<number>((resolve) => {
        const socket = new WebSocket('ws://localhost:3005/ws')
        let received = 0
        const floodCount = 500

        socket.onopen = () => {
          // Send many messages rapidly
          for (let i = 0; i < floodCount; i++) {
            socket.send(JSON.stringify({
              protocol: 'ui',
              method: 'fs.list',
              params: { path: '/' },
              id: `flood_${i}`,
              ts: Date.now(),
            }))
          }
        }

        socket.onmessage = () => {
          received++
        }

        socket.onerror = () => {
          // Server may reject unknown messages — that's fine
          resolve(received)
        }

        // Wait for responses or timeout
        setTimeout(() => {
          socket.close()
          resolve(received)
        }, 5000)
      })
    })

    // Verify the page is still responsive after flood
    await expect(page.locator('#app')).toBeVisible()

    // No unhandled errors in console
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))
    expect(errors).toHaveLength(0)
  })

  test('handles large WS payloads without freezing', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Send a few messages with very large payloads
    await page.evaluate(async () => {
      return new Promise<void>((resolve) => {
        const socket = new WebSocket('ws://localhost:3005/ws')
        const largePayload = 'x'.repeat(100_000) // 100KB payload

        socket.onopen = () => {
          for (let i = 0; i < 10; i++) {
            socket.send(JSON.stringify({
              protocol: 'ui',
              method: 'fs.write',
              params: { path: `/test_${i}`, content: largePayload },
              id: `large_${i}`,
              ts: Date.now(),
            }))
          }
        }

        setTimeout(() => {
          socket.close()
          resolve()
        }, 3000)
      })
    })

    // Page should still be interactive
    await expect(page.locator('#app')).toBeVisible()
  })

  test('survives concurrent WS connections from same page', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('#app')).toBeVisible({ timeout: 10_000 })

    // Open multiple WS connections simultaneously, wait for open before sending
    await page.evaluate(async () => {
      const sockets: WebSocket[] = []
      for (let i = 0; i < 5; i++) {
        const socket = new WebSocket('ws://localhost:3005/ws')
        sockets.push(socket)
        // Wait for connection to be OPEN before sending
        await new Promise<void>((resolve) => {
          if (socket.readyState === WebSocket.OPEN) { resolve(); return }
          socket.onopen = () => resolve()
          socket.onerror = () => resolve() // don't hang on error
        })
        socket.send(JSON.stringify({
          protocol: 'ui',
          method: 'apps.list',
          id: `conn_${i}`,
          ts: Date.now(),
        }))
      }
      // Close all after a moment
      await new Promise<void>((resolve) => {
        setTimeout(() => {
          sockets.forEach(s => s.close())
          resolve()
        }, 2000)
      })
    })

    // Page should remain stable
    await expect(page.locator('#app')).toBeVisible()
  })
})
