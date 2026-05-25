/**
 * PreviewPanel.svelte — Inline file preview panel
 * Shows file content with type-aware rendering: markdown, image, code, PDF message
 * Phase 2.1: File browser preview
 */
<script lang="ts">
  import { onMount, onDestroy } from "svelte"
  import { on } from "../stores/ws"

  interface PreviewData {
    path: string
    content: string
    encoding: string
    size: number
    mtime: string | null
    is_binary: boolean
  }

  interface Props {
    filePath?: string | null
    onclose?: () => void
  }

  let { filePath = null, onclose }: Props = $props()

  let preview = $state<PreviewData | null>(null)
  let loading = $state(false)
  let error = $state<string | null>(null)
  let unsubFsRead: (() => void) | null = null
  let unsubErr: (() => void) | null = null

  function fileExt(name: string): string {
    const last = name.lastIndexOf(".")
    return last > 0 ? name.slice(last + 1).toLowerCase() : ""
  }

  function isImage(ext: string): boolean {
    return ["png", "jpg", "jpeg", "gif", "webp", "svg", "bmp", "ico"].includes(ext)
  }

  function isMarkdown(ext: string): boolean {
    return ["md", "markdown", "txt", "rst", "log"].includes(ext)
  }

  function isCode(ext: string): boolean {
    return ["js", "ts", "jsx", "tsx", "vue", "svelte", "css", "scss", "html", "json", "yaml", "yml", "toml", "xml", "sh", "bash", "zsh", "py", "go", "rs", "java", "c", "cpp", "h", "hpp", "cs", "rb", "php", "pl", "sql", "graphql", "env", "gitignore", "dockerfile", "makefile"].includes(ext)
  }

  function isPdf(ext: string): boolean {
    return ["pdf"].includes(ext)
  }

  function getFileName(path: string): string {
    const parts = path.split("/").filter(Boolean)
    return parts[parts.length - 1] || path
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
  }

  function loadFile(path: string) {
    if (!path) return
    loading = true
    error = null
    preview = null

    unsubFsRead = on("fs.read.response", (data: unknown) => {
      const resp = data as PreviewData
      if (resp.path === path) {
        loading = false
        preview = resp
        cleanup()
      }
    })

    unsubErr = on("fs.error", (data: unknown) => {
      const resp = data as { message?: string; path?: string }
      if (resp.path === path) {
        loading = false
        error = resp.message || "Failed to load file"
        cleanup()
      }
    })

    import("../stores/ws").then(({ fsRead }) => {
      fsRead(path)
    })

    setTimeout(() => {
      if (loading) {
        cleanup()
        loading = false
        error = "Request timed out"
      }
    }, 10_000)
  }

  function cleanup() {
    if (unsubFsRead) { unsubFsRead(); unsubFsRead = null }
    if (unsubErr) { unsubErr(); unsubErr = null }
  }

  function handleClose() {
    cleanup()
    preview = null
    filePath = null
    onclose?.()
  }

  function getLanguage(ext: string): string {
    const map: Record<string, string> = {
      js: "javascript", ts: "typescript", jsx: "javascript", tsx: "typescript",
      vue: "html", svelte: "html", css: "css", scss: "css", html: "html",
      json: "json", yaml: "yaml", yml: "yaml", toml: "toml", xml: "xml",
      sh: "bash", bash: "bash", zsh: "bash", py: "python", go: "go",
      rs: "rust", java: "java", c: "c", cpp: "cpp", h: "c", hpp: "cpp",
      cs: "csharp", rb: "ruby", php: "php", pl: "perl", sql: "sql",
      graphql: "graphql", env: "bash", gitignore: "bash", dockerfile: "dockerfile",
      makefile: "makefile",
    }
    return map[ext] || "text"
  }

  $effect(() => {
    if (filePath) {
      loadFile(filePath)
    } else {
      cleanup()
      preview = null
      loading = false
      error = null
    }
  })

  onDestroy(() => {
    cleanup()
  })
</script>

