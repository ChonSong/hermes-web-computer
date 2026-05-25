<script lang="ts">
  /**
   * SessionsPanel — Left panel tab for browsing, searching,
   * creating, and managing chat sessions.
   * Mirrors hermes-webui session list UI in Svelte 5.
   */
  import { onMount, onDestroy } from "svelte"
  import { sessionStore, type Session } from "../stores/sessions.svelte"
  import { sendOp } from "../stores/ws"
  import { layoutState } from "../stores/layout.svelte"

  // Search
  let searchQuery = $state("")
  let searchResults = $state<Session[] | null>(null)

  // Context menu
  let contextMenu = $state<{ visible: boolean; x: number; y: number; session: Session | null }>({
    visible: false,
    x: 0,
    y: 0,
    session: null,
  })

  // Inline editing
  let editingId = $state<string | null>(null)
  let editingTitle = $state("")

  // Delete confirmation
  let deleteConfirm = $state<{ visible: boolean; session: Session | null }>({
    visible: false,
    session: null,
  })

  // Project colors (6 preset)
  const PROJECT_COLORS = [
    "#ef4444", // red
    "#f97316", // orange
    "#eab308", // yellow
    "#22c55e", // green
    "#3b82f6", // blue
    "#a855f7", // purple
  ]

  function getProjectColor(projectId: string | undefined): string {
    if (!projectId) return "transparent"
    // Simple hash to get consistent color for project
    let hash = 0
    for (let i = 0; i < projectId.length; i++) {
      hash = projectId.charCodeAt(i) + ((hash << 5) - hash)
    }
    return PROJECT_COLORS[Math.abs(hash) % PROJECT_COLORS.length]
  }

  // Group sessions by project_id
  let groupedSessions = $derived.by(() => {
    const sessions = searchQuery ? (searchResults ?? []) : sessionStore.sessions
    const groups = new Map<string | null, Session[]>()
    for (const s of sessions) {
      const pid = s.project_id ?? null
      if (!groups.has(pid)) groups.set(pid, [])
      groups.get(pid)!.push(s)
    }
    return groups
  })

  let filteredSessions = $derived(
    searchQuery
      ? searchResults ?? []
      : sessionStore.sessions
  )

  let pinnedSessions = $derived(filteredSessions.filter(s => s.pinned))
  let recentSessions = $derived(filteredSessions.filter(s => !s.pinned))

  function formatTime(ts: number): string {
    const d = new Date(ts * 1000)
    const now = new Date()
    const diffMs = now.getTime() - d.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMins / 60)
    const diffDays = Math.floor(diffHours / 24)
    if (diffMins < 1) return "just now"
    if (diffMins < 60) return `${diffMins}m ago`
    if (diffHours < 24) return `${diffHours}h ago`
    if (diffDays < 7) return `${diffDays}d ago`
    return d.toLocaleDateString()
  }

  async function handleNewSession() {
    const sess = await sessionStore.create()
    if (sess) {
      openChatTile(sess.session_id)
    }
  }

  function openChatTile(sessionId: string) {
    // Find a splittable leaf node to split
    function findLeaf(node: any, path: string): string | null {
      if (node.type === 'leaf') return path
      if (node.children) {
        for (const child of node.children) {
          const found = findLeaf(child, child.id)
          if (found) return found
        }
      }
      return null
    }
    const targetId = layoutState.tree ? findLeaf(layoutState.tree, layoutState.tree.id) : "root"
    sendOp({
      op: "split",
      target_id: targetId ?? "root",
      direction: "v",
      content: "chat",
      size: 0.5,
    })
  }

  function handleSelectSession(sess: Session) {
    const firstOpen = !sessionStore.activeId
    sessionStore.select(sess.session_id)
    if (firstOpen) {
      openChatTile(sess.session_id)
    }
  }

  async function handleDelete(e: Event, id: string) {
    e.stopPropagation()
    await sessionStore.delete(id)
  }

  async function handlePin(e: Event, sess: Session) {
    e.stopPropagation()
    await sessionStore.pin(sess.session_id, !sess.pinned)
  }

  function handleSearch() {
    if (!searchQuery.trim()) {
      searchResults = null
      return
    }
    const q = searchQuery.toLowerCase()
    searchResults = sessionStore.sessions.filter(s =>
      s.title.toLowerCase().includes(q) ||
      s.session_id.includes(q)
    )
  }

  // Context menu
  function handleContextMenu(e: MouseEvent, sess: Session) {
    e.preventDefault()
    contextMenu = { visible: true, x: e.clientX, y: e.clientY, session: sess }
  }

  function closeContextMenu() {
    contextMenu = { ...contextMenu, visible: false }
  }

  async function handleRename() {
    const sess = contextMenu.session
    if (!sess) return
    editingId = sess.session_id
    editingTitle = sess.title
    closeContextMenu()
  }

  async function handleConfirmRename() {
    if (!editingId) return
    const newTitle = editingTitle.trim()
    if (newTitle && newTitle !== sessionStore.sessions.find(s => s.session_id === editingId)?.title) {
      await sessionStore.updateTitle(editingId, newTitle)
    }
    editingId = null
    editingTitle = ""
  }

  function handleRenameKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") {
      e.preventDefault()
      handleConfirmRename()
    } else if (e.key === "Escape") {
      editingId = null
      editingTitle = ""
    }
  }

  async function handleArchive() {
    const sess = contextMenu.session
    if (!sess) return
    await sessionStore.archive(sess.session_id, !sess.archived)
    closeContextMenu()
  }

  async function handleDuplicate() {
    const sess = contextMenu.session
    if (!sess) return
    await sessionStore.duplicate(sess.session_id)
    closeContextMenu()
  }

  function handleDeleteClick() {
    deleteConfirm = { visible: true, session: contextMenu.session }
    closeContextMenu()
  }

  async function handleConfirmDelete() {
    const sess = deleteConfirm.session
    if (sess) {
      await sessionStore.delete(sess.session_id)
    }
    deleteConfirm = { visible: false, session: null }
  }

  function handleCancelDelete() {
    deleteConfirm = { visible: false, session: null }
  }

  // Click outside to close menus
  function handleWindowClick(e: MouseEvent) {
    if (contextMenu.visible) {
      closeContextMenu()
    }
    if (deleteConfirm.visible) {
      handleCancelDelete()
    }
  }

  onMount(() => {
    sessionStore.refresh()
    window.addEventListener("click", handleWindowClick)
  })

  onDestroy(() => {
    window.removeEventListener("click", handleWindowClick)
  })
