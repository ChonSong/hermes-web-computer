<script lang="ts">
  /**
   * ObservabilityPanel — Right panel tab for real-time event feed.
   * Features: live event stream, filtering by event type, connection status.
   * Step 6.1: Wired to observability.events WS handler for real telemetry data.
   */
  import { onMount } from "svelte"
  import { on, observabilityStatus, observabilityEvents } from "../stores/ws"

  type EventType = "tool_call" | "task_complete" | "delegation" | "assumption" | "drift" | "circuit_open" | "session.connected" | "layout.update" | "interrupt" | "approval.granted" | "pty.write" | "chat.send" | "all"

  interface AIEEvent {
    type: string
    timestamp: string
    data: Record<string, unknown>
  }

  interface TelemetryEvent {
    type: string
    timestamp: string
    data: {
      session?: string
      user?: string
      policy?: string
      command?: string
      tool?: string
      path?: string
      outcome?: string
      drift_score?: number
    }
  }

  const EVENT_COLORS: Record<string, { text: string; bg: string }> = {
    tool_call: { text: "text-blue-400", bg: "bg-blue-400/10" },
    task_complete: { text: "text-emerald-400", bg: "bg-emerald-400/10" },
    delegation: { text: "text-purple-400", bg: "bg-purple-400/10" },
    assumption: { text: "text-amber-400", bg: "bg-yellow-400/10" },
    drift: { text: "text-red-400", bg: "bg-red-400/10" },
    circuit_open: { text: "text-orange-400", bg: "bg-orange-400/10" },
    "session.connected": { text: "text-cyan-400", bg: "bg-cyan-400/10" },
    "layout.update": { text: "text-teal-400", bg: "bg-teal-400/10" },
    "interrupt": { text: "text-red-400", bg: "bg-red-400/10" },
    "approval.granted": { text: "text-green-400", bg: "bg-green-400/10" },
    "pty.write": { text: "text-yellow-400", bg: "bg-yellow-400/10" },
    "chat.send": { text: "text-indigo-400", bg: "bg-indigo-400/10" },
  }

  const ALL_FILTERS = [
    { value: "all", label: "All" },
    { value: "tool_call", label: "Tool" },
    { value: "task_complete", label: "Done" },
    { value: "delegation", label: "Delegation" },
    { value: "assumption", label: "Assumption" },
    { value: "drift", label: "Drift" },
    { value: "circuit_open", label: "Circuit" },
    { value: "session.connected", label: "Session" },
    { value: "layout.update", label: "Layout" },
    { value: "interrupt", label: "Interrupt" },
  ] as const

  let events = $state<AIEEvent[]>([])
  let connected = $state(false)
  let filter = $state<string>("all")

  function formatEvent(event: AIEEvent): { summary: string; detail: string } {
    const d = event.data ?? {}
    const t = event.type

    switch (t) {
      case "tool_call":
        return {
          summary: `🔧 ${d.tool_name ?? d.tool ?? "unknown"}`,
          detail: String(d.tool_args ?? d.command ?? d.tool ?? "").slice(0, 120),
        }
      case "task_complete":
        return {
          summary: `✅ Done (iter ${d.iteration ?? "?"})`,
          detail: String(d.final_content ?? d.outcome ?? "").slice(0, 150),
        }
      case "delegation":
        return { summary: `🤖 Delegation`, detail: JSON.stringify(d, null, 0).slice(0, 120) }
      case "assumption":
        return { summary: `💭 Assumption`, detail: String(d.message ?? JSON.stringify(d)).slice(0, 120) }
      case "drift":
        return { summary: `⚠️ Drift`, detail: String(d.message ?? JSON.stringify(d)).slice(0, 120) }
      case "circuit_open":
        return { summary: `🔌 Circuit Open`, detail: JSON.stringify(d, null, 0).slice(0, 120) }
      case "session.connected":
        return { summary: `🟢 Session connected`, detail: `session: ${d.session ?? "?"}` }
      case "layout.update":
        return { summary: `📐 Layout ${d.command ?? "update"}`, detail: `session: ${d.session ?? "?"}` }
      case "interrupt":
        return { summary: `⏹ Interrupt`, detail: `session: ${d.session ?? "?"}` }
      case "approval.granted":
        return { summary: `🔓 Approval granted`, detail: `token: ${String(d.token ?? d.command ?? "").slice(0, 60)}` }
      case "pty.write":
        return { summary: `⌨️ PTY write`, detail: String(d.command ?? "").slice(0, 80) }
      case "chat.send":
        return { summary: `💬 Chat send`, detail: String(d.command ?? "").slice(0, 80) }
      default:
        return { summary: t.replace(/[._]/g, " "), detail: JSON.stringify(d, null, 0).slice(0, 120) }
    }
  }

  const filtered = $derived(
    filter === "all" ? events : events.filter(e => e.type === filter)
  )

  const typeCounts = $derived.by(() => {
    const acc: Record<string, number> = {}
    for (const e of events) {
      if (e.type !== "all") {
        acc[e.type] = (acc[e.type] || 0) + 1
      }
    }
    return acc
  })

  onMount(() => {
    // Listen for live events from WS
    const unsub1 = on("observability.event", (data: unknown) => {
      if (data) {
        const evt = data as AIEEvent
        events = [evt, ...events].slice(0, 200)
        connected = true
      }
    })

    // Step 6.1: Listen for bulk telemetry from ring buffer
    const unsub2 = on("observability.events.result", (data: unknown) => {
      const d = data as { events?: TelemetryEvent[]; count?: number }
      if (d?.events) {
        const existing = new Set(events.map(ev => `${ev.type}:${ev.timestamp}`))
        const newEvents: AIEEvent[] = d.events
          .map(e => ({
            type: e.type,
            timestamp: e.timestamp,
            data: e.data as Record<string, unknown>,
          }))
          .filter(e => !existing.has(`${e.type}:${e.timestamp}`))
        if (newEvents.length > 0) {
          events = [...newEvents, ...events].slice(0, 200)
        }
        connected = true
      }
    })

    // Listen for status updates
    const unsub3 = on("observability.status", (data: unknown) => {
      const d = data as { connected?: boolean }
      connected = d?.connected ?? false
    })

    // Step 6.1: Request telemetry ring buffer events on mount
    observabilityStatus()
    observabilityEvents(200)

    return () => {
      unsub1()
      unsub2()
      unsub3()
    }
  })
