<script lang="ts">
  import { send } from "../stores/ws"

  let { visible = $bindable(false) }: { visible?: boolean } = $props()
  let query = $state("")
  let selectedIndex = $state(0)
  let activeCategory = $state("all")

  interface Command {
    id: string
    name: string
    description: string
    category: string
    shortcut: string
    action: () => void
  }

  // Fuzzy search function using substring matching with scoring
  function fuzzyScore(cmd: Command, q: string): number {
    if (!q) return 1
    const lowerName = cmd.name.toLowerCase()
    const lowerDesc = cmd.description.toLowerCase()
    const lowerQuery = q.toLowerCase()
    
    // Exact match gets highest score
    if (lowerName === lowerQuery) return 100
    // Starts with query
    if (lowerName.startsWith(lowerQuery)) return 80
    // Word boundary match
    const words = lowerName.split(/\s+/)
    for (const word of words) {
      if (word.startsWith(lowerQuery)) return 60
    }
    // Contains query
    if (lowerName.includes(lowerQuery)) return 40
    if (lowerDesc.includes(lowerQuery)) return 20
    // Character-by-character fuzzy match
    let queryIdx = 0
    let score = 0
    let consecutiveBonus = 0
    for (let i = 0; i < lowerName.length && queryIdx < lowerQuery.length; i++) {
      if (lowerName[i] === lowerQuery[queryIdx]) {
        score += 1 + consecutiveBonus
        consecutiveBonus += 0.5
        queryIdx++
      } else {
        consecutiveBonus = 0
      }
    }
    return queryIdx === lowerQuery.length ? score : 0
  }

  const commands: Command[] = [
    // Layout commands
    { id: "layout-split-h", name: "Split Horizontal", description: "Split panel horizontally", category: "Layout", shortcut: "⇧D", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "swap", direction: "h" }}) },
    { id: "layout-split-v", name: "Split Vertical", description: "Split panel vertically", category: "Layout", shortcut: "⇧D", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "swap", direction: "v" }}) },
    { id: "layout-close", name: "Close Tile", description: "Close current tile", category: "Layout", shortcut: "⇧Q", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "unmount" }}) },
    { id: "layout-fullscreen", name: "Toggle Fullscreen", description: "Toggle fullscreen mode", category: "Layout", shortcut: "⇧F", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "fullscreen" }}) },
    { id: "layout-reset", name: "Reset Layout", description: "Reset to default layout", category: "Layout", shortcut: "", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "reset" }}) },
    
    // Terminal commands
    { id: "term-new-right", name: "New Terminal (Right)", description: "Open new terminal in horizontal split", category: "Terminal", shortcut: "", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "split", direction: "h", content: "xterm" }}) },
    { id: "term-new-below", name: "New Terminal (Below)", description: "Open new terminal in vertical split", category: "Terminal", shortcut: "", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "split", direction: "v", content: "xterm" }}) },
    { id: "term-clear", name: "Clear Terminal", description: "Clear terminal output", category: "Terminal", shortcut: "", action: () => send({ protocol: "ui", method: "terminal.clear", params: {}}) },
    { id: "term-kill", name: "Kill Terminal Process", description: "Kill current terminal process", category: "Terminal", shortcut: "", action: () => send({ protocol: "ui", method: "terminal.kill", params: {}}) },
    
    // Navigation commands
    { id: "nav-left", name: "Focus Left Panel", description: "Navigate to left panel", category: "Navigation", shortcut: "⌘[", action: () => send({ protocol: "ui", method: "layout.focus", params: { direction: "left" }}) },
    { id: "nav-right", name: "Focus Right Panel", description: "Navigate to right panel", category: "Navigation", shortcut: "⌘]", action: () => send({ protocol: "ui", method: "layout.focus", params: { direction: "right" }}) },
    { id: "nav-up", name: "Focus Upper Panel", description: "Navigate to upper panel", category: "Navigation", shortcut: "", action: () => send({ protocol: "ui", method: "layout.focus", params: { direction: "up" }}) },
    { id: "nav-down", name: "Focus Lower Panel", description: "Navigate to lower panel", category: "Navigation", shortcut: "", action: () => send({ protocol: "ui", method: "layout.focus", params: { direction: "down" }}) },
    
    // Session commands
    { id: "session-new", name: "New Session", description: "Create a new session", category: "Session", shortcut: "⌘N", action: () => send({ protocol: "ui", method: "session.create", params: {}}) },
    { id: "session-save", name: "Save Session", description: "Save current session state", category: "Session", shortcut: "⌘S", action: () => send({ protocol: "ui", method: "session.save", params: {}}) },
    { id: "session-load", name: "Load Session", description: "Load a saved session", category: "Session", shortcut: "⌘O", action: () => send({ protocol: "ui", method: "session.load", params: {}}) },
    
    // File commands
    { id: "file-new", name: "New File", description: "Create a new file", category: "Files", shortcut: "⌘N", action: () => send({ protocol: "ui", method: "fs.create", params: { type: "file" }}) },
    { id: "file-folder", name: "New Folder", description: "Create a new folder", category: "Files", shortcut: "⇧⌘N", action: () => send({ protocol: "ui", method: "fs.create", params: { type: "folder" }}) },
    { id: "file-save", name: "Save File", description: "Save current file", category: "Files", shortcut: "⌘S", action: () => send({ protocol: "ui", method: "fs.save", params: {}}) },
    
    // LLM commands
    { id: "llm-chat", name: "Open Chat Panel", description: "Open LLM chat interface", category: "LLM", shortcut: "⌘L", action: () => send({ protocol: "ui", method: "panel.toggle", params: { panel: "chat" }}) },
    { id: "llm-model", name: "Switch LLM Model", description: "Change current LLM model", category: "LLM", shortcut: "", action: () => send({ protocol: "ui", method: "llm.model", params: {}}) },
    { id: "llm-provider", name: "Switch LLM Provider", description: "Change LLM provider (OpenAI, Anthropic, etc.)", category: "LLM", shortcut: "", action: () => send({ protocol: "ui", method: "llm.provider", params: {}}) },
    
    // Settings commands
    { id: "settings-open", name: "Open Settings", description: "Open settings panel", category: "Settings", shortcut: "⌘,", action: () => send({ protocol: "ui", method: "panel.toggle", params: { panel: "settings" }}) },
    { id: "settings-theme", name: "Change Theme", description: "Switch color theme", category: "Settings", shortcut: "", action: () => send({ protocol: "ui", method: "settings.theme", params: {}}) },
    { id: "settings-keymap", name: "Keyboard Shortcuts", description: "View keyboard shortcuts", category: "Settings", shortcut: "⌘/", action: () => send({ protocol: "ui", method: "panel.toggle", params: { panel: "keymap" }}) },
  ]

  const categories = ["all", "Layout", "Terminal", "Navigation", "Session", "Files", "LLM", "Settings"]

  const filtered = $derived((() => {
    const q = query.trim()
    let results: Command[]
    
    if (q === "") {
      results = activeCategory === "all" ? commands : commands.filter(c => c.category === activeCategory)
    } else {
      // Filter by category if not "all"
      const categoryFiltered = activeCategory === "all" ? commands : commands.filter(c => c.category === activeCategory)
      // Score and sort by fuzzy match
      results = categoryFiltered
        .map(cmd => ({ cmd, score: fuzzyScore(cmd, q) }))
        .filter(({ score }) => score > 0)
        .sort((a, b) => b.score - a.score)
        .map(({ cmd }) => cmd)
    }
    
    return results
  })())

  $effect(() => {
    if (!visible) { query = ""; selectedIndex = 0; activeCategory = "all" }
  })

  $effect(() => {
    if (selectedIndex >= filtered.length) selectedIndex = Math.max(0, filtered.length - 1)
  })

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "ArrowDown") { e.preventDefault(); selectedIndex = Math.min(selectedIndex + 1, filtered.length - 1) }
    if (e.key === "ArrowUp") { e.preventDefault(); selectedIndex = Math.max(selectedIndex - 1, 0) }
    if (e.key === "Enter" && filtered[selectedIndex]) { filtered[selectedIndex].action(); visible = false }
    if (e.key === "Escape") visible = false
    if (e.key === "Tab" && filtered.length > 0) {
      e.preventDefault()
      const currentCategory = filtered[selectedIndex].category
      const categoryCommands = filtered.filter(c => c.category === currentCategory)
      const currentIdxInCategory = categoryCommands.findIndex(c => c.id === filtered[selectedIndex].id)
      const nextIdx = (currentIdxInCategory + 1) % categoryCommands.length
      const nextCommand = categoryCommands[nextIdx]
      selectedIndex = filtered.findIndex(c => c.id === nextCommand.id)
    }
  }
  
  function selectCategory(cat: string) {
    activeCategory = cat
    selectedIndex = 0
  }
  
  function getCategoryIcon(cat: string): string {
    const icons: Record<string, string> = {
      "all": "◎",
      "Layout": "⊞",
      "Terminal": "▽",
      "Navigation": "◁",
      "Session": "◯",
      "Files": "⊠",
      "LLM": "◉",
      "Settings": "⚙"
    }
    return icons[cat] || "○"
  }
