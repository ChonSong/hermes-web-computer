/**
 * FileTree.svelte — VSCode-style collapsible file tree
 * Phase 2.1: Wired to Go fs.* backend methods via ws.ts helpers
 * - fs.list (via fsList)
 * - fs.read (via fsRead) → dispatches file:open
 * - fs.write (via fsWrite) → create files
 * - fs.stat (via fsStat)
 * - fs.rename (via fsRename)
 * - fs.delete (via fsDelete)
 */
<script lang="ts">
  import { onMount, createEventDispatcher } from "svelte"
  import { fsList, fsRead, fsWrite, fsStat, fsRename, fsDelete, on } from "../stores/ws"

  interface Entry {
    name: string
    type: "file" | "directory" | "dir"
    size: number
    mod_time: string
  }

  // File type colors by extension
  function fileColor(name: string, type: string): string {
    if (type === "directory" || type === "dir") return "text-purple-400"
    const ext = name.split(".").pop()?.toLowerCase() || ""
    const colors: Record<string, string> = {
      // Images
      png: "text-green-400", jpg: "text-green-400", jpeg: "text-green-400",
      gif: "text-green-400", webp: "text-green-400", svg: "text-green-400",
      // Code
      ts: "text-blue-400", tsx: "text-blue-400", js: "text-yellow-400",
      jsx: "text-yellow-400", vue: "text-green-400", svelte: "text-orange-400",
      css: "text-pink-400", scss: "text-pink-400", html: "text-orange-400",
      // Data
      json: "text-yellow-300", yaml: "text-red-400", yml: "text-red-400",
      toml: "text-red-400", xml: "text-orange-400",
      // Scripts
      sh: "text-green-400", bash: "text-green-400", zsh: "text-green-400",
      py: "text-blue-400", go: "text-cyan-400", rs: "text-orange-400",
      // Docs
      md: "text-gray-300", markdown: "text-gray-300", txt: "text-gray-400",
      // Config
      env: "text-yellow-500", gitignore: "text-gray-400",
      dockerfile: "text-blue-400", makefile: "text-gray-400",
    }
    return colors[ext] || "text-gray-300"
  }

  function fileIcon(name: string, type: string, expanded: boolean): string {
    if (type === "directory" || type === "dir") return expanded ? "📂" : "📁"
    const ext = name.split(".").pop()?.toLowerCase() || ""
    const icons: Record<string, string> = {
      ts: "🔷", tsx: "🔷", js: "🟨", jsx: "🟨",
      vue: "💚", svelte: "🧡", css: "🎨", scss: "🎨",
      html: "🌐", json: "📋", yaml: "📄", yml: "📄",
      md: "📝", markdown: "📝", txt: "📄", pdf: "📕",
      py: "🐍", go: "🔵", rs: "🦀", sh: "💻", bash: "💻",
      png: "🖼️", jpg: "🖼️", jpeg: "🖼️", gif: "🖼️", webp: "🖼️", svg: "🖼️",
      env: "🔐", gitignore: "🔒", dockerfile: "🐳",
    }
    return icons[ext] || "📄"
  }

  // Expanded state: path → boolean
  let expandedPaths = $state<Set<string>>(new Set())
  let currentPath = $state("/")
  let entries = $state<Entry[]>([])
  let loading = $state(false)
  let error = $state<string | null>(null)

  // Context menu state
  let contextMenu = $state<{ x: number; y: number; entry: Entry | null } | null>(null)
  let renaming = $state<string | null>(null) // path being renamed
  let renameInput = $state("")
  let deleteConfirm = $state<string | null>(null) // path being deleted

  // New file dialog
  let showNewFile = $state(false)
  let newFileName = $state("")
  let newFileError = $state("")

  const dispatch = createEventDispatcher<{ "file:open": { path: string } }>()

  // --- Navigation ---

  function navigateTo(path: string) {
    currentPath = path
    fetchEntries(path)
  }

  function navigateUp() {
    const parts = currentPath.split("/").filter(Boolean)
    if (parts.length === 0) {
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

    const pathLower = path.toLowerCase()
    const cleanPath = pathLower === "/workspace" ? "/workspace" : path

    const unsub = on("fs.list.response", (data: unknown) => {
      unsub()
      unsubErr()
      loading = false
      const resp = data as { entries?: Entry[]; error?: string; path?: string }
      if (resp.error) {
        error = resp.error
      } else if (resp.entries) {
        entries = resp.entries.sort((a, b) => {
          if (a.type === b.type) return a.name.localeCompare(b.name)
          return a.type === "directory" || a.type === "dir" ? -1 : 1
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

    fsList(cleanPath)

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
      if (entry.type === "directory" || entry.type === "dir") {
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
    fsList(path)
  }

  function isExpanded(path: string): boolean {
    return expandedPaths.has(path)
  }

  function getChildren(path: string): Entry[] {
    return childEntries.get(path) || []
  }

  // --- Path helpers ---

  function fullPath(entry: Entry): string {
    if (currentPath === "/" || currentPath === "") {
      return `/${entry.name}`
    }
    return `${currentPath}/${entry.name}`
  }

  // --- Click handlers ---

  function handleEntryClick(entry: Entry) {
    if (entry.type === "directory" || entry.type === "dir") {
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
    e.stopPropagation()
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

  // --- Rename ---

  function commitRename() {
    if (!renaming || !renameInput.trim()) {
      renaming = null
      return
    }
    const oldPath = renaming
    const parts = oldPath.split("/").filter(Boolean)
    const oldName = parts[parts.length - 1]
    const newName = renameInput.trim()
    if (newName === oldName) {
      renaming = null
      return
    }
    parts[parts.length - 1] = newName
    const newPath = "/" + parts.join("/")

    const unsub = on("fs.rename.success", (data: unknown) => {
      unsub()
      unsubErr()
      renaming = null
      // Refresh the directory listing
      fetchEntries(currentPath)
    })
    const unsubErr = on("fs.error", (data: unknown) => {
      unsub()
      unsubErr()
      error = (data as { message?: string }).message || "Rename failed"
      renaming = null
    })

    fsRename(oldPath, newPath)

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
    const unsub = on("fs.delete.success", (data: unknown) => {
      unsub()
      unsubErr()
      // Remove from entries
      entries = entries.filter(e => fullPath(e) !== path)
    })
    const unsubErr = on("fs.delete.error", (data: unknown) => {
      unsub()
      unsubErr()
      error = (data as { message?: string }).message || "Delete failed"
    })

    fsDelete(path)

    setTimeout(() => { unsub(); unsubErr() }, 10000)
  }

  function cancelDelete() {
    deleteConfirm = null
  }

  // --- File create ---

  function handleCreateFile() {
    if (!newFileName.trim()) {
      newFileError = "Name required"
      return
    }
    if (newFileName.includes("/") || newFileName.includes("..")) {
      newFileError = "No slashes or .."
      return
    }
    const fullPath = currentPath === "/" ? `/${newFileName}` : `${currentPath}/${newFileName}`

    fsWrite(fullPath, "# New file\n")

    showNewFile = false
    newFileName = ""
    newFileError = ""

    // Refresh directory
    setTimeout(() => fetchEntries(currentPath), 300)
  }

  function openNewFileDialog() {
    showNewFile = true
    newFileName = ""
    newFileError = ""
  }

  // --- Size formatter ---

  function formatSize(bytes: number): string {
    if (bytes === 0) return "0 B"
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
    const handler = () => closeContextMenu()
    document.addEventListener("click", handler)
    return () => document.removeEventListener("click", handler)
  })
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="flex flex-col h-full text-gray-200 text-sm font-mono bg-[#191919]" onclick={closeContextMenu}>
  <!-- Header with breadcrumb + new file button -->
  <div class="flex items-center gap-2 px-3 py-2 border-b border-white/5 shrink-0">
    <button
      class="flex items-center gap-1 px-2 py-1 bg-white/5 hover:bg-white/10 text-gray-400 hover:text-gray-200 rounded text-xs transition-colors shrink-0"
      onclick={(e) => { e.stopPropagation(); openNewFileDialog() }}
      title="New file"
    >
      <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
      New
    </button>
    <div class="flex items-center gap-1 overflow-x-auto whitespace-nowrap flex-1 min-w-0">
      {#each breadcrumbs() as crumb, i}
        <button
          class="text-purple-400 hover:text-purple-300 hover:underline cursor-pointer text-xs shrink-0"
          onclick={(e) => { e.stopPropagation(); navigateTo(crumb || "/") }}
        >
          {crumb === "/" ? "root" : crumb.split("/").pop() || "/"}
        </button>
        {#if i < breadcrumbs().length - 1}
          <span class="text-gray-600 text-xs shrink-0">/</span>
        {/if}
      {/each}
    </div>
  </div>

  <!-- Parent directory button -->
  {#if currentPath !== "/" && currentPath !== ""}
    <div class="px-3 py-1 border-b border-white/5 shrink-0">
      <button
        class="flex items-center gap-2 px-2 py-1 rounded hover:bg-white/5 text-gray-500 hover:text-gray-300 cursor-pointer transition-colors text-xs w-full"
        onclick={(e) => { e.stopPropagation(); navigateUp() }}
      >
        <span class="text-xs">..</span>
        <span class="text-gray-600 text-[10px]">Go up</span>
      </button>
    </div>
  {/if}

  <!-- Loading -->
  {#if loading}
    <div class="flex items-center justify-center py-6 shrink-0">
      <div class="animate-spin h-4 w-4 border-2 border-purple-400 border-t-transparent rounded-full"></div>
      <span class="ml-2 text-gray-400 text-xs">Loading...</span>
    </div>
  {/if}

  <!-- Error -->
  {#if error}
    <div class="flex flex-col items-center justify-center py-6 gap-2 shrink-0">
      <span class="text-red-400 text-xs">⚠ {error}</span>
      <button
        class="px-2 py-1 bg-red-900/30 text-red-300 rounded hover:bg-red-900/50 text-xs cursor-pointer transition-colors"
        onclick={(e) => { e.stopPropagation(); fetchEntries(currentPath) }}
      >
        Retry
      </button>
    </div>
  {/if}

  <!-- File list -->
  {#if !loading && !error}
    <div class="flex-1 overflow-y-auto min-h-0">
      {#if entries.length === 0}
        <div class="flex flex-col items-center justify-center py-8 gap-2">
          <span class="text-gray-600 text-xs">Empty directory</span>
          <button
            class="px-2 py-1 bg-white/5 hover:bg-white/10 text-gray-400 rounded text-xs cursor-pointer transition-colors"
            onclick={(e) => { e.stopPropagation(); openNewFileDialog() }}
          >
            + Create file
          </button>
        </div>
      {:else}
        <ul class="py-1">
          {#each entries as entry (entry.name)}
            {@const path = fullPath(entry)}
            {@const isDir = entry.type === "directory" || entry.type === "dir"}
            {@const childList = isDir ? getChildren(path) : []}
            {@const isExp = isExpanded(path)}
            {@const isRenaming = renaming === path}
            {@const color = fileColor(entry.name, entry.type)}

            <!-- svelte-ignore a11y_click_events_have_key_events -->
            <!-- svelte-ignore a11y_no_static_element_interactions -->
            <li
              class="group flex items-center gap-1.5 px-2 py-0.5 hover:bg-white/5 cursor-pointer transition-colors rounded-sm mx-1"
              onclick={(e) => { e.stopPropagation(); handleEntryClick(entry) }}
              ondblclick={(e) => { e.stopPropagation(); handleEntryDoubleClick(entry) }}
              oncontextmenu={(e) => { e.stopPropagation(); handleContextMenu(e, entry) }}
            >
              <!-- Chevron (dirs only) -->
              <span class="w-3 shrink-0 text-[9px] text-gray-600">
                {#if isDir}{isExp ? "▼" : "▶"}{/if}
              </span>

              <!-- Icon with type color -->
              <span class="text-base shrink-0 {color}">
                {fileIcon(entry.name, entry.type, isExp)}
              </span>

              <!-- Name or rename input -->
              {#if isRenaming}
                <!-- svelte-ignore a11y_autofocus -->
                <input
                  autofocus
                  class="flex-1 bg-gray-800 text-white rounded px-1 text-xs outline-none border border-purple-400 min-w-0"
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
                <span class="flex-1 truncate {color} text-xs {isDir ? 'font-medium' : ''}">
                  {entry.name}
                </span>
              {/if}

              <!-- File size -->
              {#if !isDir}
                <span class="text-gray-600 text-[9px] shrink-0 ml-1">
                  {formatSize(entry.size)}
                </span>
              {/if}
            </li>

            <!-- Children (recursive when expanded) -->
            {#if isDir && isExp && childList.length > 0}
              <ul class="pl-4">
                {#each childList as child (child.name)}
                  {@const childPath = path + "/" + child.name}
                  {@const childIsDir = child.type === "directory" || child.type === "dir"}
                  {@const childIsExp = isExpanded(childPath)}
                  {@const childIsRenaming = renaming === childPath}
                  {@const childColor = fileColor(child.name, child.type)}

                  <!-- svelte-ignore a11y_click_events_have_key_events -->
                  <!-- svelte-ignore a11y_no_static_element_interactions -->
                  <li
                    class="group flex items-center gap-1.5 px-2 py-0.5 hover:bg-white/5 cursor-pointer transition-colors rounded-sm mx-1"
                    onclick={(e) => { e.stopPropagation(); childIsDir ? toggleExpand(child) : dispatch("file:open", { path: childPath }) }}
                    ondblclick={(e) => { e.stopPropagation(); if (!childIsDir) dispatch("file:open", { path: childPath }) }}
                    oncontextmenu={(e) => { e.stopPropagation(); handleContextMenu(e, child) }}
                  >
                    <span class="w-3 shrink-0 text-[9px] text-gray-600">
                      {childIsDir ? (childIsExp ? "▼" : "▶") : ""}
                    </span>
                    <span class="text-base shrink-0 {childColor}">
                      {fileIcon(child.name, child.type, childIsExp)}
                    </span>
                    {#if childIsRenaming}
                      <!-- svelte-ignore a11y_autofocus -->
                      <input
                        autofocus
                        class="flex-1 bg-gray-800 text-white rounded px-1 text-xs outline-none border border-purple-400 min-w-0"
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
                      <span class="flex-1 truncate {childColor} text-xs {childIsDir ? 'font-medium' : ''}">
                        {child.name}
                      </span>
                    {/if}
                    {#if !childIsDir}
                      <span class="text-gray-600 text-[9px] shrink-0 ml-1">
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
    class="fixed z-50 bg-[#1e1e2e] border border-white/10 rounded-xl shadow-xl py-1 min-w-[160px] text-xs"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px"
    onclick={(e) => e.stopPropagation()}
  >
    <button
      class="w-full px-3 py-1.5 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextOpen}
    >
      <span>📄</span> Open
    </button>
    <button
      class="w-full px-3 py-1.5 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextRename}
    >
      <span>✏️</span> Rename
    </button>
    <button
      class="w-full px-3 py-1.5 text-left hover:bg-white/10 text-gray-200 transition-colors flex items-center gap-2"
      onclick={handleContextCopyPath}
    >
      <span>📋</span> Copy Path
    </button>
    <div class="h-px bg-white/10 mx-2 my-1"></div>
    <button
      class="w-full px-3 py-1.5 text-left hover:bg-white/10 text-red-400 transition-colors flex items-center gap-2"
      onclick={handleContextDelete}
    >
      <span>🗑️</span> Delete
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
    <div class="bg-[#1e1e2e] border border-white/10 rounded-2xl p-5 shadow-2xl max-w-sm w-full mx-4">
      <h3 class="text-white font-semibold text-sm mb-2">Delete file?</h3>
      <p class="text-gray-400 text-xs mb-3 break-all font-mono bg-black/20 rounded p-2">{deleteConfirm}</p>
      <p class="text-gray-600 text-[10px] mb-4">This action cannot be undone.</p>
      <div class="flex gap-3 justify-end">
        <button
          class="px-4 py-2 rounded-lg text-gray-300 hover:bg-white/10 transition-colors text-xs cursor-pointer"
          onclick={cancelDelete}
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 rounded-lg bg-red-600 hover:bg-red-500 text-white transition-colors text-xs cursor-pointer"
          onclick={commitDelete}
        >
          Delete
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- New File Dialog -->
{#if showNewFile}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    onclick={(e) => { if (e.target === e.currentTarget) showNewFile = false }}
  >
    <div class="bg-[#1e1e2e] border border-white/10 rounded-2xl p-5 shadow-2xl max-w-sm w-full mx-4">
      <h3 class="text-white font-semibold text-sm mb-3">New File</h3>
      <div class="text-gray-500 text-[10px] mb-2 truncate font-mono bg-black/20 rounded p-2">
        {currentPath}/{newFileName || "filename.ext"}
      </div>
      <input
        autofocus
        bind:value={newFileName}
        onkeydown={(e) => { if (e.key === "Enter") handleCreateFile() }}
        placeholder="filename.ext"
        class="w-full px-3 py-2 rounded-lg bg-gray-800 border border-gray-600 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-purple-400 mb-2"
      />
      {#if newFileError}
        <p class="text-red-400 text-[10px] mb-2">{newFileError}</p>
      {/if}
      <div class="flex gap-3 justify-end mt-3">
        <button
          class="px-4 py-2 rounded-lg text-gray-300 hover:bg-white/10 transition-colors text-xs cursor-pointer"
          onclick={() => showNewFile = false}
        >
          Cancel
        </button>
        <button
          class="px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 text-white transition-colors text-xs cursor-pointer"
          onclick={handleCreateFile}
        >
          Create
        </button>
      </div>
    </div>
  </div>
{/if}