<script lang="ts">
  import FileTree from "./FileTree.svelte"
  import AppLauncher from "./AppLauncher.svelte"
  import { sendOp, on, appsLaunch, fsRead } from "../stores/ws"

  let activeTab = $state<"files" | "apps">("files")

  function handleFileOpen(event: CustomEvent<{ path: string }>) {
    const path = event.detail.path
    // Create an editor tile in the layout tree
    sendOp({
      op: "split",
      target_id: "root",
      direction: "v",
      content: "editor",
      path: path,
    })
    // Read the file content to send to the editor
    fsRead(path)
  }

  function handleLaunch(event: CustomEvent<{ type: string }>) {
    appsLaunch(event.detail.type)
  }
</script>

<div class="h-full bg-gray-900 border-r border-gray-800 flex flex-col overflow-hidden">
  <!-- Tab bar -->
  <div class="flex border-b border-gray-800 shrink-0">
    <button
      class="flex-1 px-3 py-2 text-sm font-medium transition-colors {activeTab === 'files' ? 'text-white bg-gray-800 border-b-2 border-blue-500' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'}"
      onclick={() => activeTab = "files"}
    >
      📁 Files
    </button>
    <button
      class="flex-1 px-3 py-2 text-sm font-medium transition-colors {activeTab === 'apps' ? 'text-white bg-gray-800 border-b-2 border-blue-500' : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800/50'}"
      onclick={() => activeTab = "apps"}
    >
      🚀 Apps
    </button>
  </div>

  <!-- Content -->
  <div class="flex-1 overflow-y-auto">
    {#if activeTab === "files"}
      <FileTree on:file:open={handleFileOpen} />
    {:else}
      <AppLauncher on:launch={handleLaunch} />
    {/if}
  </div>
</div>
