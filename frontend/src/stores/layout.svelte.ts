// Global reactive layout state using class with $state
// This pattern works in Svelte 5 for cross-component reactivity
import type { LayoutTree } from './ws'

class LayoutState {
  tree: LayoutTree | null = $state(null)
  version: number = $state(0)

  set(tree: LayoutTree | null, version: number) {
    this.tree = tree
    this.version = version
  }
}

export const layoutState = new LayoutState()
