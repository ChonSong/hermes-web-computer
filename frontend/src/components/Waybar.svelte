/**
 * Waybar.svelte — Top bar with workspace indicators + system tray
 * Replaces WorkspacePill. Spec: docs/WAYBAR-SPEC.md §2, §4
 */
<script lang="ts">
  import { onMount } from "svelte"
  import { workspaceStore, setActiveWorkspace } from "../stores/workspace"
  import { on } from "../stores/ws"

  const workspaceCount = 9

  // --- Clock ---
  function formatTime(): string {
    const now = new Date()
    return now.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", hour12: false })
  }

  let currentTime = $state(formatTime())
  $effect(() => {
    const interval = setInterval(() => { currentTime = formatTime() }, 10_000)
    return () => clearInterval(interval)
  })

  // --- Active workspace from store ---
  let activeWorkspace = $derived($workspaceStore.activeWorkspace)

  // --- Window title from focused tile ---
  let windowTitle = $state("")
  let focusedTileId = $state("")

  onMount(() => {
    const cleanup = on("ui.focus.changed", (data: unknown) => {
      const d = data as { tile_id?: string; title?: string } | null
      if (d?.tile_id) focusedTileId = d.tile_id
      if (d?.title) windowTitle = d.title
    })
    return cleanup
  })

  // --- Agent status ---
  type AgentStatus = "idle" | "processing" | "error"
  let agentStatus = $state<AgentStatus>("idle")

  onMount(() => {
    const cleanup = on("agent.stream_start", () => { agentStatus = "processing" })
    const cleanup2 = on("agent.stream_end", () => { agentStatus = "idle" })
    const cleanup3 = on("agent.error", () => { agentStatus = "error" })
    return () => { cleanup(); cleanup2(); cleanup3() }
  })

  // --- System tray metrics (polled from global metrics store) ---
  let cpuPercent = $state(0)
  let memPercent = $state(0)
  let tempCelsius = $state(0)
  let volumeLevel = $state(100) // 0-100, 0=muted
  let wifiConnected = $state(true)
  let batteryPercent = $state(100)
  let batteryCharging = $state(false)

  onMount(() => {
    // Poll metrics from the global window object (set by backend via WS event)
    const updateMetrics = () => {
      const win = globalThis as typeof globalThis & {
        __hwcMetrics?: {
          cpu?: number; mem?: number; temp?: number;
          volume?: number; wifi?: boolean; battery?: number; charging?: boolean
        }
      }
      const m = win.__hwcMetrics
      if (m) {
        if (m.cpu !== undefined) cpuPercent = m.cpu
        if (m.mem !== undefined) memPercent = m.mem
        if (m.temp !== undefined) tempCelsius = m.temp
        if (m.volume !== undefined) volumeLevel = m.volume
        if (m.wifi !== undefined) wifiConnected = m.wifi
        if (m.battery !== undefined) batteryPercent = m.battery
        if (m.charging !== undefined) batteryCharging = m.charging
      }
    }

    const interval = setInterval(updateMetrics, 5000)
    updateMetrics()
    return () => clearInterval(interval)
  })

  // Subscribe to system.metrics WS events for real-time updates
  onMount(() => {
    const cleanup = on("system.metrics", (data: unknown) => {
      const m = data as {
        cpu?: { percent?: number }; memory?: { used_percent?: number }
        temperature?: { celsius?: number }; audio?: { icon?: string }
      } | null
      if (m?.cpu?.percent !== undefined) cpuPercent = m.cpu.percent
      if (m?.memory?.used_percent !== undefined) memPercent = m.memory.used_percent
      if (m?.temperature?.celsius !== undefined) tempCelsius = m.temperature.celsius
    })
    return cleanup
  })

  // --- Workspace dot click ---
  function handleWsClick(ws: number) {
    setActiveWorkspace(ws)
  }

  // --- Occupied workspaces (tiles present in each workspace) ---
  // We track this by listening to layout events
  let occupiedWorkspaces = $state<Set<number>>(new Set([1]))

  onMount(() => {
    const cleanup = on("layout.delta", (data: unknown) => {
      // Re-scan layout tree for occupied workspaces
      // For now, mark any workspace with a layout tree as occupied
      const d = data as { workspace?: number } | null
      if (d?.workspace) {
        occupiedWorkspaces = new Set([...occupiedWorkspaces, d.workspace])
      }
    })
    return cleanup
  })

  // --- Dot appearance helper ---
  function dotClass(ws: number): string {
    const isActive = ws === activeWorkspace
    const isOccupied = occupiedWorkspaces.has(ws)
    if (isActive) return "bg-purple-400 shadow-[0_0_8px_rgba(167,139,250,0.6)]"
    if (isOccupied) return "bg-white/30"
    return "bg-white/15"
  }
