<script lang="ts">
  /**
   * DockerPanel — Right panel tab for container lifecycle management.
   * Shows containers, images, and compose projects with full CRUD support.
   * Dark theme (#191919), Svelte 5 runes.
   */
  import { onMount, onDestroy } from "svelte"
  import { dockerStore, type DockerImage, type ComposeProject, type Container, type ContainerStats } from "../stores/docker.svelte"

  // Active tab: "containers" | "images" | "compose"
  let activeTab = $state<"containers" | "images" | "compose">("containers")

  // Derived state
  let loading = $derived(dockerStore.loading)
  let containers = $derived(dockerStore.filteredContainers)
  let images = $derived(dockerStore.images)
  let projects = $derived(dockerStore.projects)
  let error = $derived(dockerStore.error)
  let searchQuery = $derived(dockerStore.searchQuery)
  let selectedId = $derived(dockerStore.selectedId)
  let stats = $derived(dockerStore.stats)
  let logs = $derived(dockerStore.logs)

  // Create dialog state
  let showCreateDialog = $state(false)
  let createImage = $state("")
  let createName = $state("")
  let createPorts = $state("")
  let createEnvVars = $state("")
  let createVolumes = $state("")
  let createError = $state<string | null>(null)
  let creating = $state(false)

  // Pull dialog state
  let showPullDialog = $state(false)
  let pullImageName = $state("")
  let pulling = $state(false)
  let pullError = $state<string | null>(null)

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
    if (autoRefresh && activeTab === "containers") {
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

  // Switch tab and refresh appropriate data
  function switchTab(tab: "containers" | "images" | "compose") {
    activeTab = tab
    if (tab === "images") {
      dockerStore.listImages()
    } else if (tab === "compose") {
      dockerStore.listProjects()
    }
  }

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
    if (activeTab === "containers") {
      await dockerStore.refresh()
      if (selectedId) {
        await dockerStore.fetchStats(selectedId)
        await dockerStore.fetchLogs(selectedId)
      }
    } else if (activeTab === "images") {
      await dockerStore.listImages()
    } else if (activeTab === "compose") {
      await dockerStore.listProjects()
    }
  }

  function formatBytes(s: string): string {
    if (!s) return "0 B"
    const parts = s.split(/[\s]+/)
    if (parts.length >= 2) return `${parts[0]} ${parts[1]}`
    return s
  }

  // ---- Create container ----
  function openCreateDialog() {
    createImage = ""
    createName = ""
    createPorts = ""
    createEnvVars = ""
    createVolumes = ""
    createError = null
    showCreateDialog = true
  }

  function closeCreateDialog() {
    showCreateDialog = false
    createError = null
  }

  async function handleCreate() {
    if (!createImage.trim()) {
      createError = "Image name is required"
      return
    }
    creating = true
    createError = null

    const ports = createPorts.split(",").map(p => p.trim()).filter(p => p)
    const envVars = createEnvVars.split("\n").map(e => e.trim()).filter(e => e)
    const volumes = createVolumes.split(",").map(v => v.trim()).filter(v => v)

    const result = await dockerStore.create(createImage.trim(), createName.trim(), ports, envVars, volumes)
    creating = false

    if (result.error) {
      createError = result.error
    } else {
      showCreateDialog = false
    }
  }

  // ---- Image management ----
  async function handleRemoveImage(id: string) {
    await dockerStore.removeImage(id, false)
  }

  function openPullDialog() {
    pullImageName = ""
    pullError = null
    showPullDialog = true
  }

  function closePullDialog() {
    showPullDialog = false
    pullError = null
  }

  async function handlePull() {
    if (!pullImageName.trim()) return
    pulling = true
    pullError = null
    const result = await dockerStore.pullImage(pullImageName.trim())
    pulling = false
    if (result.error) {
      pullError = result.error
    } else {
      showPullDialog = false
    }
  }

  // ---- Compose management ----
  async function handleComposeUp(path: string) {
    await dockerStore.composeUp(path)
  }

  async function handleComposeDown(path: string) {
    await dockerStore.composeDown(path, false)
  }

  async function handleComposeStop(path: string) {
    await dockerStore.composeStop(path)
  }

  let selectedStats = $derived(selectedId ? stats.get(selectedId) : null) as ContainerStats | null | undefined
  let selectedLogs = $derived(selectedId ? logs.get(selectedId) : null) as string | null | undefined
