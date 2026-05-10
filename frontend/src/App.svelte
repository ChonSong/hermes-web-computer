<script lang="ts">
  import Terminal from "./components/Terminal.svelte"
  import { ws, layout } from "./stores/ws"

  $: connected = $ws.connected
</script>

<div class="h-screen w-screen bg-gray-950 text-gray-100 flex flex-col">
  {#if connected}
    <div class="flex-1 p-2">
      {#if $layout.tree}
        <Terminal />
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

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    overflow: hidden;
  }
</style>
