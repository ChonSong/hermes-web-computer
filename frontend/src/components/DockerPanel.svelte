<script lang="ts">
  /**
   * DockerPanel — Right panel tab for container lifecycle management.
   * Shows running/stopped containers, stats, logs, and lifecycle actions.
   */
  import { onMount, onDestroy } from "svelte"
  import { dockerStore, type Container, type ContainerStats } from "../stores/docker.svelte"

  // Derived state
  let loading = $derived(dockerStore.loading)
  let containers = $derived(dockerStore.filteredContainers)
  let error = $derived(dockerStore.error)
  let searchQuery = $derived(dockerStore.searchQuery)
  let selectedId = $derived(dockerStore.selectedId)
  let stats = $derived(dockerStore.stats)
  let logs = $derived(dockerStore.logs)

  // Local UI state
  let confirmRemoveId = $state<string | null>(null)
  let autoRefresh = $state(true)
  let showLogs = $state(false)
  let refreshInterval: ReturnType<typeof setInterval> | null = null

  onMount(() => {
    dockerStore.refresh()
    startAutoRefresh()
  })

  onDestroy(() => {
    stopAutoRefresh()
  })

  function startAutoRefresh() {
    stopAutoRefresh()
    if (autoRefresh) {
      refreshInterval = setInterval(() => {
        dockerStore.refresh()
        if (selectedId) {
          dockerStore.fetchStats(selectedId)
        }
      }, 30000)
    }
  }

  function stopAutoRefresh() {
    if (refreshInterval) {
      clearInterval(refreshInterval)
      refreshInterval = null
    }
  }

  $effect(() => {
    if (autoRefresh) {
      startAutoRefresh()
    } else {
      stopAutoRefresh()
    }
  })

  function shortId(id: string): string {
    return id.length > 12 ? id.substring(0, 12) : id
  }

  function stateColor(state: string): string {
    switch (state) {
      case "running": return "bg-green-500/20 text-green-400"
      case "paused": return "bg-yellow-500/20 text-yellow-400"
      case "exited": return "bg-gray-500/20 text-gray-400"
      case "dead": return "bg-red-500/20 text-red-400"
      case "restarting": return "bg-blue-500/20 text-blue-400"
      default: return "bg-white/10 text-gray-400"
    }
  }

  function stateIcon(state: string): string {
    switch (state) {
      case "running": return "▶"
      case "paused": return "⏸"
      case "exited": return "■"
      case "dead": return "✕"
      case "restarting": return "↻"
      default: return "○"
    }
  }

  function selectContainer(id: string) {
    if (selectedId === id) {
      dockerStore.select(null)
      showLogs = false
    } else {
      dockerStore.select(id)
      showLogs = false
    }
  }

  function toggleLogs() {
    showLogs = !showLogs
  }

  async function handleStart(id: string) {
    await dockerStore.start(id)
  }

  async function handleStop(id: string) {
    await dockerStore.stop(id)
  }

  async function handleRestart(id: string) {
    await dockerStore.restart(id)
  }

  async function handleRemove(id: string) {
    if (confirmRemoveId === id) {
      await dockerStore.remove(id)
      confirmRemoveId = null
      if (selectedId === id) {
        dockerStore.select(null)
      }
    } else {
      confirmRemoveId = id
      setTimeout(() => { confirmRemoveId = null }, 3000)
    }
  }

  async function refreshAll() {
    await dockerStore.refresh()
    if (selectedId) {
      await dockerStore.fetchStats(selectedId)
      await dockerStore.fetchLogs(selectedId)
    }
  }

  function formatBytes(s: string): string {
    if (!s) return "0 B"
    const parts = s.split(/[\s]+/)
    if (parts.length >= 2) return `${parts[0]} ${parts[1]}`
    return s
  }

  let selectedStats = $derived(selectedId ? stats.get(selectedId) : null) as ContainerStats | null | undefined
  let selectedLogs = $derived(selectedId ? logs.get(selectedId) : null) as string | null | undefined
</script>

