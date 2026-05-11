<script lang="ts">
  /**
   * DashFileManager — Migrated from agent-os FileExplorerPage.tsx
   * Browse and edit files via WebSocket API.
   */
  import { fsList, fsRead, fsWrite, send, on } from "../stores/ws"

  interface DirEntry {
    name: string
    type: "file" | "dir"
    size: number
    mtime: string | null
  }

  interface FileContent {
    path: string
    content: string
    size: number
    mtime: string | null
    is_binary: boolean
  }

  let cwd = $state("/opt/data")
  let entries = $state<DirEntry[]>([])
  let loading = $state(false)
  let error = $state<string | null>(null)
  let selected = $state<string | null>(null)
  let preview = $state<FileContent | null>(null)
  let editing = $state(false)
  let editContent = $state("")
  let editDirty = $state(false)
  let showNewFile = $state(false)
  let newName = $state("")
  let newError = $state("")
  let saving = $state(false)
  let deleteTarget = $state<{ name: string; path: string; type: "file" | "dir" } | null>(null)

  function formatSize(bytes: number): string {
    if (bytes === 0) return "—"
    if (bytes < 1024) return `${bytes}B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`
    return `${(bytes / 1024 / 1024).toFixed(1)}MB`
  }

  function formatAge(iso: string | null): string {
    if (!iso) return "—"
    const diff = Date.now() - new Date(iso).getTime()
    if (diff < 60000) return "just now"
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
    return `${Math.floor(diff / 86400000)}d ago`
  }

  async function load(dir: string) {
    loading = true
    error = null
    preview = null
    selected = null
    editing = false
    try {
      fsList(dir)
      // Results come via WS event
    } catch (e) {
      error = String(e)
      entries = []
    } finally {
      loading = false
    }
  }

  function handleFsListResult(data: any) {
    if (data?.entries) {
      const sorted = [...data.entries].sort((a, b) => {
        if (a.type !== b.type) return a.type === "dir" ? -1 : 1
        return a.name.localeCompare(b.name)
      })
      entries = sorted
      cwd = data.path ?? cwd
    }
  }

  function navigate(dir: string) {
    load(dir)
  }

  function navigateUp() {
    const parts = cwd.split("/").filter(Boolean)
    if (parts.length <= 1) { load("/"); return }
    parts.pop()
    load("/" + parts.join("/"))
  }

  async function openFile(name: string) {
    const fullPath = cwd === "/" ? name : `${cwd}/${name}`
    try {
      fsRead(fullPath)
    } catch (e) {
      console.error("Failed to open:", e)
    }
  }

  function handleFsReadResult(data: any) {
    if (data?.path) {
      preview = data
      selected = data.path.split("/").pop() ?? data.path
      editing = false
      editDirty = false
      editContent = data.content ?? ""
    }
  }

  async function handleSave() {
    if (!selected || !preview) return
    const fullPath = cwd === "/" ? selected : `${cwd}/${selected}`
    saving = true
    try {
      fsWrite(fullPath, editContent)
      editDirty = false
      editing = false
      showToastMessage(`Saved ${selected}`, "success")
    } catch (e) {
      showToastMessage(`Save failed: ${e}`, "error")
    } finally {
      saving = false
    }
  }

  async function handleCreateFile() {
    if (!newName.trim()) { newError = "Name required"; return }
    if (newName.includes("/") || newName.includes("..")) { newError = "No slashes or .."; return }
    const fullPath = cwd === "/" ? `/${newName}` : `${cwd}/${newName}`
    try {
      fsWrite(fullPath, "# New file\n")
      showNewFile = false
      newName = ""
      newError = ""
      load(cwd)
    } catch (e: unknown) {
      newError = (e as Error).message ?? "Failed to create file"
    }
  }

  async function handleDelete(name: string, type: "file" | "dir") {
    const fullPath = cwd === "/" ? name : `${cwd}/${name}`
    try {
      send({ protocol: "ui", method: "fs.delete", params: { path: fullPath } })
      showToastMessage(`Deleted ${name}`, "success")
      deleteTarget = null
      if (selected === name) { preview = null; selected = null }
      load(cwd)
    } catch (e) {
      showToastMessage(`Delete failed: ${e}`, "error")
    }
  }

  // Toast
  let toastMessage = $state("")
  let toastType = $state<"success" | "error" | "">("")
  let toastTimer: ReturnType<typeof setTimeout> | null = null

  function showToastMessage(msg: string, type: "success" | "error") {
    toastMessage = msg
    toastType = type
    if (toastTimer) clearTimeout(toastTimer)
    toastTimer = setTimeout(() => { toastMessage = ""; toastType = "" }, 3000)
  }

  // Breadcrumb parts
  function breadcrumbParts() {
    return cwd.split("/").filter(Boolean)
  }

  // Setup WS listeners
  on("fs.list.result", handleFsListResult)
  on("fs.read.result", handleFsReadResult)
