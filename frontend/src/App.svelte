<script lang="ts">
  import LeftPanel from "./components/LeftPanel.svelte"
  import MiddlePanel from "./components/MiddlePanel.svelte"
  import RightPanel from "./components/RightPanel.svelte"
  import ResizeHandle from "./components/ResizeHandle.svelte"
  import CommandPalette from "./components/CommandPalette.svelte"
  import KeymapOverlay from "./components/KeymapOverlay.svelte"
  import { ws } from "./stores/ws"

  let connected = $derived($ws.connected)
  let showPalette = $state(false)
  let showKeymap = $state(false)
  let showLeft = $state(true)
  let showRight = $state(true)

  let savedWidths: { left?: number; right?: number } = {}
  try { savedWidths = JSON.parse(localStorage.getItem('ao-col-widths') || '{}') } catch {}
  let leftW = $state(savedWidths.left ?? 280)
  let rightW = $state(savedWidths.right ?? 360)

  function saveWidths() {
    localStorage.setItem('ao-col-widths', JSON.stringify({ left: leftW, right: rightW }))
  }

  globalThis.addEventListener("keydown", (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "k") { e.preventDefault(); showPalette = !showPalette }
    if ((e.ctrlKey || e.metaKey) && e.key === "?") { e.preventDefault(); showKeymap = !showKeymap }
    if ((e.ctrlKey || e.metaKey) && e.key === "b") { e.preventDefault(); showLeft = !showLeft }
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "B") { e.preventDefault(); showRight = !showRight }
    if (e.key === "Escape") { showPalette = false; showKeymap = false }
  }, true)
</script>

<div class="h-screen w-screen bg-gray-950 text-gray-100 overflow-hidden">
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
</div>

<CommandPalette bind:visible={showPalette} />
<KeymapOverlay bind:visible={showKeymap} />
