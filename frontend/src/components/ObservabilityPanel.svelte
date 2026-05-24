<script lang="ts">
  /**
   * ObservabilityPanel — Right panel tab for real-time event feed.
   * Features: live event stream, filtering by event type, connection status.
   */
  import { onMount } from "svelte"
  import { on, send } from "../stores/ws"

  type EventType = "tool_call" | "task_complete" | "delegation" | "assumption" | "drift" | "circuit_open" | "all"

  interface AIEEvent {
    type: EventType
    timestamp: string
    data: Record<string, unknown>
  }

  const EVENT_COLORS: Record<string, { text: string; bg: string }> = {
    tool_call: { text: "text-blue-400", bg: "bg-blue-400/10" },
    task_complete: { text: "text-emerald-400", bg: "bg-emerald-400/10" },
    delegation: { text: "text-purple-400", bg: "bg-purple-400/10" },
    assumption: { text: "text-amber-400", bg: "bg-yellow-400/10" },
    drift: { text: "text-red-400", bg: "bg-red-400/10" },
    circuit_open: { text: "text-orange-400", bg: "bg-orange-400/10" },
  }

  let events = $state<AIEEvent[]>([])
  let connected = $state(false)
  let filter = $state<EventType>("all")

  function formatEvent(event: AIEEvent): { summary: string; detail: string } {
    const d = event.data
    switch (event.type) {
      case "tool_call":
        return {
          summary: `🔧 ${d.tool_name ?? "unknown"}`,
          detail: JSON.stringify(d.tool_args ?? {}, null, 0).slice(0, 120),
        }
      case "task_complete":
        return {
          summary: `✅ Done (iter ${d.iteration ?? "?"})`,
          detail: String(d.final_content ?? "").slice(0, 150),
        }
      case "delegation":
        return {
          summary: `🤖 Delegation`,
          detail: JSON.stringify(d, null, 0).slice(0, 120),
        }
      case "assumption":
        return {
          summary: `💭 Assumption`,
          detail: String(d.message ?? JSON.stringify(d)).slice(0, 120),
        }
      case "drift":
        return {
          summary: `⚠️ Drift`,
          detail: String(d.message ?? JSON.stringify(d)).slice(0, 120),
        }
      case "circuit_open":
        return {
          summary: `🔌 Circuit Open`,
          detail: JSON.stringify(d, null, 0).slice(0, 120),
        }
      default:
        return {
          summary: event.type,
          detail: JSON.stringify(d, null, 0).slice(0, 120),
        }
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
    // Listen for observability events from WS
    const unsub = on("observability.event", (data: unknown) => {
      if (data) {
        const evt = data as AIEEvent
        events = [evt, ...events].slice(0, 200)
        connected = true
      }
    })

    // Also listen for status updates
    const unsub2 = on("observability.status", (data: unknown) => {
      const d = data as { connected?: boolean }
      connected = d?.connected ?? false
    })

    // Request connection status
    send({ protocol: "ui", method: "observability.status" })

    return () => {
      unsub()
      unsub2()
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
    {#each ["all", "tool_call", "task_complete", "delegation", "assumption", "drift", "circuit_open"] as t}
      <button
        onclick={() => filter = t as EventType}
        class="text-[10px] px-2 py-0.5 rounded-full transition-colors {
          filter === t
            ? 'bg-white/10 text-white'
            : 'text-gray-500 hover:text-gray-300'
        }"
      >
        {t === "all" ? "All" : t.replace(/_/g, " ")}
        {#if t !== "all" && typeCounts[t]}
          <span class="ml-0.5 opacity-60">({typeCounts[t]})</span>
        {/if}
      </button>
    {/each}
  </div>

  <!-- Events list -->
  <div class="flex-1 overflow-y-auto p-3 space-y-1.5">
    {#if filtered.length === 0}
      <p class="text-xs text-gray-500 text-center py-10">
        {connected ? "Waiting for events..." : "No events (disconnected)"}
      </p>
    {:else}
      {#each filtered as event (event.timestamp + event.type)}
        {@const { summary, detail } = formatEvent(event)}
        {@const color = EVENT_COLORS[event.type] ?? { text: "text-gray-400", bg: "bg-gray-400/10" }}
        <div class="text-[10px] bg-white/5 rounded p-2.5 border border-white/5 hover:border-white/10 transition-colors">
          <div class="flex items-start justify-between gap-1.5 mb-1">
            <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[9px] font-medium {color.bg} {color.text}">
              {event.type.replace(/_/g, " ")}
            </span>
            <span class="text-gray-500 ml-auto whitespace-nowrap text-[9px]">
              {new Date(event.timestamp).toLocaleTimeString("en-AU", { hour12: false })}
            </span>
          </div>
          <div class="font-medium mb-0.5 {color.text}">{summary}</div>
          {#if detail}<div class="text-gray-500 font-mono break-all text-[9px]">{detail}</div>{/if}
        </div>
      {/each}
    {/if}
  </div>
</div>