</script>

<div class="flex flex-col h-full overflow-hidden bg-gray-950">
  
  <div class="flex items-center justify-between px-4 py-3 border-b border-gray-800 shrink-0">
    <div class="flex-1 min-w-0">
      <h2 class="text-sm font-bold text-gray-200">File Explorer</h2>
      <div class="mt-0.5 flex items-center gap-1 text-[10px] text-gray-500 flex-wrap">
        {#each breadcrumbParts() as part, i}
          {#if i > 0}<span class="text-gray-700">/</span>{/if}
          <button
            onclick={() => navigate("/" + breadcrumbParts().slice(0, i + 1).join("/"))}
            class="hover:text-gray-300 transition-colors {i === breadcrumbParts().length - 1 ? 'text-gray-300 font-medium' : ''}"
          >{part}</button>
        {/each}
      </div>
    </div>
    <div class="flex items-center gap-1.5 ml-3 shrink-0">
      <button
        onclick={() => { showNewFile = true; newName = ""; newError = "" }}
        class="flex items-center gap-1 px-2.5 py-1 bg-blue-600 hover:bg-blue-500 rounded text-[10px] font-medium text-white transition-colors"
      >
        <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
        New
      </button>
      <button
        onclick={navigateUp}
        class="p-1.5 rounded bg-gray-800 hover:bg-gray-700 text-gray-400 hover:text-gray-200 transition-all"
        title="Go up"
      >
        <svg class="w-3.5 h-3.5 rotate-180" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
      </button>
      <button
        onclick={() => load(cwd)}
        class="flex items-center gap-1 px-2.5 py-1 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded text-[10px] text-gray-400 transition-all"
      >
        <svg class="w-3 h-3 {loading ? 'animate-spin' : ''}" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16"/><path d="M8 16H3v5"/></svg>
        Refresh
      </button>
    </div>
  </div>

  <div class="flex-1 flex overflow-hidden">
    
    <div class="flex-1 overflow-y-auto p-3 {preview ? 'border-r border-gray-800' : ''}">
      {#if loading && entries.length === 0}
        <div class="flex items-center justify-center h-full">
          <svg class="w-5 h-5 animate-spin text-gray-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
        </div>
      {:else if error}
        <div class="flex flex-col items-center justify-center h-full gap-2">
          <p class="text-[10px] text-red-400">{error}</p>
          <button onclick={() => load(cwd)} class="text-[10px] text-gray-500 hover:text-gray-300">Retry</button>
        </div>
      {:else if entries.length === 0 && !loading}
        <div class="flex flex-col items-center justify-center h-full gap-2 text-gray-500">
          <svg class="w-6 h-6 opacity-20" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>
          <p class="text-[10px]">Empty directory</p>
        </div>
      {:else}
        <div class="space-y-0.5">
          {#each entries as entry}
            <div
              class="flex items-center gap-2 px-2 py-1 rounded hover:bg-gray-800/50 cursor-pointer transition-colors group {selected === entry.name ? 'bg-gray-800/50' : ''}"
            >
              <div
                class="flex-1 flex items-center gap-2 min-w-0"
                onclick={() => entry.type === "dir"
                  ? navigate(cwd === "/" ? `/${entry.name}` : `${cwd}/${entry.name}`)
                  : openFile(entry.name)
                }
              >
                {#if entry.type === "dir"}
                  <svg class="w-3.5 h-3.5 text-amber-500 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>
                {:else}
                  <svg class="w-3.5 h-3.5 text-gray-500 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/></svg>
                {/if}
                <span class="text-[10px] text-gray-400 group-hover:text-gray-200 truncate">{entry.name}</span>
              </div>
              <span class="text-[9px] text-gray-600 shrink-0 hidden group-hover:block">
                {entry.type === "dir" ? "dir" : formatSize(entry.size)}
              </span>
              <span class="text-[9px] text-gray-600 shrink-0 w-12 text-right hidden group-hover:block">
                {formatAge(entry.mtime)}
              </span>
              {#if entry.type === "file"}
                <button
                  onclick={(e) => {
                    e.stopPropagation()
                    const fullPath = cwd === "/" ? entry.name : `${cwd}/${entry.name}`
                    deleteTarget = { name: entry.name, path: fullPath, type: "file" }
                  }}
                  class="p-0.5 rounded opacity-0 group-hover:opacity-100 hover:bg-red-500/20 text-gray-500 hover:text-red-400 shrink-0 transition-all"
                  title="Delete"
                >
                  <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
                </button>
              {/if}
              {#if entry.type === "dir"}
                <svg class="w-3 h-3 text-gray-600 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m9 18 6-6-6-6"/></svg>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    
    {#if preview}
      <div class="w-80 lg:w-96 xl:w-[32rem] shrink-0 flex flex-col overflow-hidden">
        
        <div class="flex items-center justify-between px-3 py-1.5 border-b border-gray-800 shrink-0">
          <div class="flex items-center gap-1.5 min-w-0">
            <svg class="w-3 h-3 text-gray-500 shrink-0" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/></svg>
            <span class="text-[10px] text-gray-400 truncate">{selected}</span>
            {#if editDirty}<span class="text-[8px] text-amber-500">● unsaved</span>{/if}
          </div>
          <div class="flex items-center gap-0.5 shrink-0">
            {#if !editing}
              <button
                onclick={() => { editing = true; editDirty = false }}
                class="px-1.5 py-0.5 rounded text-[9px] text-gray-400 hover:text-gray-200 hover:bg-gray-800 transition-colors"
              >Edit</button>
              <button
                onclick={() => {
                  const fullPath = cwd === "/" ? selected! : `${cwd}/${selected}`
                  deleteTarget = { name: selected!, path: fullPath, type: "file" }
                }}
                class="p-1 rounded text-gray-500 hover:text-red-400 hover:bg-red-500/20 transition-colors"
                title="Delete"
              >
                <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
              </button>
            {:else}
              <button
                onclick={() => { editing = false; editContent = preview?.content ?? ""; editDirty = false }}
                class="px-1.5 py-0.5 rounded text-[9px] text-gray-400 hover:text-gray-200 hover:bg-gray-800 transition-colors"
              >Cancel</button>
              <button
                onclick={handleSave}
                disabled={saving || !editDirty}
                class="flex items-center gap-0.5 px-2 py-0.5 rounded text-[9px] font-medium bg-blue-600 hover:bg-blue-500 text-white transition-colors disabled:opacity-40"
              >
                {#if saving}
                  <svg class="w-3 h-3 animate-spin" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
                {:else}
                  <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"/><polyline points="17 21 17 13 7 13 7 21"/><polyline points="7 3 7 8 15 8"/></svg>
                {/if}
                Save
              </button>
            {/if}
            <button
              onclick={() => { preview = null; selected = null; editing = false }}
              class="ml-0.5 p-1 rounded text-gray-500 hover:text-gray-300 hover:bg-gray-800 transition-colors"
            >
              <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
            </button>
          </div>
        </div>

        
        <div class="flex-1 overflow-auto p-3">
          {#if editing}
            <textarea
              bind:value={editContent}
              oninput={(e) => { editDirty = (e.target as HTMLTextAreaElement).value !== preview?.content }}
              class="w-full h-full min-h-48 resize-none rounded bg-gray-900 border border-gray-700 text-[10px] text-gray-200 font-mono leading-relaxed p-2 focus:outline-none focus:border-blue-500"
              spellcheck={false}
            />
          {:else}
            <pre class="text-[10px] text-gray-400 font-mono whitespace-pre-wrap break-all leading-relaxed">{preview.content}</pre>
          {/if}
        </div>

        
        <div class="px-3 py-1.5 border-t border-gray-800 shrink-0">
          <span class="text-[9px] text-gray-600">
            {formatSize(preview.size)} · {preview.mtime ? new Date(preview.mtime).toLocaleString() : "—"}
          </span>
        </div>
      </div>
    {/if}
  </div>

  
  {#if showNewFile}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onclick={(e) => { if (e.target === e.currentTarget) showNewFile = false }}>
      <div class="w-80 rounded-lg border border-gray-700 bg-gray-900 shadow-2xl">
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-700">
          <div class="flex items-center gap-1.5">
            <svg class="w-3.5 h-3.5 text-blue-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
            <span class="text-[10px] font-medium text-gray-200">New File</span>
          </div>
          <button onclick={() => showNewFile = false} class="text-gray-500 hover:text-gray-300">
            <svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
          </button>
        </div>
        <div class="p-4 space-y-3">
          <div class="flex items-center gap-1.5 text-[9px] text-gray-500">
            <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z"/></svg>
            <span class="truncate">{cwd}/</span>
          </div>
          <input
            bind:value={newName}
            onkeydown={(e) => { if (e.key === "Enter") handleCreateFile() }}
            placeholder="filename.txt"
            class="w-full px-2.5 py-1.5 rounded bg-gray-800 border border-gray-700 text-[10px] text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-500"
          />
          {#if newError}<p class="text-[9px] text-red-400">{newError}</p>{/if}
          <div class="flex justify-end gap-1.5">
            <button onclick={() => showNewFile = false} class="px-3 py-1 rounded text-[9px] text-gray-500 hover:text-gray-200 transition-colors">Cancel</button>
            <button onclick={handleCreateFile} class="flex items-center gap-1 px-3 py-1 rounded text-[9px] font-medium bg-blue-600 hover:bg-blue-500 text-white transition-colors">
              <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="M12 5v14"/></svg>
              Create
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  
  {#if deleteTarget}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" onclick={(e) => { if (e.target === e.currentTarget) deleteTarget = null }}>
      <div class="w-72 rounded-lg border border-gray-700 bg-gray-900 shadow-2xl">
        <div class="flex flex-col items-center gap-2.5 p-5">
          <div class="p-2.5 rounded-full bg-red-500/20">
            <svg class="w-5 h-5 text-red-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/></svg>
          </div>
          <div class="text-center">
            <p class="text-[10px] font-medium text-gray-200">Delete {deleteTarget.type}?</p>
            <p class="text-[9px] text-gray-500 mt-0.5 font-mono">{deleteTarget.path}</p>
          </div>
          <div class="flex gap-1.5 w-full">
            <button onclick={() => deleteTarget = null} class="flex-1 py-1.5 rounded text-[9px] text-gray-500 hover:text-gray-200 border border-gray-700 hover:border-gray-500 transition-colors">Cancel</button>
            <button
              onclick={() => handleDelete(deleteTarget!.name, deleteTarget!.type)}
              class="flex-1 flex items-center justify-center gap-1 py-1.5 rounded text-[9px] font-medium bg-red-600 hover:bg-red-500 text-white transition-colors"
            >
              <svg class="w-3 h-3" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
              Delete
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  
  {#if toastMessage}
    <div class="fixed bottom-4 right-4 z-50 px-3 py-2 rounded-lg text-[10px] font-medium shadow-lg {
      toastType === 'success' ? 'bg-green-500/90 text-white' : 'bg-red-500/90 text-white'
    }">
      {toastMessage}
    </div>
  {/if}
</div>
