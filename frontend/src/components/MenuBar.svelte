/**
 * MenuBar.svelte — VSCode-style top menu bar
 * Phase 7: Menu bar with File/Edit/View/Go/Run/Terminal/Help
 * Spec: docs/WAYBAR-SPEC.md §9
 */
<script lang="ts">
  import { send, sendOp } from "../stores/ws"
  import { setActiveWorkspace } from "../stores/workspace"

  type MenuId = "file" | "edit" | "view" | "go" | "run" | "terminal" | "help" | null

  let openMenu = $state<MenuId>(null)

  function toggleMenu(id: MenuId) {
    openMenu = openMenu === id ? null : id
  }

  function closeMenu() {
    openMenu = null
  }

  // Handle custom event from App.svelte (Alt+F/E/V/G/R/T/H shortcuts)
  function handleOpenMenu(e: Event) {
    const detail = (e as CustomEvent<string>).detail
    toggleMenu(detail as MenuId)
  }

  // Close on outside click
  function handleWindowClick(e: MouseEvent) {
    const target = e.target as HTMLElement
    if (!target.closest(".menu-bar")) {
      closeMenu()
    }
  }

  // Close on Escape
  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") closeMenu()
  }

  // Listen for hwc-open-menu events from App.svelte (Alt+key shortcuts)
  $effect(() => {
    if (typeof window !== "undefined") {
      window.addEventListener("hwc-open-menu", handleOpenMenu)
      return () => window.removeEventListener("hwc-open-menu", handleOpenMenu)
    }
  })

  // --- Menu actions ---
  function newTerminal() {
    send({ protocol: "ui", method: "apps.launch", params: { type: "terminal" } })
    closeMenu()
  }

  function newSession() {
    send({ protocol: "ui", method: "session.new" })
    closeMenu()
  }

  function openSettings() {
    send({ protocol: "ui", method: "apps.launch", params: { type: "settings" } })
    closeMenu()
  }

  function toggleLeftPanel() {
    // Dispatch custom event that App.svelte listens to
    window.dispatchEvent(new CustomEvent("hwc-toggle-left-panel"))
    closeMenu()
  }

  function toggleRightPanel() {
    window.dispatchEvent(new CustomEvent("hwc-toggle-right-panel"))
    closeMenu()
  }

  function toggleBottomPanel() {
    window.dispatchEvent(new CustomEvent("hwc-toggle-bottom-panel"))
    closeMenu()
  }

  function openCommandPalette() {
    window.dispatchEvent(new CustomEvent("hwc-toggle-palette"))
    closeMenu()
  }

  function cycleLayout() {
    send({ protocol: "ui", method: "layout.update", params: { op: "layout-mode", mode: "cycle" } })
    closeMenu()
  }

  function goWorkspace(n: number) {
    setActiveWorkspace(n)
    closeMenu()
  }

  function closeFocusedTile() {
    const focusEl = document.activeElement
    // We need the focused tile ID - send via a custom event
    window.dispatchEvent(new CustomEvent("hwc-close-focused"))
    closeMenu()
  }

  function fullscreenFocused() {
    window.dispatchEvent(new CustomEvent("hwc-fullscreen-focused"))
    closeMenu()
  }

  function restartBackend() {
    send({ protocol: "ui", method: "system.restart" })
    closeMenu()
  }

  // Menu definitions
  const menus: { id: MenuId; label: string; shortcut?: string }[] = [
    { id: "file", label: "File", shortcut: "Alt+F" },
    { id: "edit", label: "Edit", shortcut: "Alt+E" },
    { id: "view", label: "View", shortcut: "Alt+V" },
    { id: "go", label: "Go", shortcut: "Alt+G" },
    { id: "run", label: "Run", shortcut: "Alt+R" },
    { id: "terminal", label: "Terminal", shortcut: "Alt+T" },
    { id: "help", label: "Help", shortcut: "Alt+H" },
  ]

  const menuItems: Record<MenuId, { label: string; shortcut?: string; action: () => void; divider?: boolean }[]> = {
    file: [
      { label: "New Session", shortcut: "Ctrl+Shift+N", action: newSession },
      { label: "New Terminal", shortcut: "Ctrl+Shift+`", action: newTerminal },
      { divider: true, label: "", action: () => {} },
      { label: "Open Settings", shortcut: "Ctrl+,", action: openSettings },
      { divider: true, label: "", action: () => {} },
      { label: "Restart Backend", action: restartBackend },
    ],
    edit: [
      { label: "Command Palette", shortcut: "Ctrl+K", action: openCommandPalette },
    ],
    view: [
      { label: "Toggle Left Panel", shortcut: "Ctrl+B", action: toggleLeftPanel },
      { label: "Toggle Right Panel", shortcut: "Ctrl+Shift+B", action: toggleRightPanel },
      { label: "Toggle Bottom Panel", shortcut: "Ctrl+`", action: toggleBottomPanel },
      { divider: true, label: "", action: () => {} },
      { label: "Cycle Layout Mode", shortcut: "Shift+D", action: cycleLayout },
    ],
    go: [
      { label: "Workspace 1", shortcut: "Shift+1", action: () => goWorkspace(1) },
      { label: "Workspace 2", shortcut: "Shift+2", action: () => goWorkspace(2) },
      { label: "Workspace 3", shortcut: "Shift+3", action: () => goWorkspace(3) },
      { label: "Workspace 4", shortcut: "Shift+4", action: () => goWorkspace(4) },
      { label: "Workspace 5", shortcut: "Shift+5", action: () => goWorkspace(5) },
      { label: "Workspace 6", shortcut: "Shift+6", action: () => goWorkspace(6) },
      { label: "Workspace 7", shortcut: "Shift+7", action: () => goWorkspace(7) },
      { label: "Workspace 8", shortcut: "Shift+8", action: () => goWorkspace(8) },
      { label: "Workspace 9", shortcut: "Shift+9", action: () => goWorkspace(9) },
    ],
    run: [
      { label: "Fullscreen Focused", shortcut: "Shift+F", action: fullscreenFocused },
      { label: "Close Focused Tile", shortcut: "Shift+Q", action: closeFocusedTile },
    ],
    terminal: [
      { label: "New Terminal", shortcut: "Ctrl+Shift+`", action: newTerminal },
      { label: "Interrupt", shortcut: "Shift+Space", action: () => { send({ protocol: "ui", method: "interrupt" }); closeMenu() } },
    ],
    help: [
      { label: "Keyboard Shortcuts", shortcut: "Ctrl+?", action: () => { window.dispatchEvent(new CustomEvent("hwc-toggle-keymap")); closeMenu() } },
      { label: "About", action: () => { closeMenu() } },
    ],
  }
