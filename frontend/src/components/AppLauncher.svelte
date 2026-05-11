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
    { type: "terminal", name: "Terminal", icon: "⬛", color: "text-gray-300", hoverBg: "hover:bg-gray-700" },
    { type: "editor", name: "Editor", icon: "📝", color: "text-blue-400", hoverBg: "hover:bg-blue-900/30" },
    { type: "preview", name: "Preview", icon: "👁", color: "text-green-400", hoverBg: "hover:bg-green-900/30" },
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

<div class="h-full bg-gray-900 overflow-auto">
  <!-- App Types Section -->
  <div class="p-3">
    <h2 class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">Launch App</h2>
    <div class="flex flex-col gap-2">
      {#each appTypes as app}
        <div
          class="flex items-center justify-between p-3 rounded-lg bg-gray-800 border border-gray-700 {app.hoverBg} transition-colors cursor-pointer"
          onclick={() => handleLaunch(app.type)}
        >
          <div class="flex items-center gap-3">
            <span class="text-lg">{app.icon}</span>
            <span class="text-sm font-medium {app.color}">{app.name}</span>
          </div>
          <button
            class="px-3 py-1 text-xs font-medium text-gray-300 bg-gray-700 rounded-md border border-gray-600 hover:bg-gray-600 transition-colors"
          >
            Launch
          </button>
        </div>
      {/each}
    </div>
  </div>

  <!-- Running Sessions Section -->
  <div class="p-3 border-t border-gray-800">
    <h2 class="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-3">Running Sessions</h2>
    {#if runningSessions.length > 0}
      <div class="flex flex-col gap-2">
        {#each runningSessions as session}
          <div class="flex items-center justify-between p-2 rounded bg-gray-800/50 border border-gray-700/50">
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
