<script lang="ts">
  import { onMount, createEventDispatcher } from "svelte"
  import { send, on, fsList } from "../stores/ws"

  interface Entry {
    name: string
    type: "file" | "directory"
    size: number
    mod_time: string
  }

  interface Events {
    "file:open": { path: string }
  }

  const dispatch = createEventDispatcher<Events>()

  let currentPath = $state("/opt/data/hermes-web-computer")
  let entries = $state<Entry[]>([])
  let loading = $state(false)
  let error = $state<string | null>(null)

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

  function fetchEntries(path: string) {
    loading = true
    error = null
    entries = []

    // Register handlers BEFORE sending to avoid race condition
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

    // Now send the request
    send({ protocol: "ui", method: "fs.list", params: { path } })

    // Timeout fallback
    setTimeout(() => {
      if (loading) {
        unsub()
        unsubErr()
        loading = false
        error = "Request timed out"
      }
    }, 10_000)
  }

  function fullPathFor(entry: Entry): string {
    return currentPath === "/" ? `/${entry.name}` : `${currentPath}/${entry.name}`
  }

  function handleFileClick(entry: Entry) {
    if (entry.type === "directory") {
      navigateTo(fullPathFor(entry))
    } else {
      dispatch("file:open", { path: fullPathFor(entry) })
    }
  }

  function handleDragStart(e: DragEvent, entry: Entry) {
    if (entry.type !== "file") return
    const filePath = fullPathFor(entry)
    e.dataTransfer?.setData("text/plain", filePath)
    e.dataTransfer!.effectAllowed = "copy"
    // Visual: mark dragged item
    setTimeout(() => (e.target as HTMLElement)?.classList.add("opacity-50"), 0)
  }

  function handleDragEnd(e: DragEvent) {
    (e.target as HTMLElement)?.classList.remove("opacity-50")
  }

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
  })
</script>

<div class="flex flex-col h-full text-gray-200 text-sm font-mono">
  <!-- Breadcrumb Navigation -->
  <div class="flex items-center gap-1 px-3 py-2 border-b border-white/5 overflow-x-auto whitespace-nowrap">
    {#each breadcrumbs() as crumb, i}
      <button
        class="text-purple-400 hover:text-purple-300 hover:underline cursor-pointer"
        onclick={() => navigateTo(crumb || "/")}
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
    <div class="px-3 py-1 border-b border-white/5">
      <button
        class="flex items-center gap-1 px-2 py-1 rounded-lg hover:bg-white/5 text-gray-400 hover:text-gray-200 cursor-pointer transition-colors"
        onclick={navigateUp}
      >
        <span>..</span>
        <span class="text-xs text-gray-500">Parent Directory</span>
      </button>
    </div>
  {/if}

  <!-- Loading Spinner -->
  {#if loading}
    <div class="flex items-center justify-center py-8">
      <div class="animate-spin h-5 w-5 border-2 border-purple-400 border-t-transparent rounded-full"></div>
      <span class="ml-2 text-gray-400">Loading...</span>
    </div>
  {/if}

  <!-- Error Message -->
  {#if error}
    <div class="flex items-center justify-center py-8">
      <span class="text-red-400">⚠ {error}</span>
      <button
        class="ml-3 px-2 py-1 bg-red-900/50 text-red-300 rounded hover:bg-red-900/80 cursor-pointer"
        onclick={() => fetchEntries(currentPath)}
      >
        Retry
      </button>
    </div>
  {/if}

  <!-- File List -->
  {#if !loading && !error}
    <div class="flex-1 overflow-y-auto">
      {#if entries.length === 0}
        <div class="flex items-center justify-center py-8 text-gray-500">
          Empty directory
        </div>
      {:else}
        <ul>
          {#each entries as entry (entry.name)}
            <li
              class="flex items-center gap-2 px-3 py-1.5 hover:bg-white/5 cursor-pointer transition-colors rounded-lg"
              onclick={() => handleFileClick(entry)}
              draggable={entry.type === "file"}
              ondragstart={(e) => handleDragStart(e, entry)}
              ondragend={handleDragEnd}
            >
              <span class="text-base shrink-0">
                {#if entry.type === "directory"}
                  📁
                {:else}
                  📄
                {/if}
              </span>
              <span class="flex-1 truncate {entry.type === "directory" ? "text-purple-300 font-medium" : "text-gray-300"}">
                {entry.name}
              </span>
              {#if entry.type === "file"}
                <span class="text-gray-500 text-xs shrink-0">
                  {formatSize(entry.size)}
                </span>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>
