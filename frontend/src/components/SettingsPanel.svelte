<script lang="ts">
  /**
   * SettingsPanel — Right panel tab for all app settings and configuration.
   * Sections: General, Appearance, Shortcuts, Connection, Advanced
   */
  import { themeStore, themes, type Theme } from "../stores/theme.svelte"
  import { configStore } from "../stores/config.svelte"
  import { wsState, forceReconnect, send } from "../stores/ws"
  import { onMount } from "svelte"

  let activeSection = $state<string | null>("appearance")

  // --- General state ---
  let fontSize = $state(14)
  let defaultWs = $state(1)

  // --- Appearance state ---
  let showGrain = $state(true)
  let panelOpacity = $state(85)

  // --- Connection state ---
  let wsUrl = $state("ws://localhost:3005/ws")
  let hermesUrl = $state("http://localhost:8787")
  let manualWsUrl = $state("")
  let showWsInput = $state(false)

  // --- Advanced state ---
  let showResetConfirm = $state(false)
  let resetMessage = $state("")

  // Load persisted settings
  onMount(() => {
    try {
      const savedFontSize = localStorage.getItem("hwc-font-size")
      if (savedFontSize) fontSize = parseInt(savedFontSize) || 14
      const savedWs = localStorage.getItem("hwc-default-ws")
      if (savedWs) defaultWs = parseInt(savedWs) || 1
      const savedGrain = localStorage.getItem("hwc-grain")
      if (savedGrain !== null) showGrain = savedGrain === "true"
      const savedOpacity = localStorage.getItem("hwc-panel-opacity")
      if (savedOpacity) panelOpacity = parseInt(savedOpacity) || 85
    } catch { /* noop */ }
  })

  // --- General handlers ---
  function handleFontSize(e: Event) {
    const target = e.target as HTMLInputElement
    fontSize = parseInt(target.value) || 14
    localStorage.setItem("hwc-font-size", String(fontSize))
    document.documentElement.style.fontSize = `${fontSize}px`
  }

  function handleDefaultWs(e: Event) {
    const target = e.target as HTMLSelectElement
    defaultWs = parseInt(target.value) || 1
    localStorage.setItem("hwc-default-ws", String(defaultWs))
  }

  // --- Appearance handlers ---
  function toggleGrain() {
    showGrain = !showGrain
    localStorage.setItem("hwc-grain", String(showGrain))
    const el = document.querySelector(".grain-overlay") as HTMLElement | null
    if (el) el.style.display = showGrain ? "" : "none"
  }

  function handleOpacity(e: Event) {
    const target = e.target as HTMLInputElement
    panelOpacity = parseInt(target.value) || 85
    localStorage.setItem("hwc-panel-opacity", String(panelOpacity))
    document.documentElement.style.setProperty("--panel-opacity", String(panelOpacity / 100))
  }

  // --- Connection handlers ---
  function handleReconnect() {
    const url = manualWsUrl.trim() || wsUrl
    forceReconnect(url)
    showWsInput = false
  }

  function handleCustomWs() {
    const url = manualWsUrl.trim()
    if (url) {
      forceReconnect(url)
      showWsInput = false
    }
  }

  // --- Advanced handlers ---
  function handleResetSettings() {
    localStorage.removeItem("hwc-theme")
    localStorage.removeItem("hwc-font-size")
    localStorage.removeItem("hwc-default-ws")
    localStorage.removeItem("hwc-grain")
    localStorage.removeItem("hwc-panel-opacity")
    localStorage.removeItem("ao-col-widths")
    themeStore.setTheme("illogical-impulse")
    fontSize = 14
    panelOpacity = 85
    showGrain = true
    document.documentElement.style.fontSize = ""
    document.documentElement.style.removeProperty("--panel-opacity")
    resetMessage = "Settings reset to defaults"
    showResetConfirm = false
    setTimeout(() => { resetMessage = "" }, 3000)
  }

  function handleClearCache() {
    try {
      const theme = localStorage.getItem("hwc-theme")
      localStorage.clear()
      if (theme) localStorage.setItem("hwc-theme", theme)
      resetMessage = "Cache cleared (theme preserved)"
      setTimeout(() => { resetMessage = "" }, 3000)
    } catch {
      resetMessage = "Failed to clear cache"
    }
  }

  // Derived connection state
  let wsConnected = $derived($wsState.connected)
  let wsReconnecting = $derived($wsState.reconnecting)
  let wsLastError = $derived($wsState.lastError)
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <h2 class="text-white font-semibold text-base">Settings</h2>
  </div>

  <div class="flex-1 overflow-y-auto px-2 py-2">
    <div class="space-y-1">

      <!-- ===== GENERAL ===== -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left {activeSection === 'general' ? 'bg-white/10' : ''}"
          onclick={() => activeSection = activeSection === "general" ? null : "general"}
        >
          <span class="text-base">⚙️</span>
          <span class="text-gray-200 text-sm">General</span>
        </button>

        {#if activeSection === "general"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Font Size -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Font Size</label>
              <div class="flex items-center gap-2">
                <input
                  type="range" min="12" max="20"
                  class="flex-1 accent-purple-400"
                  value={fontSize}
                  oninput={handleFontSize}
                />
                <span class="text-xs text-gray-300 w-6 text-right">{fontSize}px</span>
              </div>
            </div>

            <!-- Default Workspace -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Default Workspace</label>
              <select
                class="w-full bg-white/10 border border-white/20 rounded px-3 py-2 text-sm text-white"
                value={defaultWs}
                onchange={handleDefaultWs}
              >
                {#each Array(9) as _, i}
                  <option value={i + 1}>Workspace {i + 1}</option>
                {/each}
              </select>
            </div>
          </div>
        {/if}
      </div>

      <!-- ===== APPEARANCE (Theme Picker) ===== -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left {activeSection === 'appearance' ? 'bg-white/10' : ''}"
          onclick={() => activeSection = activeSection === "appearance" ? null : "appearance"}
        >
          <span class="text-base">🎨</span>
          <span class="text-gray-200 text-sm">Appearance</span>
          <span class="ml-auto text-xs text-gray-500">{themeStore.current.name}</span>
        </button>

        {#if activeSection === "appearance"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Theme selector cards -->
            <div>
              <label class="block text-xs text-gray-400 mb-2">Theme</label>
              <div class="grid grid-cols-1 gap-1.5">
                {#each themes as theme (theme.id)}
                  <button
                    class="flex items-center gap-3 px-3 py-2 rounded-lg border transition-all text-left
                           {themeStore.currentId === theme.id
                             ? 'border-purple-500/50 bg-purple-500/10 ring-1 ring-purple-500/20'
                             : 'border-white/10 hover:border-white/20 hover:bg-white/5'}"
                    onclick={() => themeStore.setTheme(theme.id)}
                  >
                    <!-- Color swatch row -->
                    <div class="flex gap-0.5 shrink-0">
                      <span class="w-4 h-4 rounded-sm" style="background:{theme.colors.bg}"></span>
                      <span class="w-4 h-4 rounded-sm" style="background:{theme.colors.surface}"></span>
                      <span class="w-4 h-4 rounded-sm" style="background:{theme.colors.accent}"></span>
                      <span class="w-4 h-4 rounded-sm" style="background:{theme.colors.primary}"></span>
                    </div>
                    <div class="min-w-0">
                      <div class="text-sm text-gray-200 truncate">{theme.name}</div>
                      <div class="text-xs text-gray-500 truncate">{theme.description}</div>
                    </div>
                    {#if themeStore.currentId === theme.id}
                      <span class="ml-auto text-purple-400 text-sm">✓</span>
                    {/if}
                  </button>
                {/each}
              </div>
            </div>

            <!-- Background grain toggle -->
            <div class="flex items-center justify-between">
              <span class="text-xs text-gray-400">Background Grain</span>
              <button
                class="w-9 h-5 rounded-full transition-colors relative {showGrain ? 'bg-purple-500' : 'bg-white/20'}"
                onclick={toggleGrain}
                role="switch"
                aria-checked={showGrain}
              >
                <span class="absolute top-0.5 w-4 h-4 rounded-full bg-white transition-transform {showGrain ? 'translate-x-4' : 'translate-x-0.5'}"></span>
              </button>
            </div>

            <!-- Panel opacity slider -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Panel Opacity</label>
              <div class="flex items-center gap-2">
                <input
                  type="range" min="60" max="100"
                  class="flex-1 accent-purple-400"
                  value={panelOpacity}
                  oninput={handleOpacity}
                />
                <span class="text-xs text-gray-300 w-8 text-right">{panelOpacity}%</span>
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- ===== SHORTCUTS ===== -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left {activeSection === 'shortcuts' ? 'bg-white/10' : ''}"
          onclick={() => activeSection = activeSection === "shortcuts" ? null : "shortcuts"}
        >
          <span class="text-base">⌨️</span>
          <span class="text-gray-200 text-sm">Shortcuts</span>
        </button>

        {#if activeSection === "shortcuts"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg">
            <!-- Layout shortcuts -->
            <div class="mb-3">
              <div class="text-xs text-gray-400 uppercase tracking-wider mb-1.5">Layout Controls</div>
              <div class="space-y-1">
                {#each [
                  { keys: "Shift+1-9", label: "Switch workspace" },
                  { keys: "Shift+Arrow", label: "Focus adjacent tile" },
                  { keys: "Shift+Alt+Arrow", label: "Resize tile" },
                  { keys: "Shift+D", label: "Cycle layout mode" },
                  { keys: "Shift+F", label: "Toggle fullscreen" },
                  { keys: "Shift+Q", label: "Close focused tile" },
                  { keys: "Shift+Space", label: "Toggle float tile" },
                ] as shortcut}
                  <div class="flex items-center justify-between py-1 px-2 rounded hover:bg-white/5">
                    <span class="text-xs text-gray-400">{shortcut.label}</span>
                    <kbd class="text-[10px] font-mono bg-white/10 text-gray-300 px-1.5 py-0.5 rounded">{shortcut.keys}</kbd>
                  </div>
                {/each}
              </div>
            </div>

            <!-- Panel shortcuts -->
            <div class="mb-3">
              <div class="text-xs text-gray-400 uppercase tracking-wider mb-1.5">Panels</div>
              <div class="space-y-1">
                {#each [
                  { keys: "Ctrl+B", label: "Toggle left panel" },
                  { keys: "Ctrl+Shift+B", label: "Toggle right panel" },
                  { keys: "Ctrl+`", label: "Toggle bottom terminal" },
                  { keys: "Ctrl+K", label: "Command palette" },
                  { keys: "Ctrl+?", label: "Show keymap overlay" },
                ] as shortcut}
                  <div class="flex items-center justify-between py-1 px-2 rounded hover:bg-white/5">
                    <span class="text-xs text-gray-400">{shortcut.label}</span>
                    <kbd class="text-[10px] font-mono bg-white/10 text-gray-300 px-1.5 py-0.5 rounded">{shortcut.keys}</kbd>
                  </div>
                {/each}
              </div>
            </div>

            <!-- Menu bar shortcuts -->
            <div>
              <div class="text-xs text-gray-400 uppercase tracking-wider mb-1.5">Menu Bar</div>
              <div class="space-y-1">
                {#each [
                  { keys: "Alt+F", label: "File menu" },
                  { keys: "Alt+E", label: "Edit menu" },
                  { keys: "Alt+V", label: "View menu" },
                  { keys: "Alt+G", label: "Go menu" },
                  { keys: "Alt+T", label: "Terminal menu" },
                  { keys: "Alt+H", label: "Help menu" },
                ] as shortcut}
                  <div class="flex items-center justify-between py-1 px-2 rounded hover:bg-white/5">
                    <span class="text-xs text-gray-400">{shortcut.label}</span>
                    <kbd class="text-[10px] font-mono bg-white/10 text-gray-300 px-1.5 py-0.5 rounded">{shortcut.keys}</kbd>
                  </div>
                {/each}
              </div>
            </div>
          </div>
        {/if}
      </div>

      <!-- ===== CONNECTION ===== -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left {activeSection === 'connection' ? 'bg-white/10' : ''}"
          onclick={() => activeSection = activeSection === "connection" ? null : "connection"}
        >
          <span class="text-base">🔌</span>
          <span class="text-gray-200 text-sm">Connection</span>
          <span class="ml-auto flex items-center gap-1.5">
            <span class="w-1.5 h-1.5 rounded-full {wsConnected ? 'bg-green-400' : wsReconnecting ? 'bg-yellow-400' : 'bg-red-400'}"></span>
            <span class="text-xs {wsConnected ? 'text-green-400' : wsReconnecting ? 'text-yellow-400' : 'text-red-400'}">
              {wsConnected ? "Connected" : wsReconnecting ? "Reconnecting" : "Disconnected"}
            </span>
          </span>
        </button>

        {#if activeSection === "connection"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Connection status -->
            <div class="flex items-center justify-between">
              <span class="text-xs text-gray-400">Status</span>
              <span class="text-xs {wsConnected ? 'text-green-400' : wsReconnecting ? 'text-yellow-400' : 'text-red-400'}">
                {#if wsConnected}
                  Connected
                {:else if wsReconnecting}
                  Reconnecting ({$wsState.retryCount}/10)…
                {:else}
                  {$wsState.lastError || "Disconnected"}
                {/if}
              </span>
            </div>

            <!-- WebSocket URL -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">WebSocket URL</label>
              <div class="text-xs text-gray-500 font-mono bg-black/20 rounded px-2 py-1.5 break-all">{wsUrl}</div>
            </div>

            <!-- Hermes API URL -->
            <div>
              <label class="block text-xs text-gray-400 mb-1">Hermes API URL</label>
              <div class="text-xs text-gray-500 font-mono bg-black/20 rounded px-2 py-1.5 break-all">{hermesUrl}</div>
            </div>

            <!-- Reconnect button -->
            <div class="flex gap-2">
              <button
                class="flex-1 px-3 py-2 text-xs rounded bg-purple-500/20 text-purple-400 hover:bg-purple-500/30 transition-colors"
                onclick={handleReconnect}
              >
                Reconnect
              </button>
              <button
                class="px-3 py-2 text-xs rounded bg-white/10 text-gray-400 hover:bg-white/20 transition-colors"
                onclick={() => showWsInput = !showWsInput}
              >
                Custom WS →
              </button>
            </div>

            {#if showWsInput}
              <div class="flex gap-2">
                <input
                  type="text"
                  placeholder="ws://host:port/ws"
                  class="flex-1 bg-white/10 border border-white/20 rounded px-2 py-1.5 text-xs text-white placeholder-gray-500 font-mono"
                  bind:value={manualWsUrl}
                  onkeydown={(e) => { if (e.key === 'Enter') handleCustomWs() }}
                />
                <button
                  class="shrink-0 px-3 py-1.5 text-xs rounded bg-green-500/20 text-green-400 hover:bg-green-500/30"
                  onclick={handleCustomWs}
                >
                  Connect
                </button>
              </div>
            {/if}
          </div>
        {/if}
      </div>

      <!-- ===== ADVANCED ===== -->
      <div>
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left {activeSection === 'advanced' ? 'bg-white/10' : ''}"
          onclick={() => activeSection = activeSection === "advanced" ? null : "advanced"}
        >
          <span class="text-base">🔧</span>
          <span class="text-gray-200 text-sm">Advanced</span>
        </button>

        {#if activeSection === "advanced"}
          <div class="mt-2 px-4 py-3 bg-white/5 rounded-lg space-y-3">
            <!-- Version -->
            <div class="flex items-center justify-between">
              <span class="text-xs text-gray-400">Version</span>
              <span class="text-xs text-gray-500 font-mono">v1.4.0</span>
            </div>

            <!-- Server info -->
            <div class="flex items-center justify-between">
              <span class="text-xs text-gray-400">Backend</span>
              <span class="text-xs text-gray-500 font-mono">Go + Svelte 5</span>
            </div>

            <!-- Clear local cache -->
            <button
              class="w-full px-3 py-2 text-xs rounded bg-white/10 text-gray-400 hover:bg-white/20 transition-colors text-left"
              onclick={handleClearCache}
            >
              Clear Local Cache (preserves theme)
            </button>

            <!-- Reset all settings -->
            {#if !showResetConfirm}
              <button
                class="w-full px-3 py-2 text-xs rounded bg-red-500/10 text-red-400 hover:bg-red-500/20 transition-colors text-left"
                onclick={() => showResetConfirm = true}
              >
                Reset All Settings to Defaults
              </button>
            {:else}
              <div class="space-y-2">
                <div class="text-xs text-red-400">Are you sure? This will reset theme, font size, opacity, and panel widths.</div>
                <div class="flex gap-2">
                  <button
                    class="flex-1 px-3 py-2 text-xs rounded bg-red-500/20 text-red-400 hover:bg-red-500/30"
                    onclick={handleResetSettings}
                  >
                    Yes, Reset
                  </button>
                  <button
                    class="flex-1 px-3 py-2 text-xs rounded bg-white/10 text-gray-400 hover:bg-white/20"
                    onclick={() => showResetConfirm = false}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            {/if}

            {#if resetMessage}
              <div class="text-xs text-green-400 text-center py-1">{resetMessage}</div>
            {/if}
          </div>
        {/if}
      </div>

    </div>
  </div>
</div>