</script>

<div class="flex flex-col h-full bg-[#191919]">
  <!-- Header -->
  <div class="flex-none px-4 py-2.5 border-b border-white/10 flex items-center justify-between">
    <div class="flex items-center gap-2">
      <h2 class="text-white font-semibold text-sm">Observability</h2>
      <span
        class="text-[10px] px-2 py-0.5 rounded-full {
          connected ? 'bg-emerald-400/10 text-emerald-400' : 'bg-gray-500/10 text-gray-500'
        }"
      >
        {connected ? "Live" : "Disconnected"}
      </span>
      <span class="text-[10px] text-gray-500">{events.length} events</span>
    </div>
  </div>

  <!-- Filter bar -->
  <div class="flex-none px-3 py-2 border-b border-white/5 flex gap-0.5 flex-wrap">
    {#each ALL_FILTERS as f}
      <button
        onclick={() => filter = f.value}
        class="text-[10px] px-2 py-0.5 rounded-full transition-colors {
          filter === f.value
            ? 'bg-white/10 text-white'
            : 'text-gray-500 hover:text-gray-300'
        }"
      >
        {f.label}
        {#if f.value !== "all" && typeCounts[f.value]}
          <span class="ml-0.5 opacity-60">({typeCounts[f.value]})</span>
        {/if}
      </button>
    {/each}
  </div>

  <!-- Events list -->
  <div class="flex-1 overflow-y-auto p-3 space-y-1.5">
    {#if filtered.length === 0}
      <p class="text-xs text-gray-500 text-center py-10">
        {connected ? "Waiting for events..." : "No events (telemetry may be empty)"}
      </p>
    {:else}
      {#each filtered as event (event.timestamp + event.type)}
        {@const { summary, detail } = formatEvent(event)}
        {@const color = EVENT_COLORS[event.type] ?? { text: "text-gray-400", bg: "bg-gray-400/10" }}
        <div class="text-[10px] bg-white/5 rounded p-2.5 border border-white/5 hover:border-white/10 transition-colors">
          <div class="flex items-start justify-between gap-1.5 mb-1">
            <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[9px] font-medium {color.bg} {color.text}">
              {event.type.replace(/[._]/g, " ")}
            </span>
            <span class="text-gray-500 ml-auto whitespace-nowrap text-[9px]">
              {(() => {
                try { return new Date(event.timestamp).toLocaleTimeString("en-AU", { hour12: false }) }
                catch { return event.timestamp }
              })()}
            </span>
          </div>
          <div class="font-medium mb-0.5 {color.text}">{summary}</div>
          {#if detail}<div class="text-gray-500 font-mono break-all text-[9px]">{detail}</div>{/if}
        </div>
      {/each}
    {/if}
  </div>
</div>