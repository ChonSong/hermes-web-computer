<script lang="ts">
  import LeftPanel from "./components/LeftPanel.svelte"
  import MiddlePanel from "./components/MiddlePanel.svelte"
  import RightPanel from "./components/RightPanel.svelte"
  import ResizeHandle from "./components/ResizeHandle.svelte"
  import CommandPalette from "./components/CommandPalette.svelte"
  import KeymapOverlay from "./components/KeymapOverlay.svelte"
  import WorkspacePill from "./components/WorkspacePill.svelte"
  import Dock from "./components/Dock.svelte"
  import { workspaceStore, setActiveWorkspace, moveTileToWorkspace, toggleFloating } from "./stores/workspace"
  import { ws, send, focus, layout, type LayoutTree } from "./stores/ws"
  import { setContext, getContext } from "svelte"

  let connected = $derived($ws.connected)
  let showPalette = $state(false)
  let showKeymap = $state(false)
  let showLeft = $state(true)
  let showRight = $state(true)
  // Active workspace from shared store (auto-subscribed, persists to localStorage)
  let activeWorkspace = $derived($workspaceStore.activeWorkspace)

  // Panel width state
  let savedWidths: { left?: number; right?: number } = {}
  try { savedWidths = JSON.parse(localStorage.getItem('ao-col-widths') || '{}') } catch {}
  let leftW = $state(savedWidths.left ?? 280)
  let rightW = $state(savedWidths.right ?? 360)

  function saveWidths() {
    localStorage.setItem('ao-col-widths', JSON.stringify({ left: leftW, right: rightW }))
  }

  // Layout flash for Shift+D cycle feedback
  let layoutFlash = $state("")
  let layoutFlashTimer: ReturnType<typeof setTimeout> | null = null
  const layoutModes = ["master-stack", "even-split", "columns", "rows"] as const
  let layoutModeIdx = $state(0)

  function flashLayout(name: string) {
    layoutFlash = name
    if (layoutFlashTimer) clearTimeout(layoutFlashTimer)
    layoutFlashTimer = setTimeout(() => { layoutFlash = "" }, 800)
  }

  // Focus tracking for keyboard navigation
  // We collect all leaf tile IDs from the layout tree to cycle focus
  function getLeafIds(node: LayoutTree | null): string[] {
    if (!node) return []
    if (node.type === "split" && node.children) {
      return node.children.flatMap(c => getLeafIds(c))
    }
    return [node.id]
  }

  let leafIds = $derived(getLeafIds($layout.tree))

  function focusAdjacentTile(direction: "left" | "right" | "up" | "down") {
    const current = $focus
    const ids = leafIds
    if (ids.length <= 1) return
    const idx = ids.indexOf(current)
    if (idx === -1) { focus.set(ids[0]); return }

    // For a simple linear ordering, map directional intent to next/prev
    // A full spatial lookup would need bounding boxes — we use wrap-around
    // cycling as a practical baseline that works with any layout.
    if (direction === "right" || direction === "down") {
      focus.set(ids[(idx + 1) % ids.length])
    } else {
      focus.set(ids[(idx - 1 + ids.length) % ids.length])
    }
  }

  // Full keyboard shortcut map
  globalThis.addEventListener("keydown", (e: KeyboardEvent) => {
    // Ctrl+K: command palette (capture phase to beat app inputs)
    if ((e.ctrlKey || e.metaKey) && e.key === "k") {
      e.preventDefault()
      showPalette = !showPalette
      return
    }
    // Ctrl+?: keymap overlay
    if ((e.ctrlKey || e.metaKey) && e.key === "?") {
      e.preventDefault()
      showKeymap = !showKeymap
      return
    }
    // Ctrl+B: toggle left panel
    if ((e.ctrlKey || e.metaKey) && e.key === "b") {
      e.preventDefault()
      showLeft = !showLeft
      return
    }
    // Ctrl+Shift+B: toggle right panel
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "B") {
      e.preventDefault()
      showRight = !showRight
      return
    }
    // Escape: close overlays
    if (e.key === "Escape") {
      showPalette = false
      showKeymap = false
      return
    }

    // Shift+Arrow: focus adjacent tile
    if (e.shiftKey && !e.altKey && ["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(e.key)) {
      e.preventDefault()
      const dirMap: Record<string, "left" | "right" | "up" | "down"> = {
        ArrowLeft: "left", ArrowRight: "right", ArrowUp: "up", ArrowDown: "down"
      }
      focusAdjacentTile(dirMap[e.key])
      return
    }

    // Shift+Alt+Arrow: resize tile (send resize op to backend)
    if (e.shiftKey && e.altKey && ["ArrowLeft", "ArrowRight", "ArrowUp", "ArrowDown"].includes(e.key)) {
      e.preventDefault()
      const currentFocus = $focus
      const dirMap: Record<string, string> = {
        ArrowLeft: "shrink-left", ArrowRight: "grow-right",
        ArrowUp: "shrink-top", ArrowDown: "grow-bottom"
      }
      send({ protocol: "ui", method: "layout.update", params: {
        op: "resize", direction: dirMap[e.key], target_id: currentFocus
      }})
      return
    }

    // Shift+D: cycle layout modes
    if (e.shiftKey && e.key === "d" && !e.altKey && !e.ctrlKey && !e.metaKey) {
      e.preventDefault()
      layoutModeIdx = (layoutModeIdx + 1) % layoutModes.length
      flashLayout(layoutModes[layoutModeIdx])
      // Send layout change to backend
      send({ protocol: "ui", method: "layout.update", params: {
        op: "layout-mode", mode: layoutModes[layoutModeIdx]
      }})
      return
    }

    // Shift+F: toggle fullscreen on focused tile
    if (e.shiftKey && e.key === "f" && !e.altKey && !e.ctrlKey && !e.metaKey) {
      e.preventDefault()
      const currentFocus = $focus
      send({ protocol: "ui", method: "layout.update", params: {
        op: "fullscreen", target_id: currentFocus
      }})
      return
    }

    // Shift+Q: close focused tile
    if (e.shiftKey && e.key === "q" && !e.altKey && !e.ctrlKey && !e.metaKey) {
      e.preventDefault()
      const currentFocus = $focus
      send({ protocol: "ui", method: "layout.update", params: {
        op: "unmount", target_id: currentFocus
      }})
      return
    }

    // Shift+1-9: switch workspace (via shared store, persists to localStorage)
    if (e.shiftKey && !e.altKey && e.key >= "1" && e.key <= "9") {
      e.preventDefault()
      setActiveWorkspace(parseInt(e.key, 10))
      return
    }

    // Shift+Alt+1-9: move focused tile to workspace
    if (e.shiftKey && e.altKey && e.key >= "1" && e.key <= "9") {
      e.preventDefault()
      const targetWs = parseInt(e.key, 10)
      const currentFocus = $focus
      moveTileToWorkspace(currentFocus, targetWs)
      toggleFloating(currentFocus)
      return
    }

    // Shift+Space: toggle float focused tile
    if (e.shiftKey && e.key === " " && !e.altKey && !e.ctrlKey && !e.metaKey) {
      e.preventDefault()
      const currentFocus = $focus
      toggleFloating(currentFocus)
      return
    }
  }, true)

  // Provide workspace state via context for WorkspacePill
  setContext("activeWorkspace", {
    get: () => $workspaceStore.activeWorkspace,
    set: setActiveWorkspace
  })