</script>

<!-- Waybar: full-width top bar with workspace pill centered + system tray right -->
<div class="fixed top-0 left-0 right-0 z-[1000] flex items-center justify-between px-4 h-9 pointer-events-none">
  <!-- Left spacer (transparent, for balance) -->
  <div class="w-32"></div>

  <!-- Workspace Pill — centered -->
  <div class="pointer-events-auto flex items-center gap-3 px-5 py-1
    backdrop-blur-xl border border-white/10 rounded-full
    shadow-[0_4px_16px_rgba(0,0,0,0.3)]
    bg-[rgba(18,18,26,0.85)]">
    <!-- 9 workspace dots -->
    {#each Array.from({ length: workspaceCount }, (_, i) => i + 1) as ws}
      <button
        class="w-2.5 h-2.5 rounded-full transition-all duration-150 hover:scale-125 {dotClass(ws)}"
        onclick={() => handleWsClick(ws)}
        aria-label="Workspace {ws}"
        title="Workspace {ws}"
      ></button>
    {/each}

    <!-- Separator -->
    <div class="w-px h-4 bg-white/10"></div>

    <!-- Window title (from focused tile) -->
    {#if windowTitle}
      <span class="text-[10px] text-white/50 font-mono max-w-[120px] truncate hidden sm:block">
        {windowTitle}
      </span>
      <div class="w-px h-4 bg-white/10"></div>
    {/if}

    <!-- Clock -->
    <span class="text-[11px] text-white/60 font-mono tabular-nums">{currentTime}</span>

    <!-- Separator -->
    <div class="w-px h-4 bg-white/10"></div>

    <!-- Agent status dot -->
    {#if agentStatus === "idle"}
      <div class="w-1.5 h-1.5 rounded-full bg-gray-400"></div>
    {:else if agentStatus === "processing"}
      <div class="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse"></div>
    {:else if agentStatus === "error"}
      <div class="w-1.5 h-1.5 rounded-full bg-red-400"></div>
    {/if}
  </div>

  <!-- System Tray — right side -->
  <div class="pointer-events-auto flex items-center gap-1 px-3 py-1
    backdrop-blur-xl border border-white/10 rounded-full
    shadow-[0_4px_16px_rgba(0,0,0,0.3)]
    bg-[rgba(18,18,26,0.85)]">

    <!-- Wifi -->
    <span class="text-xs" title="WiFi: {wifiConnected ? 'Connected' : 'Disconnected'}">
      {wifiConnected ? "🌐" : "📡"}
    </span>

    <!-- Volume -->
    <span class="text-xs" title="Volume: {volumeLevel}%">
      {volumeLevel === 0 ? "🔇" : volumeLevel < 50 ? "🔉" : "🔊"}
    </span>

    <!-- Battery -->
    <span class="text-xs" title="Battery: {batteryPercent}%{batteryCharging ? ' (charging)' : ''}">
      {batteryCharging ? "⚡" : batteryPercent < 20 ? "🪫" : "🔋"}
    </span>

    <!-- Temperature -->
    <span class="text-xs" title="CPU: {tempCelsius.toFixed(0)}°C">
      {tempCelsius > 80 ? "🔥" : tempCelsius > 60 ? "🌡️" : "🌡️"}
    </span>

    <!-- Separator -->
    <div class="w-px h-3 bg-white/10 mx-1"></div>

    <!-- Time (same as clock in pill) -->
    <span class="text-[11px] text-white/60 font-mono tabular-nums">{currentTime}</span>
  </div>
</div>