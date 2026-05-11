<script lang="ts">
  import LeftPanel from "./components/LeftPanel.svelte"
  import MiddlePanel from "./components/MiddlePanel.svelte"
  import RightPanel from "./components/RightPanel.svelte"
  import ResizeHandle from "./components/ResizeHandle.svelte"
  import CommandPalette from "./components/CommandPalette.svelte"
  import KeymapOverlay from "./components/KeymapOverlay.svelte"
  import WorkspacePill from "./components/WorkspacePill.svelte"
  import Dock from "./components/Dock.svelte"
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

<div class="h-screen w-screen text-gray-100 overflow-hidden relative">
  <!-- Gradient background layer -->
  <div class="fixed inset-0 -z-20 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-[#1a0a2e] via-[#0a0a0f] to-[#0a0a0f]"></div>

  <!-- Subtle animated grain overlay -->
  <div
    class="fixed inset-0 -z-10 opacity-[0.03] pointer-events-none"
    style="background-image: url('data:image/svg+xml,%3Csvg viewBox=%220 0 256 256%22 xmlns=%22http://www.w3.org/2000/svg%22%3E%3Cfilter id=%22noise%22%3E%3CfeTurbulence type=%22fractalNoise%22 baseFrequency=%220.9%22 numOctaves=%224%22 stitchTiles=%22stitch%22/%3E%3C/filter%3E%3Crect width=%22100%25%22 height=%22100%25%22 filter=%22url(%23noise)%22/%3E%3C/svg%3E');
           animation: grain 8s steps(10) infinite;"
  ></div>

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
