<script lang="ts">
  import { send } from "../stores/ws"

  let { visible = $bindable(false) }: { visible?: boolean } = $props()
  let query = $state("")
  let selectedIndex = $state(0)

  const commands = [
    { name: "New Terminal (Right)", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "split", direction: "h", content: "xterm" }}), shortcut: "" },
    { name: "New Terminal (Below)", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "split", direction: "v", content: "xterm" }}), shortcut: "" },
    { name: "Split Horizontal", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "swap", direction: "h" }}), shortcut: "⇧D" },
    { name: "Split Vertical", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "swap", direction: "v" }}), shortcut: "⇧D" },
    { name: "Close Tile", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "unmount" }}), shortcut: "⇧Q" },
    { name: "Toggle Fullscreen", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "fullscreen" }}), shortcut: "⇧F" },
  ]

  const filtered = $derived(query
    ? commands.filter(c => c.name.toLowerCase().includes(query.toLowerCase()))
    : commands)

  $effect(() => {
    if (!visible) { query = ""; selectedIndex = 0 }
  })

  $effect(() => {
    if (selectedIndex >= filtered.length) selectedIndex = 0
  })

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "ArrowDown") { e.preventDefault(); selectedIndex = Math.min(selectedIndex + 1, filtered.length - 1) }
    if (e.key === "ArrowUp") { e.preventDefault(); selectedIndex = Math.max(selectedIndex - 1, 0) }
    if (e.key === "Enter" && filtered[selectedIndex]) { filtered[selectedIndex].action(); visible = false }
    if (e.key === "Escape") visible = false
  }
</script>

{#if visible}
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-start justify-center pt-[12vh] z-50"
    onclick={(e) => { if (e.target === e.currentTarget) visible = false }}
  >
    <div
      class="w-[520px] max-h-[60vh] backdrop-blur-3xl bg-[#16161e]/95 border border-white/15 rounded-2xl shadow-panel overflow-hidden"
      role="dialog"
      aria-label="Command palette"
    >
      <!-- Search input -->
      <div class="px-5 pt-5 pb-2">
        <input
          bind:value={query}
          onkeydown={handleKeydown}
          placeholder="Type a command..."
          class="w-full bg-transparent text-xl text-white placeholder-white/30 outline-none font-light"
          autofocus
        />
      </div>

      <!-- Results -->
      <ul class="max-h-[40vh] overflow-y-auto px-2 pb-2">
        {#each filtered as cmd, i}
          <li
            class="flex items-center justify-between px-3 py-2.5 cursor-pointer rounded-lg transition-colors duration-100
              {i === selectedIndex ? 'bg-purple-500/15' : 'hover:bg-white/5'}"
            onclick={() => { cmd.action(); visible = false }}
            onmouseenter={() => { selectedIndex = i }}
          >
            <span class="text-sm text-white/90">{cmd.name}</span>
            {#if cmd.shortcut}
              <span class="bg-white/10 text-white/50 text-[10px] px-1.5 py-0.5 rounded font-mono">{cmd.shortcut}</span>
            {/if}
          </li>
        {:else}
          <li class="px-3 py-4 text-sm text-white/30 text-center">No commands found</li>
        {/each}
      </ul>
    </div>
  </div>
{/if}
