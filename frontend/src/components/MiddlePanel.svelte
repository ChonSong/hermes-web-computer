<script lang="ts">
  import Tile from "./Tile.svelte"
  import { layout, send, focus } from "../stores/ws"
  import { workspaceStore, getFloatingTiles, isFloating, updateFloating, removeFloating } from "../stores/workspace"
  import type { FloatingTile } from "../stores/workspace"
  import type { LayoutTree } from "../stores/ws"

  let wsState = $derived($workspaceStore)
  let floatingTiles = $derived(Array.from(wsState.workspaces[wsState.activeWorkspace - 1].floating.values()))

  // Find layout node by id for floating tile content
  function findNode(tree: LayoutTree | null, id: string): LayoutTree | null {
    if (!tree) return null
    if (tree.id === id) return tree
    if (tree.children) {
      for (const child of tree.children) {
        const found = findNode(child, id)
        if (found) return found
      }
    }
    return null
  }

  let dropTargetActive = $state(false)

  function handleDragOver(e: DragEvent) {
    e.preventDefault()
    e.dataTransfer!.dropEffect = "copy"
    dropTargetActive = true
  }

  function handleDragLeave(e: DragEvent) {
    if (!e.currentTarget.contains(e.relatedTarget as Node)) {
      dropTargetActive = false
    }
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault()
    dropTargetActive = false
    const filePath = e.dataTransfer?.getData("text/plain")
    if (!filePath) return

    send({ protocol: "ui", method: "layout.update", params: {
      op: "open", content: "editor", path: filePath
    }})
  }

  // Floating tile drag state
  let dragTile = $state<{ id: string; startX: number; startY: number; origX: number; origY: number } | null>(null)
  let dragTitle = $state<{ id: string; startX: number; startY: number; origX: number; origY: number } | null>(null)

  function onTitleMouseDown(e: MouseEvent, ft: FloatingTile) {
    if ((e.target as HTMLElement).closest("button")) return
    dragTitle = { id: ft.id, startX: e.clientX, startY: e.clientY, origX: ft.x, origY: ft.y }
    e.preventDefault()
  }

  function onBodyMouseDown(e: MouseEvent, ft: FloatingTile) {
    // Bring to front: move to end of map
    workspaceStore.update(s => {
      const idx = s.activeWorkspace - 1
      const ws = s.workspaces[idx]
      const existing = ws.floating.get(ft.id)
      if (!existing) return s
      const newFloating = new Map(ws.floating)
      newFloating.delete(ft.id)
      newFloating.set(ft.id, existing)
      return { ...s, workspaces: s.workspaces.map((w, i) => i === idx ? { ...w, floating: newFloating } : w) }
    })
  }

  globalThis.addEventListener("mousemove", (e: MouseEvent) => {
    if (dragTitle) {
      const dx = e.clientX - dragTitle.startX
      const dy = e.clientY - dragTitle.startY
      updateFloating(dragTitle.id, { x: dragTitle.origX + dx, y: dragTitle.origY + dy })
    }
  })

  globalThis.addEventListener("mouseup", () => {
    dragTitle = null
  })

  function contentLabel(content: string): string {
    const labels: Record<string, string> = {
      "xterm": "Terminal", "editor": "Editor", "browser": "Browser",
      "dash-overview": "Dashboard", "dash-filemanager": "Files",
      "dash-observability": "Observability", "dash-analytics": "Analytics",
      "dash-system-status": "System", "welcome": "Welcome",
    }
    return labels[content] || content
  }

  function closeFloat(ft: FloatingTile) {
    removeFloating(ft.id)
    send({ protocol: "ui", method: "layout.update", params: { op: "unmount", target_id: ft.id }})
  }

  function toggleMinimize(ft: FloatingTile) {
    updateFloating(ft.id, { minimized: !ft.minimized })
  }

  function maximize(ft: FloatingTile) {
    updateFloating(ft.id, { x: 8, y: 8, width: window.innerWidth - 300, height: window.innerHeight - 16 })
  }
</script>

<div
  class="relative h-full bg-transparent overflow-hidden p-1 gap-1"
  class:border-purple-500={dropTargetActive}
  class:border-2={dropTargetActive}
  role="region"
  aria-label="Editor area — drop files to open"
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
>
  <!-- Tiled layout -->
  {#if $layout.tree}
    <Tile node={$layout.tree} />
  {:else}
    <div class="flex items-center justify-center h-full text-gray-500">
      <div class="text-center">
        <p class="text-lg font-bold text-gray-400">Agent-OS v1.2</p>
        <p class="text-sm mt-2 text-gray-500">Connecting...</p>
      </div>
    </div>
  {/if}

  <!-- Floating tiles overlay -->
  {#each floatingTiles as ft (ft.id)}
    <div
      class="absolute z-50 backdrop-blur-2xl bg-[#12121a]/92 border border-white/10 rounded-2xl shadow-[0_8px_32px_rgba(0,0,0,0.4),0_0_0_1px_rgba(168,85,247,0.1)] transition-shadow duration-200 overflow-hidden"
      style="left: {ft.x}px; top: {ft.y}px; width: {ft.width}px; height: {ft.height}px;"
      onmousedown={(e) => onBodyMouseDown(e, ft)}
    >
      <!-- Title bar (drag handle) -->
      <div
        class="flex items-center h-8 px-2 bg-white/[0.03] border-b border-white/[0.05] select-none cursor-grab active:cursor-grabbing"
        onmousedown={(e) => onTitleMouseDown(e, ft)}
      >
        <span class="flex-1 text-[11px] text-purple-300/80 font-mono truncate pl-1">
          {contentLabel($layout.tree?.id === ft.id ? $layout.tree.content || '' : 'tile')}: {ft.id.slice(0, 8)}
        </span>
        <button class="w-4 h-4 rounded-full bg-yellow-500/60 hover:bg-yellow-500 text-[9px] text-black/60 flex items-center justify-center ml-1.5" title="Minimize"
          onclick={() => toggleMinimize(ft)}>−</button>
        <button class="w-4 h-4 rounded-full bg-green-500/60 hover:bg-green-500 text-[9px] text-white flex items-center justify-center ml-1.5" title="Maximize"
          onclick={() => maximize(ft)}>□</button>
        <button class="w-4 h-4 rounded-full bg-red-500/60 hover:bg-red-500 text-[9px] text-white flex items-center justify-center ml-1.5" title="Close"
          onclick={() => closeFloat(ft)}>×</button>
      </div>

      <!-- Content -->
      {#if !ft.minimized}
        {@const node = findNode($layout.tree, ft.id)}
        {#if node}
          <div class="h-[calc(100%-2rem)] overflow-hidden">
            <Tile node={node} depth={0} />
          </div>
        {:else}
          <div class="h-[calc(100%-2rem)] flex items-center justify-center text-gray-500 text-sm">
            Tile not found in layout
          </div>
        {/if}
      {:else}
        <div class="h-8 flex items-center px-2 text-xs text-gray-500 italic">
          Minimized
        </div>
      {/if}
    </div>
  {/each}
</div>
