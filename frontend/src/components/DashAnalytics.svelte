<script lang="ts">
  /**
   * DashAnalytics — Migrated from agent-os AnalyticsPage.tsx
   * Token usage charts, daily breakdown, model/skill tables.
   * Simplified for tile rendering (no i18n, no plugin slots).
   */
  import { onMount } from "svelte"
  import { send, on } from "../stores/ws"

  interface DailyEntry {
    day: string
    sessions: number
    input_tokens: number
    output_tokens: number
  }

  interface ModelEntry {
    model: string
    provider: string
    sessions: number
    input_tokens: number
    output_tokens: number
    cache_read_tokens: number
    reasoning_tokens: number
    api_calls: number
    estimated_cost: number
    tool_calls: number
    last_used_at: number
    capabilities: {
      supports_tools: boolean
      supports_vision: boolean
      supports_reasoning: boolean
      context_window: number
      max_output_tokens: number
      model_family: string
    }
  }

  interface SkillEntry {
    skill: string
    view_count: number
    manage_count: number
    total_count: number
    last_used_at: string | null
  }

  interface AnalyticsData {
    totals: {
      total_input: number
      total_output: number
      total_sessions: number
      total_api_calls: number
      total_estimated_cost: number | null
    }
    daily: DailyEntry[]
    by_model: ModelEntry[]
    skills: { top_skills: SkillEntry[] }
  }

  let days = $state(7)
  let data = $state<AnalyticsData | null>(null)
  let loading = $state(true)
  let error = $state<string | null>(null)

  const PERIODS = [
    { label: "7d", days: 7 },
    { label: "30d", days: 30 },
    { label: "90d", days: 90 },
  ] as const

  function formatTokens(n: number): string {
    if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
    if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
    return String(n)
  }

  function formatDate(day: string): string {
    try {
      const d = /^\d{4}-\d{2}-\d{2}$/.test(day)
        ? new Date(day + "T00:00:00")
        : new Date(day)
      if (Number.isNaN(d.getTime())) return day
      return d.toLocaleDateString("en-US", { month: "short", day: "numeric" })
    } catch {
      return day
    }
  }

  function shortModelName(model: string): string {
    const slashIdx = model.indexOf("/")
    if (slashIdx > 0) return model.slice(slashIdx + 1)
    return model
  }

  async function load() {
    loading = true
    error = null
    try {
      send({ protocol: "ui", method: "analytics.get", params: { days } })
    } catch (e) {
      error = String(e)
    } finally {
      // Loading cleared when result arrives
    }
  }

  on("analytics.result", (rawData: any) => {
    if (rawData) {
      // Assign directly
      const d = rawData as AnalyticsData
      data = d
    }
    loading = false
  })

  onMount(() => {
    load()
  })
</script>