</script>

<div class="flex flex-col h-full">
  <!-- New session button -->
  <button
    class="mx-1 mb-2 px-3 py-2 text-sm font-medium rounded-lg
           bg-purple-600 hover:bg-purple-500 text-white
           transition-colors flex items-center gap-2"
    onclick={handleNewSession}
  >
    <span>+</span> New Chat
  </button>

  <!-- Search -->
  <div class="px-1 mb-2">
    <input
      type="text"
      placeholder="Search sessions..."
      class="w-full px-3 py-1.5 text-sm rounded-lg
             bg-white/5 border border-white/10 text-gray-200
             placeholder-gray-500 focus:outline-none focus:border-purple-500/50"
      bind:value={searchQuery}
      oninput={handleSearch}
    />
  </div>

  <!-- Session list -->
  <div class="flex-1 overflow-y-auto space-y-1 px-1">
    {#if sessionStore.loading}
      <div class="text-center py-4 text-gray-500 text-sm">Loading...</div>
    {:else if filteredSessions.length === 0}
      <div class="text-center py-4 text-gray-500 text-sm">No sessions yet</div>
    {:else}
      {#each [...groupedSessions.entries()] as [projectId, sessions] (projectId ?? "no-project")}
        {#if groupedSessions.size > 1}
          <div class="flex items-center gap-2 px-2 pt-3 pb-1">
            <span
              class="w-2.5 h-2.5 rounded-full shrink-0"
              style="background-color: {getProjectColor(projectId ?? undefined)}"
            ></span>
            <span class="text-xs text-gray-400 uppercase tracking-wider">
              {projectId ? projectId.slice(0, 8) : "No Project"}
            </span>
          </div>
        {/if}
        {#each sessions as sess (sess.session_id)}
          <div
            class="group flex items-center gap-1 px-2 py-1.5 rounded-lg transition-colors
                   {sessionStore.activeId === sess.session_id ? 'bg-white/10' : 'hover:bg-white/5'}"
            onclick={() => handleSelectSession(sess)}
            oncontextmenu={(e) => handleContextMenu(e, sess)}
          >
            <!-- Project color dot -->
            {#if projectId}
              <span
                class="w-2 h-2 rounded-full shrink-0"
                style="background-color: {getProjectColor(projectId ?? undefined)}"
              ></span>
            {:else}
              <span class="w-2 h-2 rounded-full shrink-0 bg-gray-600"></span>
            {/if}

            <!-- Pinned icon -->
            {#if sess.pinned}
              <span class="text-yellow-400 shrink-0 text-xs">📌</span>
            {:else}
              <span class="text-gray-600 shrink-0 text-xs">💬</span>
            {/if}

            <!-- Title (inline edit) -->
            {#if editingId === sess.session_id}
              <input
                type="text"
                class="flex-1 bg-white/10 px-1 py-0.5 text-sm text-gray-200 rounded
                       outline-none focus:border-purple-500/50 border border-transparent"
                bind:value={editingTitle}
                onkeydown={handleRenameKeydown}
                onblur={handleConfirmRename}
                autofocus
                onclick={(e) => e.stopPropagation()}
              />
            {:else}
              <span class="flex-1 truncate text-gray-200 text-sm">{sess.title}</span>
            {/if}

            <!-- Timestamp -->
            <span class="text-xs text-gray-600 shrink-0">{formatTime(sess.updated_at)}</span>

            <!-- Inline pin/unpin -->
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-white/10 text-gray-500 shrink-0"
              onclick={(e) => { e.stopPropagation(); handlePin(e, sess) }}
              title={sess.pinned ? "Unpin" : "Pin"}
            >{sess.pinned ? "📌" : "○"}</button>

            <!-- Inline delete -->
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-red-500/20 text-gray-500 hover:text-red-400 shrink-0"
              onclick={(e) => { e.stopPropagation(); handleDelete(e, sess.session_id) }}
              title="Delete"
            >✕</button>
          </div>
        {/each}
      {/each}
    {/if}
  </div>
</div>

<!-- Context Menu -->
{#if contextMenu.visible}
  <div
    class="fixed z-50 bg-gray-900 border border-white/10 rounded-lg shadow-xl py-1 min-w-40"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px"
    onclick={(e) => e.stopPropagation()}
  >
    <button
      class="w-full px-3 py-1.5 text-sm text-gray-200 hover:bg-white/10 text-left flex items-center gap-2"
      onclick={handleRename}
    >
      <span>✏️</span> Rename
    </button>
    <button
      class="w-full px-3 py-1.5 text-sm text-gray-200 hover:bg-white/10 text-left flex items-center gap-2"
      onclick={handleDuplicate}
    >
      <span>📋</span> Duplicate
    </button>
    <button
      class="w-full px-3 py-1.5 text-sm text-gray-200 hover:bg-white/10 text-left flex items-center gap-2"
      onclick={handleArchive}
    >
      <span>📦</span> {contextMenu.session?.archived ? "Unarchive" : "Archive"}
    </button>
    <button
      class="w-full px-3 py-1.5 text-sm text-gray-200 hover:bg-white/10 text-left flex items-center gap-2"
      onclick={(e) => { e.stopPropagation(); handlePin(e as any, contextMenu.session!) }}
    >
      <span>📌</span> {contextMenu.session?.pinned ? "Unpin" : "Pin"}
    </button>
    <div class="border-t border-white/10 my-1"></div>
    <button
      class="w-full px-3 py-1.5 text-sm text-red-400 hover:bg-red-500/10 text-left flex items-center gap-2"
      onclick={handleDeleteClick}
    >
      <span>🗑️</span> Delete
    </button>
  </div>
{/if}

<!-- Delete Confirmation Dialog -->
{#if deleteConfirm.visible}
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
    onclick={handleCancelDelete}
  >
    <div
      class="bg-gray-900 border border-white/10 rounded-xl p-6 max-w-sm w-full mx-4 shadow-2xl"
      onclick={(e) => e.stopPropagation()}
    >
      <h3 class="text-lg font-medium text-gray-200 mb-2">Delete Session</h3>
      <p class="text-sm text-gray-400 mb-4">
        Delete "{deleteConfirm.session?.title}"? This cannot be undone.
      </p>
      <div class="flex gap-3 justify-end">
        <button
          class="px-4 py-2 text-sm text-gray-300 hover:bg-white/10 rounded-lg transition-colors"
          onclick={handleCancelDelete}
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 text-sm bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors"
          onclick={handleConfirmDelete}
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}
