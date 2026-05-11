<script lang="ts">
  /**
   * DashOverview — Migrated from agent-os DashboardPage.tsx
   * Aggregated metrics overview with KPI cards, session activity,
   * model mix, and quick access to system health.
   */
  import { onMount } from "svelte"
  import { send, on, ws } from "../stores/ws"

  interface SessionEntry {
    id: string
    session_key: string
    created_at: string
    message_count: number
    total_chars: number
  }

  interface EventBreakdown {
    type: string
    count: number
  }

  interface ContainerInfo {
    Id: string
    Names: string[]
    State: string
  }

  interface ContainerStats {
    cpu_percent: number
    memory_percent: number
  }

  let analytics = $state<{
    sessions: SessionEntry[]
    event_breakdown: EventBreakdown[]
  } | null>(null)

  let usage = $state<{
    total_sessions: number
    total_messages: number
    total_tokens: number
    avg_tokens_per_session: number
    sessions_last_7d: number
  } | null>(null)

  let status = $state<{
    gateway_running?: boolean
    version?: string
    started_at?: number
  } | null>(null)

  let loading = $state(true)
  let containers = $state<ContainerInfo[]>([])
  let containerStats = $state<Record<string, ContainerStats>>({})

  async function loadData() {
    loading = true
    try {
      // Fetch analytics from backend API
      const [analyticsRes, statusRes] = await Promise.allSettled([
        fetch("/api/analytics/real").then(r => r.json()),
        fetch("/api/status").then(r => r.json()),
      ])

      if (analyticsRes.status === "fulfilled") {
        analytics = analyticsRes.value
      }

      if (statusRes.status === "fulfilled") {
        status = statusRes.value
      }

      // Also request from WS
      send({ protocol: "ui", method: "dashboard.stats" })
    } catch { /* handled by Promise.allSettled */ }
    loading = false
  }

  function formatUptime(seconds?: number): string {
    if (!seconds) return "—"
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return `${h}h ${m}m`
  }

  function formatTokens(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
    return n.toString()
  }

  function computedUptime(): string {
    if (!status?.started_at) return "—"
    const seconds = Math.floor((Date.now() - status.started_at * 1000) / 1000)
    return formatUptime(seconds)
  }

  const kpis = $derived([
    {
      label: "Total Sessions",
      value: usage?.total_sessions ?? analytics?.sessions?.length ?? "—",
      icon: "MessageSquare",
      trend: usage?.sessions_last_7d ? `${usage.sessions_last_7d} this week` : undefined,
      color: "bg-blue-500/10 text-blue-400",
    },
    {
      label: "Total Messages",
      value: usage?.total_messages ?? "—",
      icon: "Brain",
      color: "bg-purple-500/10 text-purple-400",
    },
    {
      label: "Total Tokens",
      value: usage?.total_tokens ? formatTokens(usage.total_tokens) : "—",
      icon: "Zap",
      trend: usage?.avg_tokens_per_session ? `~${formatTokens(usage.avg_tokens_per_session)}/session` : undefined,
      color: "bg-amber-500/10 text-amber-400",
    },
    {
      label: "Uptime",
      value: computedUptime(),
      icon: "Clock",
      color: "bg-green-500/10 text-green-400",
    },
    {
      label: "Containers",
      value: containers.length,
      icon: "Server",
      trend: `${containers.filter(c => c.State === "running").length} running`,
      color: "bg-cyan-500/10 text-cyan-400",
    },
    {
      label: "Events (7d)",
      value: analytics?.event_breakdown?.reduce((sum, e) => sum + e.count, 0) ?? "—",
      icon: "Activity",
      color: "bg-rose-500/10 text-rose-400",
    },
  ])

  // Lucide icon components as simple SVG renderers
  function icon(name: string, cls: string) {
    const icons: Record<string, string> = {
      MessageSquare: '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z"/></svg>',
      Brain: '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5a3 3 0 1 0-5.997.125 4 4 0 0 0-2.526 5.77 4 4 0 0 0 .556 6.588A4 4 0 1 0 12 18Z"/><path d="M12 5a3 3 0 1 1 5.997.125 4 4 0 0 1 2.526 5.77 4 4 0 0 1-.556 6.588A4 4 0 1 1 12 18Z"/><path d="M15 13a4.5 4.5 0 0 1-3-4 4.5 4.5 0 0 1-3 4"/><path d="M17.599 6.5a3 3 0 0 0 .399-1.375"/><path d="M6.003 5.125A3 3 0 0 0 6.401 6.5"/><path d="M3.477 10.896a4 4 0 0 1 .585-.396"/><path d="M19.938 10.5a4 4 0 0 1 .585.396"/><path d="M6 18a4 4 0 0 1-1.967-.516"/><path d="M19.967 17.484A4 4 0 0 1 18 18"/></svg>',
      Zap: '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/></svg>',
      Clock: '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>',
      Server: '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="2" rx="2" ry="2"/><rect width="20" height="8" x="2" y="14" rx="2" ry="2"/><line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/></svg>',
      Activity: '<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/></svg>',
    }
    return `<span class="${cls}">${icons[name] ?? ""}</span>`
  }

  onMount(() => {
    loadData()

    // Listen for container stats from WS
    const unsub = on("dashboard.containers", (data: any) => {
      containers = data?.containers ?? []
      containerStats = data?.stats ?? {}
    })

    return () => {
      unsub()
    }
  })
