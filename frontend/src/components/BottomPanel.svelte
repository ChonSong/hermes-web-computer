<script lang="ts">
  import Terminal from "./Terminal.svelte"
  import { ptyOutputs, send } from "../stores/ws"

  // Panel visibility
  let visible = $state(true)
  // Active tab
  type TabId = "terminal" | "problems" | "output" | "ports"
  let activeTab = $state<TabId>("terminal")
  // Terminal instances (multiple tabs)
  interface PtyTab { id: string; title: string; ptyId: string }
  let tabs = $state<PtyTab[]>([
    { id: "term-1", title: "bash", ptyId: "main" }
  ])
  let activeTabId = $state("term-1")

  // Resize state
  let height = $state(240)
  let minHeight = 120
  let maxHeight = 600
  let resizing = $state(false)
  let startY = $state(0)
  let startHeight = $state(0)

  // Keyboard shortcut: Ctrl+` toggle visibility
  function handleGlobalKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key === "`") {
      e.preventDefault()
      visible = !visible
    }
  }

  // Only add listener once on mount
  $effect(() => {
    if (typeof window !== "undefined") {
      window.addEventListener("keydown", handleGlobalKeydown)
    }
  })

  // Drag-to-resize
  function startResize(e: MouseEvent) {
    resizing = true
    startY = e.clientY
    startHeight = height
    e.preventDefault()
  }

  function onMouseMove(e: MouseEvent) {
    if (!resizing) return
    const delta = startY - e.clientY
    height = Math.min(maxHeight, Math.max(minHeight, startHeight + delta))
  }

  function onMouseUp() {
    if (resizing) {
      resizing = false
      // Persist height
      try { localStorage.setItem("hwc-bottom-height", String(height)) } catch {}
    }
  }

  // Restore saved height
  $effect(() => {
    try {
      const saved = localStorage.getItem("hwc-bottom-height")
      if (saved) height = parseInt(saved, 10)
    } catch {}
  })

  // Active tab title computed
  let activePtyId = $derived(tabs.find(t => t.id === activeTabId)?.ptyId ?? "")

  // New terminal tab
  function newTab() {
    const id = "term-" + Date.now()
    tabs.push({ id, title: "bash", ptyId: id })
    activeTabId = id
  }

  // Close tab
  function closeTab(id: string) {
    const idx = tabs.findIndex(t => t.id === id)
    if (tabs.length === 1) return // keep at least one
    tabs.splice(idx, 1)
    if (activeTabId === id) {
      activeTabId = tabs[Math.max(0, idx - 1)].id
    }
  }

  // Middle-click to close tab
  function handleTabMouseDown(e: MouseEvent, id: string) {
    if (e.button === 1) {
      e.preventDefault()
      closeTab(id)
    }
  }

  function setActiveTab(id: string) { activeTabId = id }
  function setTab(tab: TabId) { activeTab = tab }

  // Terminal content for each tab
  function getOutput(ptyId: string): string {
    return $ptyOutputs.get(ptyId) ?? ""
  }
</script>

<svelte:window onmousemove={onMouseMove} onmouseup={onMouseUp} />

<!-- Resize handle at top of panel -->
<div
  class="absolute bottom-0 left-3 right-3 h-[3px] cursor-ns-resize hover:bg-purple-500/50 transition-colors z-50 group {resizing ? 'bg-purple-500/40' : 'bg-transparent'}"
  style="top: -3px;"
  onmousedown={startResize}
  role="separator"
  aria-orientation="horizontal"
  aria-label="Resize bottom panel"
  tabindex="-1"
>
  <!-- Visual handle indicator -->
  <div class="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-8 h-1 rounded-full bg-white/20 group-hover:bg-purple-400/60 transition-colors pointer-events-none"></div>
</div>

