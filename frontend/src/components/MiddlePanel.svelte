<script lang="ts">
  import Tile from "./Tile.svelte"
  import { layout, send } from "../stores/ws"

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

    // Open file in editor tile via layout update
    send({ protocol: "ui", method: "layout.update", params: {
      op: "open", content: "editor", path: filePath
    }})
  }
</script>

<div
  class="h-full bg-transparent overflow-hidden p-1 gap-1"
  class:border-purple-500={dropTargetActive}
  class:border-2={dropTargetActive}
  role="region"
  aria-label="Editor area — drop files to open"
  ondragover={handleDragOver}
  ondragleave={handleDragLeave}
  ondrop={handleDrop}
>
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
