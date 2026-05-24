<script lang="ts">
  import { appsLaunch, send, on } from "../stores/ws"
  import { layout } from "../stores/ws"

  interface DockItem {
    id: string
    label: string
    emoji: string
    type: string
    isPanelFeature?: boolean  // Features that are always-on panels, not tiles
  }

  const dockItems: DockItem[] = [
    { id: "files", label: "Files", emoji: "📁", type: "file-manager", isPanelFeature: true },
    { id: "terminal", label: "Terminal", emoji: "🖥️", type: "terminal" },
    { id: "agent", label: "Agent", emoji: "💬", type: "agent", isPanelFeature: true },
    { id: "browser", label: "Browser", emoji: "🌐", type: "browser" },
    { id: "dashboard", label: "Dashboard", emoji: "📊", type: "dashboard", isPanelFeature: true },
    { id: "voice", label: "Voice", emoji: "🎤", type: "audio", isPanelFeature: true },
    { id: "profiles", label: "Profiles", emoji: "👤", type: "profiles", isPanelFeature: true },
    { id: "skills", label: "Skills", emoji: "◆", type: "skills", isPanelFeature: true },
    { id: "crons", label: "Crons", emoji: "⏰", type: "crons", isPanelFeature: true },
    { id: "memory", label: "Memory", emoji: "🧠", type: "memory", isPanelFeature: true },
  ]

  let hoveredApp = $state<string | null>(null)
  let contextMenu = $state<{ app: DockItem; x: number; y: number } | null>(null)

  // Track running apps: app.id -> Set of tile IDs
  const runningApps = new Map<string, Set<string>>()
  // Track pinned apps
  const pinnedApps = new Set<string>(["terminal", "browser"])

  // Map of browser_id -> browser tile node id for Browser tiles
  const browserTiles = new Map<string, string>()

  // Subscribe to layout changes to detect which tiles are running
  const unsubscribeLayout = layout.subscribe(({ tree }) => {
    if (!tree) return
    const tileTypes = new Set<string>()
    collectTileTypes(tree, tileTypes)
    // Update running state for each dock item
    for (const item of dockItems) {
      if (!item.isPanelFeature && tileTypes.has(item.type)) {
        addRunningApp(item.id)
      }
    }
    // Clean up apps that are no longer in the layout
    for (const [appId, tileIds] of runningApps.entries()) {
      if (tileIds.size === 0) {
        runningApps.delete(appId)
      }
    }
  })

  function collectTileTypes(node: { type: string; content?: string; children?: { type: string; content?: string }[] }, set: Set<string>) {
    if (node.content) set.add(node.content)
    if (node.children) {
      for (const child of node.children) {
        collectTileTypes(child, set)
      }
    }
  }

  function addRunningApp(appId: string, tileId?: string) {
    if (!runningApps.has(appId)) {
      runningApps.set(appId, new Set())
    }
    if (tileId) {
      runningApps.get(appId)!.add(tileId)
    }
  }

  function removeRunningApp(appId: string, tileId?: string) {
    const tiles = runningApps.get(appId)
    if (!tiles) return
    if (tileId) {
      tiles.delete(tileId)
    }
    if (!tileId || tiles.size === 0) {
      runningApps.delete(appId)
    }
  }

  function isAppRunning(appId: string): boolean {
    return runningApps.has(appId) && (runningApps.get(appId)?.size ?? 0) > 0
  }

  function isAppPinned(appId: string): boolean {
    return pinnedApps.has(appId)
  }

  function getRunningTileIds(appId: string): Set<string> {
    return runningApps.get(appId) ?? new Set()
  }

  // Listen for browser launch responses to add the tile with browser_id
  on("apps.launch.response", (data: unknown) => {
    const resp = data as { type?: string; browser_id?: string; pty_id?: string; note?: string }
    console.log('[Dock] apps.launch.response received:', resp)

    if (resp.type === "browser" && resp.browser_id) {
      const tileId = `browser_${resp.browser_id}`
      browserTiles.set(resp.browser_id, tileId)
      addRunningApp("browser", tileId)
      console.log('[Dock] Creating browser tile with id:', tileId)

      send({ protocol: "ui", method: "layout.update", params: {
        op: "split",
        target_id: "root",
        direction: "h",
        content: "browser",
        browser_id: resp.browser_id,
      }})
      console.log('[Dock] layout.update sent for browser tile')
    }
  })

  function handleLaunch(item: DockItem, e?: MouseEvent) {
    // Middle-click: always launch new instance
    if (e && e.button === 1) {
      e.preventDefault()
      launchNewInstance(item)
      return
    }

    if (item.isPanelFeature) {
      window.dispatchEvent(new CustomEvent('hwc-dock-panel', { detail: { panel: item.type } }))
      return
    }

    // If running, focus existing tile; otherwise launch
    if (isAppRunning(item.id)) {
      focusTile(item.id)
    } else {
      launchNewInstance(item)
    }
  }

  function launchNewInstance(item: DockItem) {
    if (item.type === "browser") {
      appsLaunch(item.type)
    } else {
      const tileContent: Record<string, string> = {
        "terminal": "xterm",
        "editor": "editor",
        "preview": "preview",
      }
      const content = tileContent[item.type]
      if (content) {
        send({ protocol: "ui", method: "layout.update", params: {
          op: "split",
          target_id: "root",
          direction: "h",
          content,
        }})
      }
    }
    addRunningApp(item.id)
  }

  function focusTile(appId: string) {
    // Find the first tile of this type and focus it
    const tileIds = getRunningTileIds(appId)
    if (tileIds.size > 0) {
      const firstTileId = Array.from(tileIds)[0]
      send({ protocol: "ui", method: "layout.focus", params: { tile_id: firstTileId } })
    }
  }

  function handleContextMenu(item: DockItem, e: MouseEvent) {
    e.preventDefault()
    contextMenu = { app: item, x: e.clientX, y: e.clientY }
  }

  function closeContextMenu() {
    contextMenu = null
  }

  function togglePin(item: DockItem) {
    if (pinnedApps.has(item.id)) {
      pinnedApps.delete(item.id)
    } else {
      pinnedApps.add(item.id)
    }
    closeContextMenu()
    // Force reactivity by reassigning
  }

  function handleClickOutside(e: MouseEvent) {
    if (contextMenu) {
      closeContextMenu()
    }
  }

  // Clean up subscription on unmount
  $effect(() => {
    return () => {
      unsubscribeLayout()
    }
  })
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="fixed bottom-3 left-1/2 -translate-x-1/2 z-50
  bg-[#191919] border border-white/10 rounded-full
  px-4 py-2 flex items-center gap-3 shadow-panel
  animate-fade-in"
  onclick={handleClickOutside}
  oncontextmenu={(e) => e.preventDefault()}>

  {#each dockItems as item}
    {@const running = isAppRunning(item.id)}
    {@const pinned = isAppPinned(item.id)}
    <div class="relative flex flex-col items-center">
      <!-- svelte-ignore a11y_click_events_have_key_events -->
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <button
        class="w-10 h-10 flex items-center justify-center rounded-full
          text-lg transition-all duration-150
          {hoveredApp === item.id ? 'scale-110' : 'scale-100'}
          {running ? 'bg-purple-500/20 hover:bg-purple-500/30' : 'hover:bg-white/10'}"
        onclick={(e) => handleLaunch(item, e)}
        oncontextmenu={(e) => handleContextMenu(item, e)}
        onmouseenter={() => hoveredApp = item.id}
        onmouseleave={() => hoveredApp = null}
        aria-label={item.label}
        title={item.label}
      >
        {item.emoji}
      </button>
      <!-- Running indicator dot (purple) -->
      {#if running}
        <div class="w-1.5 h-1.5 rounded-full bg-purple-400 mt-0.5"></div>
      {:else if pinned}
        <!-- Pinned but not running: small white dot -->
        <div class="w-1 h-1 rounded-full bg-white/40 mt-0.5"></div>
      {/if}
    </div>
  {/each}
</div>

<!-- Context Menu -->
{#if contextMenu}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed z-[200] bg-[#1e1e1e] border border-white/15 rounded-lg shadow-xl py-1 min-w-[160px]"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px"
    onclick={(e) => e.stopPropagation()}
    oncontextmenu={(e) => e.preventDefault()}
  >
    <div class="px-3 py-1.5 text-xs text-white/40 border-b border-white/10">
      {contextMenu.app.emoji} {contextMenu.app.label}
    </div>
    <button
      class="w-full text-left px-3 py-1.5 text-sm text-white/90 hover:bg-white/10 flex items-center gap-2"
      onclick={() => launchNewInstance(contextMenu!.app)}
    >
      {#if isAppRunning(contextMenu!.app.id)}
        Focus {contextMenu!.app.label}
      {:else}
        Launch {contextMenu!.app.label}
      {/if}
    </button>
    <button
      class="w-full text-left px-3 py-1.5 text-sm text-white/90 hover:bg-white/10 flex items-center gap-2"
      onclick={() => { launchNewInstance(contextMenu!.app); closeContextMenu() }}
    >
      New Instance
    </button>
    <div class="border-t border-white/10 my-1"></div>
    <button
      class="w-full text-left px-3 py-1.5 text-sm {isAppPinned(contextMenu!.app.id) ? 'text-purple-400' : 'text-white/90'} hover:bg-white/10 flex items-center gap-2"
      onclick={() => togglePin(contextMenu!.app)}
    >
      {isAppPinned(contextMenu!.app.id) ? '✓' : '○'} {isAppPinned(contextMenu!.app.id) ? 'Unpin' : 'Pin'} App
    </button>
  </div>
{/if}