{#if visible}
  <div
    class="fixed bottom-3 left-3 right-3 rounded-t-2xl overflow-hidden transition-all duration-200"
    style="height: {height}px; z-index: 30;"
    role="region"
    aria-label="Bottom panel"
  >
    <!-- Glass panel background -->
    <div class="absolute inset-0 bg-[#191919]/95 border border-white/10 rounded-t-2xl backdrop-blur-xl"></div>

    <!-- Tab bar -->
    <div class="relative flex items-center h-9 px-1 gap-0.5 border-b border-white/10 bg-[#12121a]/80" style="border-radius: inherit;">
      <!-- Tabs -->
      <div class="flex items-center gap-0.5 overflow-x-auto flex-1">
        {#each tabs as tab (tab.id)}
          {@const isActive = tab.id === activeTabId && activeTab === "terminal"}
          <button
            class="relative flex items-center gap-1 px-3 h-7 text-xs font-mono rounded-t-md transition-colors whitespace-nowrap shrink-0
              {isActive
                ? 'bg-[#1e1e2e] text-white border-t-2 border-purple-400'
                : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
            onmousedown={(e) => handleTabMouseDown(e, tab.id)}
            onclick={() => { setActiveTab(tab.id); setTab("terminal") }}
            title="{tab.title} (middle-click to close)"
          >
            <!-- Active indicator dot -->
            {#if isActive}
              <span class="w-1.5 h-1.5 rounded-full bg-purple-400 shrink-0"></span>
            {/if}
            <span>{tab.title}</span>
            <span
                class="ml-0.5 text-[10px] text-gray-600 hover:text-gray-300 transition-colors"
                onclick={(e) => { e.stopPropagation(); closeTab(tab.id) }}
                onkeydown={(e) => { if (e.key === "Enter") { e.stopPropagation(); closeTab(tab.id) } }}
                role="button"
                tabindex="-1"
              >×</span>
          </button>
        {/each}

        <!-- Other content tabs -->
        {#each [["problems","⚠️ Problems"],["output","📋 Output"],["ports","🌐 Ports"]] as [id, label]}
          {@const isActive = activeTab === id}
          <button
            class="px-3 h-7 text-xs rounded-t-md transition-colors whitespace-nowrap shrink-0
              {isActive
                ? 'bg-[#1e1e2e] text-white border-t-2 border-purple-400'
                : 'text-gray-400 hover:text-gray-200 hover:bg-white/5'}"
            onclick={() => setTab(id as TabId)}
          >
            {label}
          </button>
        {/each}
      </div>

      <!-- New tab button -->
      <button
        class="w-6 h-6 flex items-center justify-center text-gray-500 hover:text-white transition-colors rounded shrink-0 mr-1"
        onclick={newTab}
        title="New terminal tab"
      >
        +
      </button>

      <!-- Hide button -->
      <button
        class="w-6 h-6 flex items-center justify-center text-gray-500 hover:text-white transition-colors rounded shrink-0 mr-1"
        onclick={() => visible = false}
        title="Hide bottom panel (Ctrl+`)"
      >
        ▼
      </button>
    </div>

    <!-- Tab content area -->
    <div class="relative flex-1 overflow-hidden" style="height: calc(100% - 36px);">
      {#if activeTab === "terminal"}
        <!-- Show terminal for active tab -->
        <div class="absolute inset-0 p-1 overflow-hidden">
          <Terminal ptyId={activePtyId} />
        </div>
      {:else if activeTab === "problems"}
        <div class="absolute inset-0 p-4 text-gray-400 text-sm font-mono overflow-auto">
          <div class="text-gray-500 text-xs">No problems detected</div>
        </div>
      {:else if activeTab === "output"}
        <div class="absolute inset-0 p-4 text-gray-400 text-sm font-mono overflow-auto">
          <div class="text-gray-500 text-xs">Output panel — select a terminal tab above to view logs</div>
        </div>
      {:else if activeTab === "ports"}
        <div class="absolute inset-0 p-4 text-gray-400 text-sm font-mono overflow-auto">
          <div class="text-gray-500 text-xs">No active ports</div>
        </div>
      {/if}
    </div>
  </div>
{:else}
  <!-- Collapsed: thin bar at bottom with show button -->
  <button
    class="fixed bottom-3 left-1/2 -translate-x-1/2 z-30 flex items-center gap-2 px-4 py-1.5
      bg-[#191919]/90 border border-white/10 rounded-full backdrop-blur-xl
      text-gray-400 hover:text-white transition-colors text-xs font-mono shadow-panel"
    onclick={() => visible = true}
    title="Show bottom panel (Ctrl+`)"
  >
    <span>Terminal</span>
    <span class="text-purple-400">Ctrl+`</span>
  </button>
{/if}