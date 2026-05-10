<script lang="ts">
  import Terminal from "./components/Terminal.svelte"
  import { ws, layout } from "./stores/ws"
  import { writable } from "svelte/store"

  $: connected = $ws.connected

  // Border state for visual feedback
  const borderState = writable<{ color: string; state: string }>({ color: "blue", state: "idle" })

  // Listen for border state changes
  import { on } from "./stores/ws"
  on("border.state", (data: unknown) => {
    const d = data as { color: string; state: string }
    borderState.set(d)
  })
</script>

<div class="h-screen w-screen bg-gray-950 text-gray-100 flex flex-col"
     class:border-amber={$borderState.color === "amber"}
     class:border-blue={$borderState.color === "blue"}
     class:border-red={$borderState.color === "red"}
     class:border-2={true}>
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
  .border-amber { border-color: #f59e0b; }
  .border-blue { border-color: #3b82f6; }
  .border-red { border-color: #ef4444; }
</style>