</script>

{#if visible}
  <div
    class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-start justify-center pt-[10vh] z-50"
    onclick={(e) => { if (e.target === e.currentTarget) visible = false }}
    onkeydown={(e) => { if (e.key === "Escape") visible = false }}
    role="dialog"
    aria-label="Command palette"
  >
    <div
      class="w-[640px] max-h-[70vh] backdrop-blur-3xl bg-[#16161e]/95 border border-white/15 rounded-2xl shadow-panel overflow-hidden"
    >
      <!-- Search input -->
      <div class="px-5 pt-5 pb-3">
        <input
          bind:value={query}
          onkeydown={handleKeydown}
          placeholder="Type a command or search..."
          class="w-full bg-transparent text-lg text-white placeholder-white/30 outline-none font-light"
          autofocus
        />
      </div>
      
      <!-- Category tabs -->
      <div class="px-4 pb-2 flex gap-1 overflow-x-auto">
        {#each categories as cat}
          <button
            class="px-3 py-1 text-xs rounded-lg transition-colors whitespace-nowrap
              {activeCategory === cat 
                ? 'bg-purple-500/25 text-purple-300' 
                : 'bg-white/5 text-white/50 hover:bg-white/10 hover:text-white/70'}"
            onclick={() => selectCategory(cat)}
          >
            {getCategoryIcon(cat)} {cat}
          </button>
        {/each}
      </div>
      
      <div class="border-t border-white/5"></div>
      
      <!-- Results -->
      <ul class="max-h-[45vh] overflow-y-auto px-2 py-2">
        {#if filtered.length > 0}
          {#each filtered as cmd, i}
            <li
              class="flex items-center justify-between px-3 py-2.5 cursor-pointer rounded-lg transition-colors duration-100 group
                {i === selectedIndex ? 'bg-purple-500/20' : 'hover:bg-white/5'}"
              onclick={() => { cmd.action(); visible = false }}
              onmouseenter={() => { selectedIndex = i }}
            >
              <div class="flex items-center gap-3">
                <span class="text-xs text-white/30 w-16 shrink-0">{cmd.category}</span>
                <div class="flex flex-col">
                  <span class="text-sm text-white/90 group-hover:text-white">{cmd.name}</span>
                  <span class="text-xs text-white/40">{cmd.description}</span>
                </div>
              </div>
              {#if cmd.shortcut}
                <span class="bg-white/10 text-white/50 text-[10px] px-1.5 py-0.5 rounded font-mono shrink-0">{cmd.shortcut}</span>
              {/if}
            </li>
          {/each}
        {:else}
          <li class="px-3 py-8 text-sm text-white/30 text-center">
            <div class="text-2xl mb-2">⚠</div>
            No commands found for "{query}"
          </li>
        {/if}
      </ul>
      
      <!-- Footer hint -->
      <div class="border-t border-white/5 px-4 py-2 flex items-center justify-between text-[10px] text-white/25">
        <span><kbd class="bg-white/10 px-1 rounded">↑↓</kbd> navigate</span>
        <span><kbd class="bg-white/10 px-1 rounded">↵</kbd> select</span>
        <span><kbd class="bg-white/10 px-1 rounded">Tab</kbd> next in category</span>
        <span><kbd class="bg-white/10 px-1 rounded">Esc</kbd> close</span>
      </div>
    </div>
  </div>
{/if}