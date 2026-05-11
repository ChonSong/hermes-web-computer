<script lang="ts">
  import Tile from "./Tile.svelte"
  import { layout } from "../stores/ws"
</script>

<div class="h-full bg-gray-950 overflow-hidden">
  {#if $layout.tree}
    <Tile node={$layout.tree} />
  {:else}
    <div class="flex items-center justify-center h-full text-gray-500">
      <p>Connecting...</p>
    </div>
  {/if}
</div>
