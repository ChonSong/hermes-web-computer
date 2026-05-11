<script lang="ts">
  import FileTree from "./FileTree.svelte"
  import AppLauncher from "./AppLauncher.svelte"
  import { sendOp, on, appsLaunch, fsRead } from "../stores/ws"

  let activeTab = $state<"files" | "apps">("files")
  let collapsed = $state(false)

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

<div
  class="h-full mt-12 ml-1 mb-1 flex flex-col overflow-hidden transition-all duration-200 ease-out
    backdrop-blur-2xl bg-[#12121a]/80 border border-white/10 rounded-2xl shadow-panel"
  class:w-0={collapsed}
  class:w-[240px]={!collapsed}
  class:opacity-0={collapsed}
>
  <!-- Tab bar -->
  <div class="flex shrink-0 px-2 pt-2 gap-1">
    <button
      class="flex-1 px-3 py-1.5 text-sm font-medium transition-colors rounded-lg
        {activeTab === 'files' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
      onclick={() => activeTab = "files"}
    >
      📁 Files
    </button>
    <button
      class="flex-1 px-3 py-1.5 text-sm font-medium transition-colors rounded-lg
        {activeTab === 'apps' ? 'text-white bg-white/10' : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
      onclick={() => activeTab = "apps"}
    >
      🚀 Apps
    </button>
    <button
      class="w-8 h-8 flex items-center justify-center text-gray-400 hover:text-white transition-colors rounded-lg hover:bg-white/5"
      onclick={() => collapsed = !collapsed}
      aria-label="Toggle panel"
    >
      ◀
    </button>
  </div>

  <!-- Content -->
  <div class="flex-1 overflow-y-auto px-1 pb-1">
    {#if activeTab === "files"}
      <FileTree on:file:open={handleFileOpen} />
    {:else}
      <AppLauncher on:launch={handleLaunch} />
    {/if}
  </div>
</div>

<!-- Collapse trigger edge -->
{#if collapsed}
  <div
    class="absolute left-0 top-12 bottom-1 w-[3px] bg-purple-500/40 cursor-pointer hover:bg-purple-500 transition-colors"
    onclick={() => collapsed = false}
    aria-label="Expand panel"
  />
{/if}
