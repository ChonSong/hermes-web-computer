<script lang="ts">
  import { createEventDispatcher, onMount } from "svelte"
  import { on, appsLaunch } from "../stores/ws"

  interface AppType {
    type: string
    name: string
    icon: string
    color: string
    hoverBg: string
  }

  const appTypes: AppType[] = [
    { type: "terminal", name: "Terminal", icon: "⬛", color: "text-gray-300", hoverBg: "hover:bg-white/5" },
    { type: "editor", name: "Editor", icon: "📝", color: "text-purple-400", hoverBg: "hover:bg-purple-500/10" },
    { type: "preview", name: "Preview", icon: "👁", color: "text-emerald-400", hoverBg: "hover:bg-emerald-500/10" },
  ]

  interface Session {
    id: string
    type: string
    name: string
  }

  let runningSessions = $state<Session[]>([])

  function handleLaunch(type: string) {
    appsLaunch(type)
  }

  // Listen for launch responses
  onMount(() => {
    on("apps.launch.response", (data: unknown) => {
      const resp = data as { type: string; pty_id?: string }
      if (resp.pty_id) {
        runningSessions.push({ id: resp.pty_id, type: resp.type, name: `${resp.type} (${resp.pty_id.slice(4, 8)})` })
      }
    })

    on("apps.error", (data: unknown) => {
      const resp = data as { message?: string }
      console.error("App launch error:", resp.message)
    })
  })
</script>

<div class="h-full overflow-auto">
  <!-- App Types Section -->
  <div class="p-3">
    <h2 class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">Launch App</h2>
    <div class="flex flex-col gap-2">
      {#each appTypes as app}
        <div
          class="flex items-center justify-between p-3 rounded-lg bg-white/5 border border-white/10 {app.hoverBg} transition-colors cursor-pointer"
          onclick={() => handleLaunch(app.type)}
        >
          <div class="flex items-center gap-3">
            <span class="text-lg">{app.icon}</span>
            <span class="text-sm font-medium {app.color}">{app.name}</span>
          </div>
          <button
            class="px-3 py-1 text-xs font-medium text-gray-300 bg-white/10 rounded-md border border-white/10 hover:bg-white/15 transition-colors"
          >
            Launch
          </button>
        </div>
      {/each}
    </div>
  </div>

  <!-- Running Sessions Section -->
  <div class="p-3 border-t border-white/5">
    <h2 class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">Running Sessions</h2>
    {#if runningSessions.length > 0}
      <div class="flex flex-col gap-2">
        {#each runningSessions as session}
          <div class="flex items-center justify-between p-2 rounded-lg bg-white/5 border border-white/10">
            <span class="text-sm text-gray-400">{session.name}</span>
            <span class="text-xs text-gray-500 capitalize">{session.type}</span>
          </div>
        {/each}
      </div>
    {:else}
      <p class="text-sm text-gray-600 italic">No running sessions</p>
    {/if}
  </div>
</div>
