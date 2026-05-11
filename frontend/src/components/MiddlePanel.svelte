<script lang="ts">
  import Tile from "./Tile.svelte"
  import { layout } from "../stores/ws"
</script>

<div class="h-full bg-transparent overflow-hidden p-1 gap-1">
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
</div>
