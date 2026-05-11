<script lang="ts">
  import { appsLaunch } from "../stores/ws"

  interface DockItem {
    id: string
    label: string
    emoji: string
    type: string
  }

  const dockItems: DockItem[] = [
    { id: "files", label: "Files", emoji: "📁", type: "file-manager" },
    { id: "terminal", label: "Terminal", emoji: "🖥️", type: "terminal" },
    { id: "agent", label: "Agent", emoji: "💬", type: "agent" },
    { id: "browser", label: "Browser", emoji: "🌐", type: "browser" },
    { id: "dashboard", label: "Dashboard", emoji: "📊", type: "dashboard" },
    { id: "voice", label: "Voice", emoji: "🎤", type: "audio" },
  ]

  let activeApp = $state<string | null>(null)
  let hoveredApp = $state<string | null>(null)

  function handleLaunch(item: DockItem) {
    appsLaunch(item.type)
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
        onclick={() => handleLaunch(item)}
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
