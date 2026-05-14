<script lang="ts">
  import { appsLaunch, send, on } from "../stores/ws"

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

  let activeApp = $state<string | null>(null)
  let hoveredApp = $state<string | null>(null)

  // Map of browser_id -> browser tile node id for Browser tiles
  const browserTiles = new Map<string, string>()

  // Listen for browser launch responses to add the tile with browser_id
  on("apps.launch.response", (data: unknown) => {
    const resp = data as { type?: string; browser_id?: string; pty_id?: string; note?: string }
    console.log('[Dock] apps.launch.response received:', resp)
    console.log('[Dock] Current globalThis.__wsEvents count:', (globalThis as typeof globalThis & { __wsEvents?: unknown[] }).__wsEvents?.length || 0)

    if (resp.type === "browser" && resp.browser_id) {
      // Create a browser tile with the browser_id
      const tileId = `browser_${resp.browser_id}`
      browserTiles.set(resp.browser_id, tileId)
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

  // Also listen for layout delta to verify our tile was added
  on("layout.delta", (data: unknown) => {
    console.log('[Dock] layout.delta received, ops:', JSON.stringify(data))
  })

  function handleLaunch(item: DockItem) {
    if (item.isPanelFeature) {
      // Dispatch custom event for right panel tab switching
      window.dispatchEvent(new CustomEvent('hwc-dock-panel', { detail: { panel: item.type } }))
      activeApp = activeApp === item.id ? null : item.id
      return
    }

    // For tile-based apps, we need to:
    // 1. Ask backend to create the app instance (for browser and terminal)
    // 2. Add a tile to the layout tree
    if (item.type === "browser") {
      // Browser needs backend to create a browser instance first
      appsLaunch(item.type)
    } else {
      // Other tile types can be directly added
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

    activeApp = item.id
  }

  
</script>

<div class="fixed bottom-3 left-1/2 -translate-x-1/2 z-50
  backdrop-blur-2xl bg-[#12121a]/70 border border-white/10 rounded-full
  px-4 py-2 flex items-center gap-3 shadow-panel
  animate-fade-in">
  {#each dockItems as item}
    <div class="relative flex flex-col items-center">
      <button
        class="w-10 h-10 flex items-center justify-center rounded-full
          text-lg transition-all duration-150
          {hoveredApp === item.id ? 'scale-110' : 'scale-100'}
          {activeApp === item.id ? 'bg-purple-500/20 hover:bg-purple-500/30' : 'hover:bg-white/10'}"
        onclick={() => {
          if (item.isPanelFeature) {
            handleDockPanelClick(item)
          } else {
            handleLaunch(item)
          }
        }}
        onmouseenter={() => hoveredApp = item.id}
        onmouseleave={() => hoveredApp = null}
        aria-label={item.label}
        title={item.label}
      >
        {item.emoji}
      </button>
      <!-- Active indicator dot -->
      {#if activeApp === item.id}
        <div class="w-1 h-1 rounded-full bg-purple-400 mt-0.5"></div>
      {/if}
    </div>
  {/each}
</div>
