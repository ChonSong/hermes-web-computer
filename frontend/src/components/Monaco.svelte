<script lang="ts">
  import { onMount } from "svelte"
  import { on, fsRead, fsWrite } from "../stores/ws"

  let { path = "" }: { path?: string } = $props()
  let container: HTMLDivElement
  let editor: any = $state(null)
  let loading = $state(true)
  let error = $state<string | null>(null)
  let saved = $state(true)
  let currentPath = $state(path)
  let dirty = $state(false)

  onMount(async () => {
    const { default: monaco } = await import("monaco-editor")

    editor = monaco.editor.create(container, {
      value: loading ? "// Loading..." : "// Open a file to edit",
      language: getLanguageFromPath(path),
      theme: "vs-dark",
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      readOnly: loading || !!error,
    })

    // Ctrl+S / Cmd+S save shortcut
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
      saveFile()
    })

    // Track dirty state
    editor.onDidChangeModelContent(() => {
      dirty = true
      saved = false
    })

    // If we have a path, fetch the file content
    if (path) {
      currentPath = path
      fsRead(path)
    }

    return () => {
      editor.dispose()
    }
  })

  function saveFile() {
    if (!editor || !currentPath) return
    const content = editor.getValue()
    fsWrite(currentPath, content)
    dirty = false
    saved = true
  }

  // Listen for fs.read.response
  on("fs.read.response", (data: unknown) => {
    const resp = data as { path?: string; content?: string; encoding?: string; error?: string }
    if (resp.error) {
      loading = false
      error = resp.error
      if (editor) {
        editor.setValue(`// Error loading file: ${resp.error}`)
        editor.updateOptions({ readOnly: true })
      }
    } else if (resp.content && editor) {
      loading = false
      editor.setValue(resp.content)
      editor.updateOptions({ readOnly: false })
      dirty = false
      saved = true
    }
  })

  // Listen for fs.write.response (save confirmation)
  on("fs.write.response", (data: unknown) => {
    const resp = data as { path?: string; bytes_written?: number }
    if (resp.path) {
      saved = true
      dirty = false
    }
  })

  // Also listen for fs.error
  on("fs.error", (data: unknown) => {
    const resp = data as { message?: string }
    loading = false
    error = resp.message || "Unknown error"
    if (editor) {
      editor.setValue(`// Error loading file: ${error}`)
      editor.updateOptions({ readOnly: true })
    }
  })

  function getLanguageFromPath(p: string): string {
    const ext = p.split(".").pop()?.toLowerCase()
    const langMap: Record<string, string> = {
      ts: "typescript", js: "javascript", py: "python", go: "go",
      rs: "rust", html: "html", css: "css", json: "json", yaml: "yaml",
      md: "markdown", sh: "shell", bash: "shell", sql: "sql",
      svelte: "html", toml: "toml", env: "ini",
    }
    return langMap[ext || ""] || "plaintext"
  }
</script>

<div class="relative w-full h-full">
  <div bind:this={container} class="w-full h-full"></div>
  <!-- Save indicator -->
  {#if currentPath}
    <div class="absolute top-2 right-2 z-10 flex items-center gap-2">
      <span class="text-xs text-gray-500 font-mono">{currentPath}</span>
      {#if dirty}
        <span class="text-xs text-amber-400">● unsaved</span>
      {:else}
        <span class="text-xs text-green-400">✓ saved</span>
      {/if}
    </div>
  {/if}
</div>
