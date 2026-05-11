import type { LayoutTree } from "./ws"

// Floating tile state
export interface FloatingTile {
  id: string
  x: number
  y: number
  width: number
  height: number
  minimized: boolean
}

// Per-workspace state
export interface WorkspaceState {
  tree: LayoutTree | null
  floating: Map<string, FloatingTile>
}

function createWorkspaceState(): WorkspaceState {
  return { tree: null, floating: new Map() }
}

// Mutable singleton state (reactivity handled in .svelte files via $derived)
const workspaces: WorkspaceState[] = Array.from({ length: 9 }, () => createWorkspaceState())
let activeWorkspace = 1

export function getActiveWorkspace(): number {
  return activeWorkspace
}

export function setActiveWorkspace(n: number): void {
  if (n < 1 || n > 9) return
  activeWorkspace = n
}

export function getWorkspaceCount(): number {
  return 9
}

/** Save current layout tree to the active workspace */
export function saveLayout(tree: LayoutTree | null): void {
  workspaces[activeWorkspace - 1].tree = tree
}

/** Get layout tree for a specific workspace */
export function getLayoutTree(n: number): LayoutTree | null {
  return workspaces[n - 1].tree
}

/** Get floating tiles for active workspace */
export function getFloatingTiles(): FloatingTile[] {
  return Array.from(workspaces[activeWorkspace - 1].floating.values())
}

/** Check if a tile is floating in active workspace */
export function isFloating(tileId: string): boolean {
  return workspaces[activeWorkspace - 1].floating.has(tileId)
}

/** Get floating state for a tile in active workspace */
export function getFloating(tileId: string): FloatingTile | undefined {
  return workspaces[activeWorkspace - 1].floating.get(tileId)
}

/** Toggle floating mode for a tile in active workspace */
export function toggleFloating(
  tileId: string,
  defaultRect?: { x: number; y: number; width: number; height: number }
): void {
  const ws = workspaces[activeWorkspace - 1]
  if (ws.floating.has(tileId)) {
    ws.floating.delete(tileId)
  } else {
    ws.floating.set(tileId, {
      id: tileId,
      x: defaultRect?.x ?? 100,
      y: defaultRect?.y ?? 80,
      width: defaultRect?.width ?? 600,
      height: defaultRect?.height ?? 400,
      minimized: false,
    })
  }
}

/** Update floating tile position/size in active workspace */
export function updateFloating(tileId: string, updates: Partial<FloatingTile>): void {
  const existing = workspaces[activeWorkspace - 1].floating.get(tileId)
  if (existing) {
    Object.assign(existing, updates)
  }
}

/** Remove a floating tile from active workspace */
export function removeFloating(tileId: string): void {
  workspaces[activeWorkspace - 1].floating.delete(tileId)
}

/** Reset all workspaces */
export function resetAll(): void {
  for (let i = 0; i < 9; i++) {
    workspaces[i] = createWorkspaceState()
  }
  activeWorkspace = 1
}
