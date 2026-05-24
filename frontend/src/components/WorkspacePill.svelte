<script lang="ts">
  let activeWorkspace = $state(1)
  const workspaceCount = 9

  function formatTime(): string {
    const now = new Date()
    return now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
  }

  let currentTime = $state(formatTime())
  $effect(() => {
    const interval = setInterval(() => { currentTime = formatTime() }, 10_000)
    return () => clearInterval(interval)
  })
</script>

<div class="fixed top-2 left-1/2 -translate-x-1/2 z-50
  backdrop-blur-xl border border-white/10 rounded-full
  px-4 py-1 flex items-center gap-3 shadow-float"
  style="background-color: #1c1c1d; opacity: 0.95;">
  <!-- Workspace dots -->
  {#each Array.from({ length: workspaceCount }, (_, i) => i + 1) as ws}
    <button
      class="w-2 h-2 rounded-full transition-all duration-150 {ws === activeWorkspace
        ? 'bg-purple-400 shadow-glow'
        : 'bg-white/20 hover:bg-white/40'}"
      onclick={() => activeWorkspace = ws}
      aria-label={`Workspace ${ws}`}
    ></button>
  {/each}

  <!-- Separator -->
  <div class="w-px h-3 bg-white/10"></div>

  <!-- System time -->
  <span class="text-[10px] text-white/60 font-mono">{currentTime}</span>

  <!-- Agent status dot (green=processing, gray=idle) -->
  <div class="w-1.5 h-1.5 rounded-full bg-green-400 animate-pulse"></div>
</div>