</script>

<svelte:window onclick={handleWindowClick} onkeydown={handleKeydown} />

<!-- Menu Bar — sits above Waybar -->
<div class="fixed top-0 left-0 right-0 z-[999] flex items-center h-7
  bg-[rgba(18,18,26,0.95)] border-b border-white/10 backdrop-blur-md
  font-mono text-[12px] menu-bar"
  role="menubar"
>
  {#each menus as menu}
    <div class="relative">
      <button
        class="h-full px-3 flex items-center gap-1 transition-colors
          {openMenu === menu.id ? 'bg-white/10 text-white' : 'text-gray-400 hover:text-white hover:bg-white/5'}"
        onclick={(e) => { e.stopPropagation(); toggleMenu(menu.id) }}
        role="menuitem"
        aria-haspopup="true"
        aria-expanded={openMenu === menu.id}
      >
        {menu.label}
        <span class="text-[9px] text-gray-600">{menu.shortcut || ""}</span>
      </button>

      {#if openMenu === menu.id}
        <!-- Dropdown -->
        <div
          class="absolute top-full left-0 mt-0.5 min-w-[220px] py-1
            bg-[#1e1e2e] border border-white/10 rounded-lg shadow-xl backdrop-blur-xl
            text-gray-300"
          role="menu"
        >
          {#each menuItems[menu.id] || [] as item}
            {#if item.divider}
              <div class="my-1 border-t border-white/5"></div>
            {:else}
              <button
                class="w-full flex items-center justify-between px-3 py-1.5 text-left
                  hover:bg-white/10 hover:text-white transition-colors"
                onclick={(e) => { e.stopPropagation(); item.action() }}
                role="menuitem"
              >
                <span>{item.label}</span>
                {#if item.shortcut}
                  <span class="text-[10px] text-gray-500 ml-4">{item.shortcut}</span>
                {/if}
              </button>
            {/if}
          {/each}
        </div>
      {/if}
    </div>
  {/each}

  <!-- Spacer -->
  <div class="flex-1"></div>

  <!-- Window title (right side) -->
  <span class="px-3 text-[11px] text-gray-500 font-mono">hermes-web-computer</span>
</div>