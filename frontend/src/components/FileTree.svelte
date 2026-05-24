/**
 * FileTree.svelte — VSCode-style collapsible file tree
 * Phase 5: collapsible tree with ▶/▼ chevrons, context menu
 * Spec: docs/WAYBAR-SPEC.md §5
 */
<script lang="ts">
  import { onMount, createEventDispatcher } from "svelte"
  import { send, on } from "../stores/ws"

  interface Entry {
    name: string
    type: "file" | "directory"
    size: number
    mod_time: string
  }

  // Expanded state: path → boolean
  let expandedPaths = $state<Set<string>>(new Set())
  let currentPath = $state("/opt/data/hermes-web-computer")
  let entries = $state<Entry[]>([])
  let loading = $state(false)
  let error = $state<string | null>(null)

  // Context menu state
  let contextMenu = $state<{ x: number; y: number; entry: Entry | null } | null>(null)
  let renaming = $state<string | null>(null) // path being renamed
  let renameInput = $state("")
  let deleteConfirm = $state<string | null>(null) // path being deleted

  const dispatch = createEventDispatcher<{ "file:open": { path: string } }>()

  // --- Navigation ---

  function navigateTo(path: string) {
    currentPath = path
    fetchEntries(path)
  }

  function navigateUp() {
    const parts = currentPath.split("/").filter(Boolean)
    if (parts.length <= 1) {
      navigateTo("/")
    } else {
      parts.pop()
      navigateTo("/" + parts.join("/"))
    }
  }

  // --- Fetch entries ---

  function fetchEntries(path: string) {
    loading = true
    error = null
    entries = []

    const unsub = on("fs.list.response", (data: unknown) => {
      unsub()
      unsubErr()
      loading = false
      const resp = data as { entries?: Entry[]; error?: string }
      if (resp.error) {
        error = resp.error
      } else if (resp.entries) {
        entries = resp.entries.sort((a, b) => {
          if (a.type === b.type) return a.name.localeCompare(b.name)
          return a.type === "directory" ? -1 : 1
        })
      }
    })

    const unsubErr = on("fs.error", (data: unknown) => {
      unsub()
      unsubErr()
      loading = false
      const resp = data as { message?: string }
      error = resp.message || "Unknown error"
    })

    send({ protocol: "ui", method: "fs.list", params: { path } })

    setTimeout(() => {
      if (loading) {
        unsub()
        unsubErr()
        loading = false
        error = "Request timed out"
      }
    }, 10_000)
  }

  // --- Expand/collapse ---

  function toggleExpand(entry: Entry) {
    const path = fullPath(entry)
    const newExpanded = new Set(expandedPaths)
    if (newExpanded.has(path)) {
      newExpanded.delete(path)
    } else {
      newExpanded.add(path)
      // If it's a directory, fetch its children
      if (entry.type === "directory") {
        fetchChildren(path)
      }
    }
    expandedPaths = newExpanded
  }

  let childEntries = $state<Map<string, Entry[]>>(new Map())

  function fetchChildren(path: string) {
    const unsub = on("fs.list.response", (data: unknown) => {
      unsub()
      unsubErr()
      const resp = data as { entries?: Entry[]; path?: string }
      if (resp.entries && resp.path) {
        childEntries = new Map(childEntries).set(resp.path, resp.entries)
      }
    })
    const unsubErr = on("fs.error", () => { unsub(); unsubErr() })
    send({ protocol: "ui", method: "fs.list", params: { path } })
  }

  function isExpanded(path: string): boolean {
    return expandedPaths.has(path)
  }

  function getChildren(path: string): Entry[] {
    return childEntries.get(path) || []
  }

  // --- Path helpers ---

  function fullPath(entry: Entry): string {
    return currentPath === "/" ? `/${entry.name}` : `${currentPath}/${entry.name}`
  }

  // --- Click handlers ---

  function handleEntryClick(entry: Entry) {
    if (entry.type === "directory") {
      toggleExpand(entry)
    } else {
      dispatch("file:open", { path: fullPath(entry) })
    }
  }

  function handleEntryDoubleClick(entry: Entry) {
    if (entry.type === "file") {
      dispatch("file:open", { path: fullPath(entry) })
    }
  }

  // --- Context menu ---

  function handleContextMenu(e: MouseEvent, entry: Entry) {
    e.preventDefault()
    contextMenu = { x: e.clientX, y: e.clientY, entry }
  }

  function closeContextMenu() {
    contextMenu = null
  }

  function handleContextOpen() {
    closeContextMenu()
    if (contextMenu?.entry) {
      const entry = contextMenu.entry
      if (entry.type === "file") {
        dispatch("file:open", { path: fullPath(entry) })
      } else {
        navigateTo(fullPath(entry))
      }
    }
  }

  function handleContextRename() {
    closeContextMenu()
    if (contextMenu?.entry) {
      renaming = fullPath(contextMenu.entry)
      renameInput = contextMenu.entry.name
    }
  }

  function handleContextDelete() {
    closeContextMenu()
    if (contextMenu?.entry) {
      deleteConfirm = fullPath(contextMenu.entry)
    }
  }

  function handleContextCopyPath() {
    closeContextMenu()
    if (contextMenu?.entry) {
      navigator.clipboard.writeText(fullPath(contextMenu.entry)).catch(() => {})
    }
  }

  function handleContextOpenInTerminal() {
    closeContextMenu()
    // This would dispatch a terminal open event
    // For now, open the containing directory in the tree
    if (contextMenu?.entry) {
      const parent = currentPath
      navigateTo(parent)
    }
  }

  // --- Rename ---

  function commitRename() {
    if (!renaming || !renameInput.trim()) {
      renaming = null
      return
    }
    const dir = currentPath
    const oldName = renaming.split("/").filter(Boolean).pop() || ""
    const newName = renameInput.trim()
    if (newName === oldName) {
      renaming = null
      return
    }
    // Send rename via backend
    const unsub = on("fs.rename.response", () => { unsub(); renaming = null })
    const unsubErr = on("fs.error", (data: unknown) => {
      unsub(); unsubErr()
      error = (data as { message?: string }).message || "Rename failed"
      renaming = null
    })
    send({ protocol: "ui", method: "fs.rename", params: { path: renaming, new_name: newName } })
    setTimeout(() => { unsub(); unsubErr(); renaming = null }, 10000)
  }

  function cancelRename() {
    renaming = null
  }

  // --- Delete ---

  function commitDelete() {
    if (!deleteConfirm) return
    const path = deleteConfirm
    deleteConfirm = null
    const unsub = on("fs.delete.success", () => {
      unsub()
      // Remove from entries
      entries = entries.filter(e => fullPath(e) !== path)
    })
    const unsubErr = on("fs.delete.error", (data: unknown) => {
      unsub(); unsubErr()
      error = (data as { message?: string }).message || "Delete failed"
    })
    send({ protocol: "ui", method: "fs.delete", params: { path } })
    setTimeout(() => { unsub(); unsubErr() }, 10000)
  }

  function cancelDelete() {
    deleteConfirm = null
  }

  // --- Drag and drop ---

  function handleDragStart(e: DragEvent, entry: Entry) {
    if (entry.type !== "file") return
    e.dataTransfer?.setData("text/plain", fullPath(entry))
    e.dataTransfer!.effectAllowed = "copy"
    setTimeout(() => (e.target as HTMLElement)?.classList.add("opacity-50"), 0)
  }

  function handleDragEnd(e: DragEvent) {
    (e.target as HTMLElement)?.classList.remove("opacity-50")
  }

  // --- Size formatter ---

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  function breadcrumbs(): string[] {
    const parts = currentPath.split("/").filter(Boolean)
    const crumbs: string[] = ["/"]
    let acc = ""
    for (const p of parts) {
      acc += `/${p}`
      crumbs.push(acc)
    }
    return crumbs
  }

  onMount(() => {
    fetchEntries(currentPath)
    // Close context menu on outside click
    const handler = () => closeContextMenu()
    document.addEventListener("click", handler)
    return () => document.removeEventListener("click", handler)
  })
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="flex flex-col h-full text-gray-200 text-sm font-mono" onclick={closeContextMenu}>
  <!-- Breadcrumb Navigation -->
  <div class="flex items-center gap-1 px-3 py-2 border-b border-white/5 overflow-x-auto whitespace-nowrap shrink-0">
    {#each breadcrumbs() as crumb, i}
      <button
        class="text-purple-400 hover:text-purple-300 hover:underline cursor-pointer"
        onclick={(e) => { e.stopPropagation(); navigateTo(crumb || "/") }}
      >
        {crumb || "/"}
      </button>
      {#if i < breadcrumbs().length - 1}
        <span class="text-gray-500">/</span>
      {/if}
    {/each}
  </div>

  <!-- Parent Directory Button -->
  {#if currentPath !== "/"}
    <div class="px-3 py-1 border-b border-white/5 shrink-0">
      <button
        class="flex items-center gap-1 px-2 py-1 rounded-lg hover:bg-white/5 text-gray-400 hover:text-gray-200 cursor-pointer transition-colors"
        onclick={(e) => { e.stopPropagation(); navigateUp() }}
      >
        <span>..</span>
        <span class="text-xs text-gray-500">Parent Directory</span>
      </button>
    </div>
  {/if}

  <!-- Loading Spinner -->
  {#if loading}
    <div class="flex items-center justify-center py-8 shrink-0">
      <div class="animate-spin h-5 w-5 border-2 border-purple-400 border-t-transparent rounded-full"></div>
      <span class="ml-2 text-gray-400">Loading...</span>
    </div>
  {/if}

  <!-- Error Message -->
  {#if error}
    <div class="flex items-center justify-center py-8 shrink-0">
      <span class="text-red-400">⚠ {error}</span>
      <button
        class="ml-3 px-2 py-1 bg-red-900/50 text-red-300 rounded hover:bg-red-900/80 cursor-pointer"
        onclick={(e) => { e.stopPropagation(); fetchEntries(currentPath) }}
      >
        Retry
      </button>
    </div>
  {/if}

  <!-- File List -->
  {#if !loading && !error}
    <div class="flex-1 overflow-y-auto min-h-0">
      {#if entries.length === 0}
        <div class="flex items-center justify-center py-8 text-gray-500">
          Empty directory
        </div>
      {:else}
        <ul class="py-1">
          {#each entries as entry (entry.name)}
            {@const path = fullPath(entry)}
            {@const isDir = entry.type === "directory"}
            {@const childList = isDir ? getChildren(path) : []}
            {@const isExp = isExpanded(path)}
            {@const isRenaming = renaming === path}

            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <li
              class="group flex items-center gap-1 px-2 py-0.5 hover:bg-white/5 cursor-pointer transition-colors rounded-sm mx-1"
              onclick={(e) => { e.stopPropagation(); handleEntryClick(entry) }}
              ondblclick={(e) => { e.stopPropagation(); handleEntryDoubleClick(entry) }}
              oncontextmenu={(e) => { e.stopPropagation(); handleContextMenu(e, entry) }}
              draggable={entry.type === "file"}
              ondragstart={(e) => handleDragStart(e, entry)}
              ondragend={handleDragEnd}
            >
              <!-- Expand/collapse chevron (directories only) -->
              <span class="w-4 shrink-0 text-[10px] text-gray-500">
                {#if isDir}
                  {isExp ? "▼" : "▶"}
                {/if}
              </span>

              <!-- Icon -->
              <span class="text-base shrink-0">
                {#if isDir}
                  {isExp ? "📂" : "📁"}
                {:else}
                  📄
                {/if}
              </span>

              <!-- Name (or rename input) -->
              {#if isRenaming}
                <!-- svelte-ignore a11y_autofocus -->
                <input
                  autofocus
                  class="flex-1 bg-gray-800 text-white rounded px-1 text-sm outline-none border border-purple-400"
                  type="text"
                  bind:value={renameInput}
                  onkeydown={(e) => {
                    if (e.key === "Enter") commitRename()
                    if (e.key === "Escape") cancelRename()
                  }}
                  onblur={() => commitRename()}
                  onclick={(e) => e.stopPropagation()}
                />
              {:else}
                <span class="flex-1 truncate {isDir ? "text-purple-300 font-medium" : "text-gray-300"}">
                  {entry.name}
                </span>
              {/if}

              <!-- File size -->
              {#if entry.type === "file"}
                <span class="text-gray-500 text-xs shrink-0 ml-auto">
                  {formatSize(entry.size)}
                </span>
              {/if}
            </li>

            <!-- Children (recursive when expanded) -->
            {#if isDir && isExp && childList.length > 0}
              <ul class="pl-4">
                {#each childList as child (child.name)}
                  {@const childPath = path + "/" + child.name}
                  {@const childIsDir = child.type === "directory"}
                  {@const childIsExp = isExpanded(childPath)}
                  {@const childIsRenaming = renaming === childPath}

                  <!-- svelte-ignore a11y_click_events_have_key_events -->
                  <!-- svelte-ignore a11y_no_static_element_interactions -->
                  <li
                    class="group flex items-center gap-1 px-2 py-0.5 hover:bg-white/5 cursor-pointer transition-colors rounded-sm mx-1"
                    onclick={(e) => { e.stopPropagation(); childIsDir ? toggleExpand(child) : dispatch("file:open", { path: childPath }) }}
                    ondblclick={(e) => { e.stopPropagation(); if (!childIsDir) dispatch("file:open", { path: childPath }) }}
                    oncontextmenu={(e) => { e.stopPropagation(); handleContextMenu(e, child) }}
                    draggable={child.type === "file"}
                    ondragstart={(e) => handleDragStart(e, child)}
                    ondragend={handleDragEnd}
                  >
                    <span class="w-4 shrink-0 text-[10px] text-gray-500">
                      {childIsDir ? (childIsExp ? "▼" : "▶") : ""}
                    </span>
                    <span class="text-base shrink-0">
                      {childIsDir ? (childIsExp ? "📂" : "📁") : "📄"}
                    </span>
                    {#if childIsRenaming}
                      <!-- svelte-ignore a11y_autofocus -->
                      <input
                        autofocus
                        class="flex-1 bg-gray-800 text-white rounded px-1 text-sm outline-none border border-purple-400"
                        type="text"
                        bind:value={renameInput}
                        onkeydown={(e) => {
                          if (e.key === "Enter") commitRename()
                          if (e.key === "Escape") cancelRename()
                        }}
                        onblur={() => commitRename()}
                        onclick={(e) => e.stopPropagation()}
                      />
                    {:else}
                      <span class="flex-1 truncate {childIsDir ? "text-purple-300 font-medium" : "text-gray-300"}">
                        {child.name}
                      </span>
                    {/if}
                    {#if child.type === "file"}
                      <span class="text-gray-500 text-xs shrink-0 ml-auto">
                        {formatSize(child.size)}
                      </span>
                    {/if}
                  </li>
                {/each}
              </ul>
            {/if}
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>

<!-- Context Menu -->
{#if contextMenu}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed z-50 bg-[#1e1e2e] border border-white/10 rounded-xl shadow-xl py-1 min-w-[160px] text-sm"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px"
    onclick={(e) => e.stopPropagation()}
  >
    <button
      class="w-full px-4 py-2 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextOpen}
    >
      📄 Open
    </button>
    <button
      class="w-full px-4 py-2 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextOpenInTerminal}
    >
      🖥️ Open in Terminal
    </button>
    <div class="h-px bg-white/10 mx-2 my-1"></div>
    <button
      class="w-full px-4 py-2 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextRename}
    >
      ✏️ Rename
    </button>
    <button
      class="w-full px-4 py-2 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextCopyPath}
    >
      📋 Copy Path
    </button>
    <div class="h-px bg-white/10 mx-2 my-1"></div>
    <button
      class="w-full px-4 py-2 text-left hover:bg-white/10 text-red-400 transition-colors flex items-center gap-2"
      onclick={handleContextDelete}
    >
      🗑️ Delete
    </button>
  </div>
{/if}

<!-- Delete Confirmation Modal -->
{#if deleteConfirm}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    onclick={(e) => { if (e.target === e.currentTarget) cancelDelete() }}
  >
    <div class="bg-[#1e1e2e] border border-white/10 rounded-2xl p-6 shadow-2xl max-w-sm w-full mx-4">
      <h3 class="text-white font-semibold mb-2">Delete file?</h3>
      <p class="text-gray-400 text-sm mb-4 break-all font-mono">{deleteConfirm}</p>
      <p class="text-gray-500 text-xs mb-4">This action cannot be undone.</p>
      <div class="flex gap-3 justify-end">
        <button
          class="px-4 py-2 rounded-lg text-gray-300 hover:bg-white/10 transition-colors text-sm"
          onclick={cancelDelete}
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 rounded-lg bg-red-600 hover:bg-red-500 text-white transition-colors text-sm"
          onclick={commitDelete}
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}