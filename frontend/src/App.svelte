<script lang="ts">
  import Tile from "./components/Tile.svelte"
  import CommandPalette from "./components/CommandPalette.svelte"
  import KeymapOverlay from "./components/KeymapOverlay.svelte"
  import { ws, layout } from "./stores/ws"

  let connected = $derived($ws.connected)
  let showPalette = $state(false)
  let showKeymap = $state(false)

  globalThis.addEventListener("keydown", (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === "k") { e.preventDefault(); showPalette = !showPalette }
    if ((e.ctrlKey || e.metaKey) && e.key === "?") { e.preventDefault(); showKeymap = !showKeymap }
    if (e.key === "Escape") { showPalette = false; showKeymap = false }
  }, true)
</script>

<div class="fixed inset-0 bg-gray-950 text-gray-100 flex flex-col">
  {#if connected}
    <div class="flex-1 p-1">
      {#if $layout.tree}
        <Tile node={$layout.tree} />
      {:else}
        <div class="flex items-center justify-center h-full text-gray-500">
          <p>Connecting...</p>
        </div>
      {/if}
    </div>
  {:else}
    <div class="flex items-center justify-center h-full">
      <p class="text-gray-500">Disconnected</p>
    </div>
  {/if}
</div>

<CommandPalette bind:visible={showPalette} />
<KeymapOverlay bind:visible={showKeymap} />
