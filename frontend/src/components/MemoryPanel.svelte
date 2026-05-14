<script lang="ts">
  /**
   * MemoryPanel — Right panel tab for viewing and editing agent memory.
   */
  import { onMount } from "svelte"
  import { memoryStore } from "../stores/memory.svelte"

  let loading = $derived(memoryStore.loading)
  let memory = $derived(memoryStore.memory)
  let user = $derived(memoryStore.user)
  let paths = $derived(memoryStore.paths)
  let error = $derived(memoryStore.error)

  let editMode = $state(false)
  let editMemory = $state("")
  let editUser = $state("")
  let saveStatus = $state<string | null>(null)

  onMount(() => {
    memoryStore.read()
  })

  function startEdit() {
    editMemory = memory
    editUser = user
    editMode = true
  }

  async function saveEdit() {
    saveStatus = "Saving..."
    const ok = await memoryStore.write(editMemory, editUser)
    saveStatus = ok ? "Saved!" : "Save failed"
    setTimeout(() => { saveStatus = null; editMode = false }, 1500)
  }

  function cancelEdit() {
    editMode = false
    saveStatus = null
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10 flex items-center justify-between">
    <h2 class="text-white font-semibold text-base">Memory</h2>
    {#if !editMode && !loading}
      <button
        class="px-2 py-1 text-xs rounded bg-white/10 text-gray-300 hover:bg-white/20 transition-colors"
        onclick={startEdit}
      >
        Edit
      </button>
    {/if}
  </div>

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if loading}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading memory...</div>
  {:else if editMode}
    <!-- Edit mode -->
    <div class="flex-1 overflow-y-auto px-4 py-3 space-y-3">
      <div>
        <label class="text-xs text-gray-400 mb-1 block">User context</label>
        <textarea
          bind:value={editUser}
          rows="3"
          class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50 resize-none"
          placeholder="User identity and preferences..."
        />
      </div>
      <div>
        <label class="text-xs text-gray-400 mb-1 block">Agent memory</label>
        <textarea
          bind:value={editMemory}
          rows="12"
          class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50 resize-none font-mono"
          placeholder="Long-term memory and context..."
        />
      </div>
      <div class="flex gap-2">
        <button
          onclick={saveEdit}
          class="px-3 py-1.5 text-sm rounded bg-purple-600 hover:bg-purple-500 text-white transition-colors"
        >
          Save
        </button>
        <button
          onclick={cancelEdit}
          class="px-3 py-1.5 text-sm rounded bg-white/10 hover:bg-white/20 text-gray-300 transition-colors"
        >
          Cancel
        </button>
        {#if saveStatus}
          <span class="text-sm text-gray-400 self-center">{saveStatus}</span>
        {/if}
      </div>
    </div>
  {:else}
    <!-- View mode -->
    <div class="flex-1 overflow-y-auto px-4 py-3 space-y-3">
      {#if user}
        <div>
          <div class="text-xs text-gray-400 mb-1">User Context</div>
          <div class="text-sm text-gray-200 bg-white/5 rounded-lg px-3 py-2">{user}</div>
        </div>
      {/if}
      {#if memory}
        <div>
          <div class="text-xs text-gray-400 mb-1">Agent Memory</div>
          <div class="text-sm text-gray-300 bg-white/5 rounded-lg px-3 py-2 font-mono whitespace-pre-wrap">{memory}</div>
        </div>
      {/if}
      {#if paths.length > 0}
        <div>
          <div class="text-xs text-gray-400 mb-1">Memory Files</div>
          <div class="space-y-1">
            {#each paths as path}
              <div class="text-xs text-gray-500 font-mono truncate">{path}</div>
            {/each}
          </div>
        </div>
      {/if}
      {#if !user && !memory && paths.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No memory data</div>
      {/if}
    </div>
  {/if}
</div>