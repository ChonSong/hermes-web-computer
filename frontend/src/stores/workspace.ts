import type { LayoutTree } from "./ws"
import { writable, get } from "svelte/store"

export interface FloatingTile {
  id: string
  x: number
  y: number
  width: number
  height: number
  minimized: boolean
}

interface WorkspaceState {
  tree: LayoutTree | null
  floating: Map<string, FloatingTile>
}

interface WorkspaceStoreState {
  activeWorkspace: number
  workspaces: WorkspaceState[]
}

function createWorkspaceState(): WorkspaceState {
  return { tree: null, floating: new Map() }
}

const initialState: WorkspaceStoreState = {
  activeWorkspace: 1,
  workspaces: Array.from({ length: 9 }, () => createWorkspaceState()),
}

// Try to restore from localStorage
try {
  const saved = localStorage.getItem("hwc-workspaces-v1")
  if (saved) {
    const parsed = JSON.parse(saved)
    if (parsed.activeWorkspace) initialState.activeWorkspace = parsed.activeWorkspace
    if (parsed.workspaces) {
      for (let i = 0; i < 9; i++) {
        const ws = parsed.workspaces[i]
        if (ws?.tree) initialState.workspaces[i].tree = ws.tree
        if (ws?.floating) {
          initialState.workspaces[i].floating = new Map(
            Object.entries(ws.floating).map(([k, v]: [string, any]) => [k, v])
          )
        }
      }
    }
  }
} catch {}

export const workspaceStore = writable<WorkspaceStoreState>(initialState)

// Auto-save to localStorage on every change
workspaceStore.subscribe((state) => {
  try {
    const serializable = {
      activeWorkspace: state.activeWorkspace,
      workspaces: state.workspaces.map(ws => ({
        tree: ws.tree,
        floating: Object.fromEntries(ws.floating),
      })),
    }
    localStorage.setItem("hwc-workspaces-v1", JSON.stringify(serializable))
  } catch {}
})

export function getActiveWorkspace(): number {
  return get(workspaceStore).activeWorkspace
}

export function setActiveWorkspace(n: number): void {
  if (n < 1 || n > 9) return
  workspaceStore.update(s => ({ ...s, activeWorkspace: n }))
}

export function getWorkspaceCount(): number {
  return 9
}

export function saveLayout(tree: LayoutTree | null): void {
  workspaceStore.update(s => {
    const ws = s.workspaces[s.activeWorkspace - 1]
    return {
      ...s,
      workspaces: s.workspaces.map((w, i) =>
        i === s.activeWorkspace - 1 ? { ...w, tree } : w
      ),
    }
  })
}

export function getLayoutTree(n: number): LayoutTree | null {
  return get(workspaceStore).workspaces[n - 1].tree
}

export function getFloatingTiles(): FloatingTile[] {
  const s = get(workspaceStore)
  return Array.from(s.workspaces[s.activeWorkspace - 1].floating.values())
}

export function isFloating(tileId: string): boolean {
  const s = get(workspaceStore)
  return s.workspaces[s.activeWorkspace - 1].floating.has(tileId)
}

export function getFloating(tileId: string): FloatingTile | undefined {
  const s = get(workspaceStore)
  return s.workspaces[s.activeWorkspace - 1].floating.get(tileId)
}

export function toggleFloating(
  tileId: string,
  defaultRect?: { x: number; y: number; width: number; height: number }
): void {
  workspaceStore.update(s => {
    const idx = s.activeWorkspace - 1
    const ws = s.workspaces[idx]
    const newFloating = new Map(ws.floating)
    if (newFloating.has(tileId)) {
      newFloating.delete(tileId)
    } else {
      newFloating.set(tileId, {
        id: tileId,
        x: defaultRect?.x ?? 100,
        y: defaultRect?.y ?? 80,
        width: defaultRect?.width ?? 600,
        height: defaultRect?.height ?? 400,
        minimized: false,
      })
    }
    return {
      ...s,
      workspaces: s.workspaces.map((w, i) =>
        i === idx ? { ...w, floating: newFloating } : w
      ),
    }
  })
}

export function updateFloating(tileId: string, updates: Partial<FloatingTile>): void {
  workspaceStore.update(s => {
    const idx = s.activeWorkspace - 1
    const ws = s.workspaces[idx]
    const existing = ws.floating.get(tileId)
    if (!existing) return s
    const newFloating = new Map(ws.floating)
    newFloating.set(tileId, { ...existing, ...updates })
    return {
      ...s,
      workspaces: s.workspaces.map((w, i) =>
        i === idx ? { ...w, floating: newFloating } : w
      ),
    }
  })
}

export function removeFloating(tileId: string): void {
  workspaceStore.update(s => {
    const idx = s.activeWorkspace - 1
    const ws = s.workspaces[idx]
    const newFloating = new Map(ws.floating)
    newFloating.delete(tileId)
    return {
      ...s,
      workspaces: s.workspaces.map((w, i) =>
        i === idx ? { ...w, floating: newFloating } : w
      ),
    }
  })
}

export function moveTileToWorkspace(tileId: string, targetWs: number): void {
  if (targetWs < 1 || targetWs > 9) return
  workspaceStore.update(s => {
    const fromIdx = s.activeWorkspace - 1
    const toIdx = targetWs - 1
    if (fromIdx === toIdx) return s
    const fromWs = s.workspaces[fromIdx]
    const toWs = s.workspaces[toIdx]
    const ft = fromWs.floating.get(tileId)
    if (!ft) return s

    const fromFloating = new Map(fromWs.floating)
    fromFloating.delete(tileId)
    const toFloating = new Map(toWs.floating)
    toFloating.set(tileId, { ...ft })

    return {
      ...s,
      activeWorkspace: targetWs,
      workspaces: s.workspaces.map((w, i) => {
        if (i === fromIdx) return { ...w, floating: fromFloating }
        if (i === toIdx) return { ...w, floating: toFloating }
        return w
      }),
    }
  })
}

export function resetAll(): void {
  workspaceStore.set({
    activeWorkspace: 1,
    workspaces: Array.from({ length: 9 }, () => createWorkspaceState()),
  })
}
