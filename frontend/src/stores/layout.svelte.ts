// Reactive layout state using createSubscriber from svelte/reactivity
// This is the officially recommended pattern for externally-synced state (e.g., WebSocket)
// See: https://svelte.dev/docs/svelte/svelte-reactivity#createSubscriber
import { createSubscriber } from "svelte/reactivity"
import type { LayoutTree } from "./ws"

function createLayoutState() {
  let tree: LayoutTree | null = $state(null)
  let version = $state(0)
  let updateFn: (() => void) | null = null

  const subscribe = createSubscriber((update) => {
    updateFn = update
    return () => {
      updateFn = null
    }
  })

  return {
    get tree() {
      subscribe()
      return tree
    },
    get version() {
      subscribe()
      return version
    },
    setLayout(newTree: LayoutTree | null, newVersion: number) {
      tree = newTree
      version = newVersion
      // Notify all subscribers that the value changed
      updateFn?.()
    }
  }
}

export const layoutState = createLayoutState()