</script>

<div class="h-screen w-screen text-gray-100 overflow-hidden relative">
  <!-- Gradient background layer -->
  <div class="fixed inset-0 -z-20 bg-[#0a0a0f]"></div>

  <!-- Subtle animated grain overlay -->
  <div
    class="fixed inset-0 -z-10 opacity-[0.03] pointer-events-none"
    style="background-image: url('data:image/svg+xml,%3Csvg viewBox=%220 0 256 256%22 xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cfilter id=%22noise%22%3E%3CfeTurbulence type=%22fractalNoise%22 baseFrequency=%220.9%22 numOctaves=%224%22 stitchTiles=%22stitch%22/%3E%3C/filter%3E%3Crect width=%22100%25%22 height=%22100%25%22 filter=%22url(%23noise)%22/%3E%3C/svg%3E');
           animation: grain 8s steps(10) infinite;"
  ></div>

  <!-- Layout mode flash indicator -->
  {#if layoutFlash}
    <div class="fixed top-10 left-1/2 -translate-x-1/2 z-50
      backdrop-blur-xl bg-[#16161e]/90 border border-purple-500/30 rounded-xl
      px-4 py-2 text-sm text-purple-300 font-mono shadow-glow animate-fade-in">
      Layout: {layoutFlash}
    </div>
  {/if}

  {#if !connected}
    <div class="flex items-center justify-center h-full">
      <p class="text-gray-500">Disconnected</p>
    </div>
  {:else}
    <div class="h-full grid" style="grid-template-columns: {showLeft ? leftW + 'px' : '0px'} {showLeft ? '4px' : '0px'} 1fr {showRight ? '4px' : '0px'} {showRight ? rightW + 'px' : '0px'};">
      {#if showLeft}
        <LeftPanel />
        <ResizeHandle side="left" bind:width={leftW} onWidthChange={saveWidths} />
      {/if}
      <MiddlePanel />
      {#if showRight}
        <ResizeHandle side="right" bind:width={rightW} onWidthChange={saveWidths} />
        <RightPanel />
      {/if}
    </div>
  {/if}

  <!-- Illogical Impulse overlay components -->
  <WorkspacePill />
  <Dock />
</div>

<CommandPalette bind:visible={showPalette} />
<KeymapOverlay bind:visible={showKeymap} />

<style>
  @keyframes grain {
    0%, 100% { transform: translate(0, 0); }
    10% { transform: translate(-5%, -10%); }
    20% { transform: translate(-15%, 5%); }
    30% { transform: translate(7%, -25%); }
    40% { transform: translate(-5%, 25%); }
    50% { transform: translate(-15%, 10%); }
    60% { transform: translate(15%, 0%); }
    70% { transform: translate(0%, 15%); }
    80% { transform: translate(3%, 35%); }
    90% { transform: translate(-10%, 10%); }
  }
</style>
