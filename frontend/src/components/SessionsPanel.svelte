<script lang="ts">
  /**
   * SessionsPanel — Left panel tab for browsing, searching,
   * creating, and managing chat sessions.
   * Mirrors hermes-webui session list UI in Svelte 5.
   */
  import { onMount } from "svelte"
  import { sessionStore, type Session } from "../stores/sessions.svelte"
  import { sendOp } from "../stores/ws"
  import { layoutState } from "../stores/layout.svelte"

  let searchQuery = $state("")
  let searchResults = $state<Session[] | null>(null)

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

  onMount(() => {
    sessionStore.refresh()
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
      {#if pinnedSessions.length > 0}
        <div class="text-xs text-gray-500 uppercase tracking-wider px-2 pt-1">Pinned</div>
        {#each pinnedSessions as sess (sess.session_id)}
          <div class="group flex items-center gap-1 px-2 py-1.5 rounded-lg hover:bg-white/5 transition-colors">
            <button
              class="flex-1 text-left flex items-center gap-2 truncate"
              onclick={() => handleSelectSession(sess)}
            >
              <span class="text-yellow-400 shrink-0">📌</span>
              <span class="truncate text-gray-200 text-sm">{sess.title}</span>
              <span class="text-xs text-gray-600 shrink-0">{formatTime(sess.updated_at)}</span>
            </button>
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-white/10 text-gray-500 shrink-0"
              onclick={(e) => handlePin(e, sess)}
              title="Unpin"
            >📌</button>
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-red-500/20 text-gray-500 hover:text-red-400 shrink-0"
              onclick={(e) => handleDelete(e, sess.session_id)}
              title="Delete"
            >✕</button>
          </div>
        {/each}
      {/if}

      {#if recentSessions.length > 0}
        <div class="text-xs text-gray-500 uppercase tracking-wider px-2 pt-1">
          {pinnedSessions.length > 0 ? "Recent" : "All Sessions"}
        </div>
        {#each recentSessions as sess (sess.session_id)}
          <div
            class="group flex items-center gap-1 px-2 py-1.5 rounded-lg transition-colors
                   {sessionStore.activeId === sess.session_id ? 'bg-white/10' : 'hover:bg-white/5'}"
          >
            <button
              class="flex-1 text-left flex items-center gap-2 truncate"
              onclick={() => handleSelectSession(sess)}
            >
              <span class="text-gray-500 shrink-0">💬</span>
              <span class="truncate text-gray-200 text-sm">{sess.title}</span>
              <span class="text-xs text-gray-600 shrink-0">{formatTime(sess.updated_at)}</span>
            </button>
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-yellow-500/20 text-gray-500 shrink-0"
              onclick={(e) => handlePin(e, sess)}
              title="Pin"
            >📌</button>
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-red-500/20 text-gray-500 hover:text-red-400 shrink-0"
              onclick={(e) => handleDelete(e, sess.session_id)}
              title="Delete"
            >✕</button>
          </div>
        {/each}
      {/if}
    {/if}
  </div>
</div>
