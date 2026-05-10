<script lang="ts">
  import Tile from "./components/Tile.svelte"
  import CommandPalette from "./components/CommandPalette.svelte"
  import KeymapOverlay from "./components/KeymapOverlay.svelte"
  import { ws, layout } from "./stores/ws"

  $: connected = $ws.connected
  let showPalette = $state(false)
  let showKeymap = $state(false)

  // Global keyboard shortcuts
  globalThis.addEventListener("keydown", (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "k") { e.preventDefault(); showPalette = !showPalette }
    if ((e.ctrlKey || e.metaKey) && e.key === "?") { e.preventDefault(); showKeymap = !showKeymap }
    if (e.key === "Escape") { showPalette = false; showKeymap = false }
  }, true)
</script>

<div class="h-screen w-screen bg-gray-950 text-gray-100 flex flex-col border-2 border-blue-500">
  {#if connected}
    <div class="flex-1 p-1">
      {#if $layout.tree}
        <Tile node={$layout.tree} />
      {:else}
        <div class="flex items-center justify-center h-full text-gray-500">
          <p>Connecting to layout server...</p>
        </div>
      {/if}
    </div>
  {:else}
    <div class="flex items-center justify-center h-full">
      <p class="text-gray-500">Disconnected — reconnecting...</p>
    </div>
  {/if}
</div>

<CommandPalette bind:visible={showPalette} />
<KeymapOverlay bind:visible={showKeymap} />