<div class="flex flex-col h-full">
  <!-- Header -->
  <div class="flex-none px-4 py-3 border-b border-white/10 flex items-center justify-between">
    <h2 class="text-white font-semibold text-base">Containers</h2>
    <div class="flex items-center gap-2">
      <label class="flex items-center gap-1 text-xs text-gray-400 cursor-pointer">
        <input
          type="checkbox"
          bind:checked={autoRefresh}
          class="accent-purple-500"
        />
        Auto
      </label>
      <button
        class="w-6 h-6 flex items-center justify-center text-xs rounded hover:bg-white/10 text-gray-400 hover:text-white transition-colors"
        onclick={refreshAll}
        title="Refresh"
      >
        ↻
      </button>
    </div>
  </div>

  <!-- Search -->
  <div class="flex-none px-4 py-2 border-b border-white/5">
    <input
      type="text"
      bind:value={dockerStore.searchQuery}
      placeholder="Filter by name, image, or state..."
      class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
    />
  </div>

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if loading && containers.length === 0}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading containers...</div>
  {:else}
    <!-- Container list -->
    <div class="flex-1 overflow-y-auto">
      {#if containers.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No containers found</div>
      {:else}
        <div class="px-2 py-1">
          {#each containers as container (container.id)}
            <div
              class="group rounded-lg mb-1 transition-colors cursor-pointer
                     {selectedId === container.id ? 'bg-white/10' : 'hover:bg-white/5'}"
              onclick={() => selectContainer(container.id)}
            >
              <!-- Main row -->
              <div class="flex items-center gap-2 px-2 py-2">
                <!-- State indicator -->
                <span class="w-2 h-2 rounded-full shrink-0 {container.state === 'running' ? 'bg-green-400' : container.state === 'paused' ? 'bg-yellow-400' : 'bg-gray-500'}"></span>

                <!-- Info -->
                <div class="flex-1 min-w-0">
                  <div class="text-sm text-gray-200 font-medium truncate">{container.name}</div>
                  <div class="text-[10px] text-gray-500 font-mono truncate">{shortId(container.id)} · {container.image}</div>
                </div>

                <!-- State badge -->
                <span class="px-1.5 py-0.5 rounded text-[10px] font-medium {stateColor(container.state)}">
                  {stateIcon(container.state)} {container.state}
                </span>

                <!-- Action buttons -->
                <div class="flex items-center gap-1 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity" onclick={(e) => e.stopPropagation()}>
                  {#if container.state !== "running"}
                    <button
                      class="w-5 h-5 flex items-center justify-center text-xs rounded hover:bg-green-500/20 text-green-400 transition-colors"
                      onclick={() => handleStart(container.id)}
                      title="Start"
                    >
                      ▶
                    </button>
                  {:else}
                    <button
                      class="w-5 h-5 flex items-center justify-center text-xs rounded hover:bg-yellow-500/20 text-yellow-400 transition-colors"
                      onclick={() => handleStop(container.id)}
                      title="Stop"
                    >
                      ■
                    </button>
                  {/if}
                  <button
                    class="w-5 h-5 flex items-center justify-center text-xs rounded hover:bg-blue-500/20 text-blue-400 transition-colors"
                    onclick={() => handleRestart(container.id)}
                    title="Restart"
                  >
                    ↻
                  </button>
                  <button
                    class="w-5 h-5 flex items-center justify-center text-xs rounded transition-colors
                           {confirmRemoveId === container.id ? 'bg-red-500/40 text-red-300' : 'hover:bg-red-500/20 text-red-400'}"
                    onclick={() => handleRemove(container.id)}
                    title={confirmRemoveId === container.id ? "Click again to confirm" : "Remove"}
                  >
                    ✕
                  </button>
                </div>
              </div>

              <!-- Expanded: Stats + Logs -->
              {#if selectedId === container.id}
                <div class="px-3 pb-3 pt-1 border-t border-white/5">
                  <!-- Stats row -->
                  {#if selectedStats}
                    <div class="grid grid-cols-4 gap-2 mb-2">
                      <div class="bg-white/5 rounded p-2 text-center">
                        <div class="text-[10px] text-gray-500 mb-0.5">CPU</div>
                        <div class="text-sm font-mono text-gray-200">{selectedStats.cpu_percent?.toFixed(1) ?? "0"}%</div>
                      </div>
                      <div class="bg-white/5 rounded p-2 text-center">
                        <div class="text-[10px] text-gray-500 mb-0.5">MEM</div>
                        <div class="text-sm font-mono text-gray-200">{selectedStats.mem_percent?.toFixed(0) ?? "0"}%</div>
                      </div>
                      <div class="bg-white/5 rounded p-2 text-center">
                        <div class="text-[10px] text-gray-500 mb-0.5">NET RX</div>
                        <div class="text-sm font-mono text-gray-200 truncate">{formatBytes(selectedStats.net_rx ?? "0")}</div>
                      </div>
                      <div class="bg-white/5 rounded p-2 text-center">
                        <div class="text-[10px] text-gray-500 mb-0.5">NET TX</div>
                        <div class="text-sm font-mono text-gray-200 truncate">{formatBytes(selectedStats.net_tx ?? "0")}</div>
                      </div>
                    </div>
                  {:else}
                    <div class="text-xs text-gray-500 mb-2">Loading stats...</div>
                  {/if}

                  <!-- Logs toggle -->
                  <button
                    class="text-xs text-gray-400 hover:text-white transition-colors mb-1"
                    onclick={toggleLogs}
                  >
                    {showLogs ? "▼ Hide logs" : "▶ Show logs"}
                  </button>

                  <!-- Log viewer -->
                  {#if showLogs && selectedLogs !== undefined}
                    <div class="bg-black/30 rounded border border-white/5 p-2">
                      <div class="text-[10px] text-gray-500 mb-1">Last 100 lines</div>
                      <pre class="text-[10px] text-gray-400 font-mono whitespace-pre-wrap overflow-x-auto max-h-32 overflow-y-auto">{selectedLogs || "No logs available"}</pre>
                    </div>
                  {/if}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  <!-- Footer: container count -->
  {#if containers.length > 0}
    <div class="flex-none px-4 py-2 border-t border-white/5 text-xs text-gray-500">
      {containers.length} container{containers.length !== 1 ? "s" : ""}
      {#if selectedId}
        · <button class="text-gray-400 hover:text-white underline" onclick={() => { dockerStore.select(null); showLogs = false }}>clear selection</button>
      {/if}
    </div>
  {/if}
</div>