</script>

<div class="flex flex-col h-full">
  <!-- Header + Tabs -->
  <div class="flex-none border-b border-white/10">
    <div class="flex items-center justify-between px-4 py-2">
      <h2 class="text-white font-semibold text-base">Docker</h2>
      <div class="flex items-center gap-2">
        {#if activeTab === "containers"}
          <button
            class="text-xs px-2 py-1 rounded bg-purple-600 hover:bg-purple-500 text-white transition-colors"
            onclick={openCreateDialog}
          >
            + Create
          </button>
        {:else if activeTab === "images"}
          <button
            class="text-xs px-2 py-1 rounded bg-purple-600 hover:bg-purple-500 text-white transition-colors"
            onclick={openPullDialog}
          >
            + Pull
          </button>
        {/if}
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

    <!-- Tab bar -->
    <div class="flex px-2 gap-1">
      <button
        class="px-3 py-1.5 text-xs rounded-t transition-colors {activeTab === 'containers' ? 'bg-white/10 text-white border-b-2 border-purple-500' : 'text-gray-400 hover:text-white hover:bg-white/5'}"
        onclick={() => switchTab("containers")}
      >
        Containers
      </button>
      <button
        class="px-3 py-1.5 text-xs rounded-t transition-colors {activeTab === 'images' ? 'bg-white/10 text-white border-b-2 border-purple-500' : 'text-gray-400 hover:text-white hover:bg-white/5'}"
        onclick={() => switchTab("images")}
      >
        Images
      </button>
      <button
        class="px-3 py-1.5 text-xs rounded-t transition-colors {activeTab === 'compose' ? 'bg-white/10 text-white border-b-2 border-purple-500' : 'text-gray-400 hover:text-white hover:bg-white/5'}"
        onclick={() => switchTab("compose")}
      >
        Compose
      </button>
    </div>
  </div>

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if activeTab === "containers"}
    <!-- Search -->
    <div class="flex-none px-4 py-2 border-b border-white/5">
      <input
        type="text"
        bind:value={dockerStore.searchQuery}
        placeholder="Filter by name, image, or state..."
        class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
      />
    </div>

    <!-- Container list -->
    <div class="flex-1 overflow-y-auto">
      {#if loading && containers.length === 0}
        <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading containers...</div>
      {:else if containers.length === 0}
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
                <span class="w-2 h-2 rounded-full shrink-0 {container.state === 'running' ? 'bg-green-400' : container.state === 'paused' ? 'bg-yellow-400' : 'bg-gray-500'}"></span>
                <div class="flex-1 min-w-0">
                  <div class="text-sm text-gray-200 font-medium truncate">{container.name}</div>
                  <div class="text-[10px] text-gray-500 font-mono truncate">{shortId(container.id)} · {container.image}</div>
                </div>
                <span class="px-1.5 py-0.5 rounded text-[10px] font-medium {stateColor(container.state)}">
                  {stateIcon(container.state)} {container.state}
                </span>
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

                  <button
                    class="text-xs text-gray-400 hover:text-white transition-colors mb-1"
                    onclick={toggleLogs}
                  >
                    {showLogs ? "▼ Hide logs" : "▶ Show logs"}
                  </button>

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

    <!-- Footer: container count -->
    {#if containers.length > 0}
      <div class="flex-none px-4 py-2 border-t border-white/5 text-xs text-gray-500">
        {containers.length} container{containers.length !== 1 ? "s" : ""}
        {#if selectedId}
          · <button class="text-gray-400 hover:text-white underline" onclick={() => { dockerStore.select(null); showLogs = false }}>clear selection</button>
        {/if}
      </div>
    {/if}

  {:else if activeTab === "images"}
    <!-- Images tab -->
    <div class="flex-1 overflow-y-auto">
      {#if loading && images.length === 0}
        <div class="px-4 py-3 text-gray-500 text-sm">Loading images...</div>
      {:else if images.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No images found</div>
      {:else}
        <div class="px-3 py-2">
          <table class="w-full text-xs">
            <thead>
              <tr class="text-gray-500 border-b border-white/10">
                <th class="text-left py-1 px-2 font-medium">Repository</th>
                <th class="text-left py-1 px-2 font-medium">Tag</th>
                <th class="text-left py-1 px-2 font-medium">Size</th>
                <th class="text-left py-1 px-2 font-medium">ID</th>
                <th class="text-right py-1 px-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {#each images as img (img.id)}
                <tr class="border-b border-white/5 hover:bg-white/5">
                  <td class="py-1.5 px-2 text-gray-200 truncate max-w-[140px]">{img.repository}</td>
                  <td class="py-1.5 px-2 text-gray-400">{img.tag}</td>
                  <td class="py-1.5 px-2 text-gray-400 font-mono">{img.size}</td>
                  <td class="py-1.5 px-2 text-gray-500 font-mono">{shortId(img.id)}</td>
                  <td class="py-1.5 px-2 text-right">
                    <button
                      class="text-xs px-2 py-0.5 rounded hover:bg-red-500/20 text-red-400 transition-colors"
                      onclick={() => handleRemoveImage(img.id)}
                      title="Remove image"
                    >
                      ✕
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
    {#if images.length > 0}
      <div class="flex-none px-4 py-2 border-t border-white/5 text-xs text-gray-500">
        {images.length} image{images.length !== 1 ? "s" : ""}
      </div>
    {/if}

  {:else if activeTab === "compose"}
    <!-- Compose projects tab -->
    <div class="flex-1 overflow-y-auto">
      {#if loading && projects.length === 0}
        <div class="px-4 py-3 text-gray-500 text-sm">Loading projects...</div>
      {:else if projects.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No compose projects found</div>
      {:else}
        <div class="px-3 py-2">
          {#each projects as proj (proj.name)}
            <div class="bg-white/5 rounded-lg p-3 mb-2">
              <div class="flex items-center justify-between mb-2">
                <div>
                  <div class="text-sm text-gray-200 font-medium">{proj.name}</div>
                  <div class="text-[10px] text-gray-500 font-mono truncate max-w-[200px]">{proj.path}</div>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-[10px] px-2 py-0.5 rounded {proj.status === 'running' ? 'bg-green-500/20 text-green-400' : 'bg-gray-500/20 text-gray-400'}">
                    {proj.status}
                  </span>
                  <button
                    class="w-6 h-6 flex items-center justify-center text-xs rounded hover:bg-green-500/20 text-green-400 transition-colors"
                    onclick={() => handleComposeUp(proj.path)}
                    title="Start"
                  >
                    ▶
                  </button>
                  <button
                    class="w-6 h-6 flex items-center justify-center text-xs rounded hover:bg-yellow-500/20 text-yellow-400 transition-colors"
                    onclick={() => handleComposeStop(proj.path)}
                    title="Stop"
                  >
                    ■
                  </button>
                  <button
                    class="w-6 h-6 flex items-center justify-center text-xs rounded hover:bg-red-500/20 text-red-400 transition-colors"
                    onclick={() => handleComposeDown(proj.path)}
                    title="Down"
                  >
                    ↓
                  </button>
                </div>
              </div>
              {#if proj.services > 0}
                <div class="text-[10px] text-gray-500">{proj.services} service{proj.services !== 1 ? "s" : ""}</div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
    {#if projects.length > 0}
      <div class="flex-none px-4 py-2 border-t border-white/5 text-xs text-gray-500">
        {projects.length} project{projects.length !== 1 ? "s" : ""}
      </div>
    {/if}
  {/if}
</div>

<!-- Create Container Dialog -->
{#if showCreateDialog}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/60 z-50 flex items-center justify-center" onclick={(e) => { if (e.target === e.currentTarget) closeCreateDialog() }}>
    <div class="bg-[#1e1e1e] border border-white/10 rounded-xl w-96 p-5 shadow-2xl">
      <h3 class="text-white font-semibold text-base mb-4">Create Container</h3>

      {#if createError}
        <div class="mb-3 px-3 py-2 bg-red-500/20 border border-red-500/30 rounded text-xs text-red-400">{createError}</div>
      {/if}

      <div class="space-y-3">
        <div>
          <label class="block text-xs text-gray-400 mb-1">Image *</label>
          <input
            type="text"
            bind:value={createImage}
            placeholder="e.g. nginx:latest"
            class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
          />
        </div>
        <div>
          <label class="block text-xs text-gray-400 mb-1">Container Name</label>
          <input
            type="text"
            bind:value={createName}
            placeholder="(auto-generated if empty)"
            class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
          />
        </div>
        <div>
          <label class="block text-xs text-gray-400 mb-1">Port Mappings</label>
          <input
            type="text"
            bind:value={createPorts}
            placeholder="e.g. 8080:80, 443:443"
            class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
          />
        </div>
        <div>
          <label class="block text-xs text-gray-400 mb-1">Environment Variables</label>
          <textarea
            bind:value={createEnvVars}
            placeholder="KEY=value (one per line)"
            rows="3"
            class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50 resize-none"
          ></textarea>
        </div>
        <div>
          <label class="block text-xs text-gray-400 mb-1">Volume Mounts</label>
          <input
            type="text"
            bind:value={createVolumes}
            placeholder="e.g. /data:/data, /home:/root"
            class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
          />
        </div>
      </div>

      <div class="flex items-center justify-end gap-2 mt-5">
        <button
          class="px-4 py-1.5 text-xs rounded hover:bg-white/10 text-gray-400 transition-colors"
          onclick={closeCreateDialog}
        >
          Cancel
        </button>
        <button
          class="px-4 py-1.5 text-xs rounded bg-purple-600 hover:bg-purple-500 text-white transition-colors disabled:opacity-50"
          onclick={handleCreate}
          disabled={creating || !createImage.trim()}
        >
          {creating ? "Creating..." : "Create"}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Pull Image Dialog -->
{#if showPullDialog}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="fixed inset-0 bg-black/60 z-50 flex items-center justify-center" onclick={(e) => { if (e.target === e.currentTarget) closePullDialog() }}>
    <div class="bg-[#1e1e1e] border border-white/10 rounded-xl w-80 p-5 shadow-2xl">
      <h3 class="text-white font-semibold text-base mb-4">Pull Image</h3>

      {#if pullError}
        <div class="mb-3 px-3 py-2 bg-red-500/20 border border-red-500/30 rounded text-xs text-red-400">{pullError}</div>
      {/if}

      <div>
        <label class="block text-xs text-gray-400 mb-1">Image Name</label>
        <input
          type="text"
          bind:value={pullImageName}
          placeholder="e.g. nginx:latest"
          class="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
        />
      </div>

      <div class="flex items-center justify-end gap-2 mt-5">
        <button
          class="px-4 py-1.5 text-xs rounded hover:bg-white/10 text-gray-400 transition-colors"
          onclick={closePullDialog}
        >
          Cancel
        </button>
        <button
          class="px-4 py-1.5 text-xs rounded bg-purple-600 hover:bg-purple-500 text-white transition-colors disabled:opacity-50"
          onclick={handlePull}
          disabled={pulling || !pullImageName.trim()}
        >
          {pulling ? "Pulling..." : "Pull"}
        </button>
      </div>
    </div>
  </div>
{/if}