{#if filePath}
  <div class="flex flex-col h-full bg-[#191919] border-l border-white/10 overflow-hidden">
    <!-- Header -->
    <div class="flex items-center justify-between px-3 py-2 border-b border-white/5 shrink-0">
      <div class="flex items-center gap-2 min-w-0">
        <span class="text-base shrink-0">📄</span>
        <span class="text-sm text-gray-200 truncate font-mono">{getFileName(filePath)}</span>
      </div>
      <button
        class="shrink-0 p-1 rounded text-gray-500 hover:text-gray-200 hover:bg-white/10 transition-colors"
        onclick={handleClose}
        title="Close preview"
      >
        <svg class="w-4 h-4" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>
      </button>
    </div>

    <!-- Path breadcrumb -->
    <div class="px-3 py-1 border-b border-white/5 shrink-0">
      <span class="text-[10px] text-gray-500 font-mono truncate">{filePath}</span>
    </div>

    <!-- Content area -->
    <div class="flex-1 overflow-auto">
      {#if loading}
        <div class="flex items-center justify-center h-full">
          <div class="flex items-center gap-2 text-gray-400">
            <div class="animate-spin h-5 w-5 border-2 border-purple-400 border-t-transparent rounded-full"></div>
            <span class="text-sm">Loading preview...</span>
          </div>
        </div>
      {:else if error}
        <div class="flex flex-col items-center justify-center h-full gap-3 p-4">
          <span class="text-red-400 text-sm">⚠ {error}</span>
          <button
            class="px-3 py-1.5 bg-white/5 hover:bg-white/10 text-gray-300 rounded text-sm transition-colors"
            onclick={() => loadFile(filePath)}
          >
            Retry
          </button>
        </div>
      {:else if preview}
        {@const ext = fileExt(getFileName(filePath))}
        {@const content = preview.encoding === "base64"
          ? atob(preview.content)
          : preview.content}

        {#if isImage(ext)}
          <!-- Image preview -->
          <div class="flex items-center justify-center p-4">
            {#if preview.encoding === "base64"}
              <img
                src="data:image/{ext};base64,{preview.content}"
                alt={getFileName(filePath)}
                class="max-w-full max-h-full object-contain rounded shadow-lg"
              />
            {:else}
              <img
                src="data:image/{ext};base64,{btoa(content)}"
                alt={getFileName(filePath)}
                class="max-w-full max-h-full object-contain rounded shadow-lg"
              />
            {/if}
          </div>
        {:else if isPdf(ext)}
          <!-- PDF placeholder -->
          <div class="flex flex-col items-center justify-center h-full gap-3 p-4">
            <svg class="w-12 h-12 text-red-400 opacity-60" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/><path d="M10 9H8"/><path d="M16 13H8"/><path d="M16 17H8"/></svg>
            <p class="text-gray-400 text-sm">PDF preview not supported in browser</p>
            <p class="text-gray-600 text-xs font-mono">{getFileName(filePath)}</p>
          </div>
        {:else if isMarkdown(ext)}
          <!-- Markdown rendered preview -->
          <div class="p-4 prose prose-invert prose-sm max-w-none">
            <pre class="whitespace-pre-wrap text-gray-300 text-sm font-mono leading-relaxed bg-transparent p-0">{content}</pre>
          </div>
        {:else}
          <!-- Code / text preview -->
          <div class="relative">
            {#if isCode(ext)}
              <div class="absolute top-2 right-2 px-2 py-0.5 rounded bg-purple-500/20 text-purple-300 text-[10px] font-mono">
                {getLanguage(ext)}
              </div>
            {/if}
            <pre class="p-4 text-[11px] text-gray-300 font-mono leading-relaxed overflow-x-auto whitespace-pre">{content}</pre>
          </div>
        {/if}
      {/if}
    </div>

    <!-- Footer -->
    {#if preview}
      <div class="px-3 py-1.5 border-t border-white/5 shrink-0 flex items-center justify-between">
        <span class="text-[10px] text-gray-600">
          {formatSize(preview.size)}
        </span>
        {#if preview.mtime}
          <span class="text-[10px] text-gray-600">
            {new Date(preview.mtime).toLocaleString()}
          </span>
        {/if}
      </div>
    {/if}
  </div>
{/if}