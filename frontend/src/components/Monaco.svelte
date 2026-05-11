<script lang="ts">
  import { onMount } from "svelte"
  import { on, fsRead, fsWrite } from "../stores/ws"

  let { path = "" }: { path?: string } = $props()
  let container: HTMLDivElement
  let editor: any = $state(null)
  let loading = $state(true)
  let error = $state<string | null>(null)
  let saved = $state(true)
  let currentPath = $state("")
  let dirty = $state(false)

  $effect(() => { currentPath = path })

  // Illogical Impulse Monaco theme
  const ILLOGICAL_THEME = {
    base: "vs-dark" as const,
    inherit: true,
    rules: [
      { token: "", foreground: "c9d1d9" },
      { token: "comment", foreground: "6b7280", fontStyle: "italic" },
      { token: "keyword", foreground: "a78bfa" },
      { token: "keyword.control", foreground: "a78bfa" },
      { token: "keyword.operator", foreground: "c9d1d9" },
      { token: "string", foreground: "34d399" },
      { token: "number", foreground: "fb923c" },
      { token: "type", foreground: "60a5fa" },
      { token: "type.identifier", foreground: "60a5fa" },
      { token: "function", foreground: "a78bfa" },
      { token: "function.declaration", foreground: "c084fc" },
      { token: "variable", foreground: "c9d1d9" },
      { token: "variable.parameter", foreground: "f9a8d4" },
      { token: "constant", foreground: "fb923c" },
      { token: "constant.character.escape", foreground: "34d399" },
      { token: "tag", foreground: "f87171" },
      { token: "tag.attribute.name", foreground: "a78bfa" },
      { token: "delimiter", foreground: "9ca3af" },
      { token: "regexp", foreground: "34d399" },
    ],
    colors: {
      "editor.background": "#12121a",
      "editor.foreground": "#c9d1d9",
      "editor.lineHighlightBackground": "#1a1a26",
      "editor.selectionBackground": "#a78bfa20",
      "editor.selectionHighlightBackground": "#a78bfa10",
      "editorCursor.foreground": "#a78bfa",
      "editor.findMatchBackground": "#a78bfa30",
      "editor.findMatchHighlightBackground": "#a78bfa20",
      "editorLineNumber.foreground": "#3a3a4a",
      "editorLineNumber.activeForeground": "#a78bfa",
      "editorIndentGuide.background": "#1e1e2e",
      "editorIndentGuide.activeBackground": "#2a2a3e",
      "editorBracketMatch.background": "#a78bfa15",
      "editorBracketMatch.border": "#a78bfa50",
      "editorBracketHighlight.foreground1": "#a78bfa",
      "editorBracketHighlight.foreground2": "#34d399",
      "editorBracketHighlight.foreground3": "#fb923c",
      "editorBracketHighlight.foreground4": "#60a5fa",
      "editorBracketHighlight.foreground5": "#f87171",
      "editorBracketHighlight.foreground6": "#f9a8d4",
      "editorSuggestWidget.background": "#16161e",
      "editorSuggestWidget.border": "rgba(255,255,255,0.1)",
      "editorSuggestWidget.selectedBackground": "rgba(167,139,250,0.15)",
      "editorHoverWidget.background": "#16161e",
      "editorHoverWidget.border": "rgba(255,255,255,0.1)",
      "minimap.background": "#0a0a0f",
      "scrollbar.shadow": "#00000000",
      "scrollbarSlider.background": "rgba(255,255,255,0.1)",
      "scrollbarSlider.hoverBackground": "rgba(255,255,255,0.2)",
    },
  }

  onMount(async () => {
    const { default: monaco } = await import("monaco-editor")

    // Register custom theme
    monaco.editor.defineTheme("illogical-impulse", ILLOGICAL_THEME)

    editor = monaco.editor.create(container, {
      value: loading ? "// Loading..." : "// Open a file to edit",
      language: getLanguageFromPath(path),
      theme: "illogical-impulse",
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