</script>

<div class="flex flex-col h-full overflow-hidden">
  <div class="flex items-center justify-between px-4 py-3 border-b border-gray-800 shrink-0">
    <div class="flex items-center gap-2">
      <svg class="w-4 h-4 text-blue-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="22 7 13.5 15.5 8.5 10.5 2 17"/><polyline points="16 7 22 7 22 13"/></svg>
      <h2 class="text-sm font-bold text-gray-200">Dashboard</h2>
    </div>
    <button
      onclick={loadData}
      disabled={loading}
      class="px-3 py-1 rounded-lg text-xs font-medium border border-gray-700 transition-colors flex items-center gap-1.5 text-gray-400 hover:text-gray-200 hover:border-gray-600 disabled:opacity-50"
    >
      <svg class="w-3 h-3 {loading ? 'animate-spin' : ''}" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>
      Refresh
    </button>
  </div>

  <div class="flex-1 overflow-y-auto p-4">
    {#if loading && !analytics}
      <div class="flex items-center justify-center h-40">
        <svg class="w-6 h-6 animate-spin text-gray-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
      </div>
    {:else}
      <!-- KPI Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-6 gap-3 mb-4">
        {#each kpis as kpi}
          <div class="bento-card p-3 flex flex-col gap-1.5 bg-gray-900/50 border border-gray-800 rounded-lg">
            <div class="flex items-center justify-between">
              <span class="text-[10px] font-bold uppercase tracking-[0.07em] text-gray-500">
                {kpi.label}
              </span>
              <div class="p-1 rounded-lg {kpi.color}">
                {@html icon(kpi.icon, "w-3.5 h-3.5")}
              </div>
            </div>
            <span class="text-lg font-semibold text-gray-100">{kpi.value}</span>
            {#if kpi.trend}
              <span class="text-[10px] text-gray-500">{kpi.trend}</span>
            {/if}
          </div>
        {/each}
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
        <!-- Container Status -->
        <div class="bento-card p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
          <div class="flex items-center gap-2 pb-2">
            <svg class="w-3.5 h-3.5 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="2" rx="2" ry="2"/><rect width="20" height="8" x="2" y="14" rx="2" ry="2"/><line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/></svg>
            <span class="text-[10px] font-bold uppercase tracking-[0.07em] text-gray-500">
              Containers
            </span>
          </div>
          <div class="space-y-1">
            {#if containers.length === 0}
              <p class="text-xs text-gray-500">Loading...</p>
            {:else}
              {#each containers as c}
                {@const name = (c.Names?.[0] || "").replace(/^\//, "")}
                {@const stats = containerStats[name]}
                <div class="flex items-center justify-between py-1 text-xs">
                  <div class="flex items-center gap-2">
                    <span class="w-1.5 h-1.5 rounded-full {c.State === 'running' ? 'bg-green-400' : 'bg-gray-400'}" />
                    <span class="font-mono text-[10px] text-gray-400">{name}</span>
                  </div>
                  <div class="flex items-center gap-2 text-[10px] text-gray-500">
                    {#if stats}
                      <span>CPU {stats.cpu_percent}%</span>
                      <span>MEM {stats.memory_percent}%</span>
                    {/if}
                    <span class="px-1 py-0.5 rounded text-[9px] {
                      c.State === 'running' ? 'bg-green-500/10 text-green-400' : 'bg-gray-500/10 text-gray-400'
                    }">
                      {c.State}
                    </span>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        </div>

        <!-- Event Breakdown -->
        <div class="bento-card p-4 bg-gray-900/50 border border-gray-800 rounded-lg">
          <div class="flex items-center gap-2 pb-2">
            <svg class="w-3.5 h-3.5 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/></svg>
            <span class="text-[10px] font-bold uppercase tracking-[0.07em] text-gray-500">
              Events (7 days)
            </span>
          </div>
          <div class="space-y-1">
            {#if !analytics?.event_breakdown?.length}
              <p class="text-xs text-gray-500">No events recorded</p>
            {:else}
              {#each analytics.event_breakdown.slice(0, 8) as event}
                <div class="flex items-center justify-between py-1 text-xs">
                  <span class="font-mono text-[10px] text-gray-400">{event.type}</span>
                  <span class="px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-400 text-[10px] font-medium">
                    {event.count}
                  </span>
                </div>
              {/each}
            {/if}
          </div>
        </div>

        <!-- Recent Sessions -->
        <div class="bento-card p-4 lg:col-span-2 bg-gray-900/50 border border-gray-800 rounded-lg">
          <div class="flex items-center gap-2 pb-2">
            <svg class="w-3.5 h-3.5 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z"/></svg>
            <span class="text-[10px] font-bold uppercase tracking-[0.07em] text-gray-500">
              Recent Sessions
            </span>
          </div>
          <div class="overflow-x-auto">
            <table class="w-full text-xs">
              <thead>
                <tr class="border-b border-gray-800">
                  <th class="text-left py-1.5 text-[9px] font-semibold uppercase tracking-wider text-gray-500">Session</th>
                  <th class="text-left py-1.5 text-[9px] font-semibold uppercase tracking-wider text-gray-500">Created</th>
                  <th class="text-right py-1.5 text-[9px] font-semibold uppercase tracking-wider text-gray-500">Messages</th>
                  <th class="text-right py-1.5 text-[9px] font-semibold uppercase tracking-wider text-gray-500">Chars</th>
                </tr>
              </thead>
              <tbody>
                {#if !analytics?.sessions?.length}
                  <tr><td colspan="4" class="py-3 text-center text-gray-500">No sessions</td></tr>
                {:else}
                  {#each analytics.sessions.slice(0, 10) as s}
                    <tr class="border-b border-gray-800/50 last:border-b-0 hover:bg-gray-800/30">
                      <td class="py-1.5 font-mono text-[10px] truncate max-w-[250px] text-gray-400">{s.session_key}</td>
                      <td class="py-1.5 text-[10px] text-gray-500">
                        {new Date(s.created_at).toLocaleDateString()}
                      </td>
                      <td class="py-1.5 text-right text-[10px] text-gray-400">{s.message_count}</td>
                      <td class="py-1.5 text-right text-[10px] text-gray-400">{s.total_chars > 1000 ? `${(s.total_chars / 1000).toFixed(1)}K` : s.total_chars}</td>
                    </tr>
                  {/each}
                {/if}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    {/if}
  </div>
</div>