<div class="flex flex-col h-full overflow-hidden bg-gray-950">
  
  <div class="flex items-center justify-between px-4 py-2.5 border-b border-gray-800 shrink-0">
    <div class="flex items-center gap-2">
      <svg class="w-3.5 h-3.5 text-blue-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>
      <h2 class="text-xs font-semibold text-gray-200">Analytics</h2>
      <span class="text-[10px] px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">
        {PERIODS.find(p => p.days === days)?.label ?? `${days}d`}
      </span>
    </div>
    <div class="flex items-center gap-1">
      {#each PERIODS as p}
        <button
          onclick={() => { days = p.days; load() }}
          class="text-[9px] px-2 py-0.5 rounded transition-colors {
            days === p.days
              ? 'bg-blue-600 text-white'
              : 'text-gray-500 hover:text-gray-300 hover:bg-gray-800'
          }"
        >{p.label}</button>
      {/each}
      <button
        onclick={load}
        disabled={loading}
        class="ml-1 p-1 rounded text-gray-500 hover:text-gray-300 disabled:opacity-50"
      >
        <svg class="w-3 h-3 {loading ? 'animate-spin' : ''}" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>
      </button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto p-3">
    {#if loading && !data}
      <div class="flex items-center justify-center h-32">
        <svg class="w-5 h-5 animate-spin text-gray-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
      </div>
    {:else if error}
      <div class="p-3 rounded border border-red-500/30 bg-red-500/5">
        <p class="text-xs text-red-400 text-center">{error}</p>
      </div>
    {:else if data}
      <!-- Stats summary -->
      <div class="grid grid-cols-3 gap-2 mb-3">
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2 text-center">
          <div class="text-sm font-semibold text-gray-100">{formatTokens(data.totals.total_input + data.totals.total_output)}</div>
          <div class="text-[9px] text-gray-500">Total Tokens</div>
        </div>
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2 text-center">
          <div class="text-sm font-semibold text-gray-100">{data.totals.total_sessions}</div>
          <div class="text-[9px] text-gray-500">Sessions</div>
        </div>
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2 text-center">
          <div class="text-sm font-semibold text-gray-100">
            {data.totals.total_estimated_cost != null ? `$${Number(data.totals.total_estimated_cost).toFixed(4)}` : "—"}
          </div>
          <div class="text-[9px] text-gray-500">Est. Cost</div>
        </div>
      </div>

      <!-- Token bar chart (simplified) -->
      {#if data.daily.length > 0}
        {@const maxTokens = Math.max(...data.daily.map(d => d.input_tokens + d.output_tokens), 1)}
        {@const CHART_H = 80}
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2.5 mb-3">
          <div class="flex items-center gap-3 text-[9px] text-gray-500 mb-2">
            <div class="flex items-center gap-1">
              <div class="h-2 w-2 bg-amber-500/50" />
              Input
            </div>
            <div class="flex items-center gap-1">
              <div class="h-2 w-2 bg-emerald-500/50" />
              Output
            </div>
          </div>
          <div class="flex items-end gap-[1px]" style="height: {CHART_H}px">
            {#each data.daily as d}
              {@const total = d.input_tokens + d.output_tokens}
              {@const inputH = Math.round((d.input_tokens / maxTokens) * CHART_H)}
              {@const outputH = Math.round((d.output_tokens / maxTokens) * CHART_H)}
              <div
                class="flex-1 min-w-0 group relative flex flex-col justify-end"
                style="height: {CHART_H}px"
              >
                <div class="w-full bg-amber-500/50" style="height: {Math.max(inputH, total > 0 ? 1 : 0)}px" />
                <div class="w-full bg-emerald-500/50" style="height: {Math.max(outputH, d.output_tokens > 0 ? 1 : 0)}px" />
              </div>
            {/each}
          </div>
          <div class="flex justify-between mt-1 text-[8px] text-gray-600">
            <span>{data.daily.length > 0 ? formatDate(data.daily[0].day) : ""}</span>
            <span>{data.daily.length > 1 ? formatDate(data.daily[data.daily.length - 1].day) : ""}</span>
          </div>
        </div>
      {/if}

      <!-- Model usage table (top 5) -->
      {#if data.by_model.length > 0}
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2.5">
          <div class="flex items-center gap-1.5 pb-1.5">
            <svg class="w-3 h-3 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M7 7h10"/><path d="M7 12h10"/><path d="M7 17h10"/></svg>
            <span class="text-[9px] font-bold uppercase tracking-wider text-gray-500">Top Models</span>
          </div>
          <div class="space-y-1">
            {#each data.by_model.slice(0, 5).sort((a, b) => (b.input_tokens + b.output_tokens) - (a.input_tokens + a.output_tokens)) as m}
              <div class="flex items-center justify-between py-0.5 text-[10px]">
                <div class="flex items-center gap-1.5">
                  <span class="font-mono text-gray-300">{shortModelName(m.model)}</span>
                  {#if m.capabilities?.supports_tools}
                    <span class="text-[8px] px-1 py-0.5 rounded bg-emerald-500/10 text-emerald-400">Tools</span>
                  {/if}
                  {#if m.capabilities?.supports_vision}
                    <span class="text-[8px] px-1 py-0.5 rounded bg-blue-500/10 text-blue-400">Vision</span>
                  {/if}
                </div>
                <div class="flex items-center gap-2 text-gray-500">
                  <span>{formatTokens(m.input_tokens)} / {formatTokens(m.output_tokens)}</span>
                  {#if m.estimated_cost > 0}
                    <span class="text-emerald-400">${Number(m.estimated_cost).toFixed(4)}</span>
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    {:else}
      <div class="flex flex-col items-center text-gray-500 py-8">
        <svg class="w-8 h-8 mb-2 opacity-40" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3v18h18"/><path d="m19 9-5 5-4-4-3 3"/></svg>
        <p class="text-xs font-semibold text-gray-300">No usage data</p>
        <p class="text-[10px] mt-1 text-gray-500">Start a session to see analytics</p>
      </div>
    {/if}
  </div>
</div>
