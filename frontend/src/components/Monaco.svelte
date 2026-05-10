<script lang="ts">
  import { onMount } from "svelte"
  import * as monaco from "monaco-editor"

  interface Props {
    path: string
  }

  let { path }: Props = $props()

  let container: HTMLDivElement | undefined = $state()
  let editor: monaco.editor.IStandaloneCodeEditor | undefined = $state()
  let loaded = $state(false)
  let inViewport = $state(false)

  // Detect language from file extension
  function getLanguageFromPath(p: string): string {
    const ext = p.split(".").pop()?.toLowerCase()
    const langMap: Record<string, string> = {
      ts: "typescript",
      tsx: "typescript",
      js: "javascript",
      jsx: "javascript",
      py: "python",
      go: "go",
      rs: "rust",
      rb: "ruby",
      java: "java",
      c: "c",
      cpp: "cpp",
      h: "c",
      hpp: "cpp",
      cs: "csharp",
      php: "php",
      swift: "swift",
      kt: "kotlin",
      scala: "scala",
      html: "html",
      css: "css",
      scss: "scss",
      json: "json",
      yaml: "yaml",
      yml: "yaml",
      xml: "xml",
      md: "markdown",
      sh: "shell",
      bash: "shell",
      zsh: "shell",
      sql: "sql",
      toml: "toml",
      dockerfile: "dockerfile",
      makefile: "makefile",
      svelte: "html",
    }
    return ext ? langMap[ext] ?? "plaintext" : "plaintext"
  }

  // Lazy load via IntersectionObserver
  $effect(() => {
    if (!container) return

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting && !loaded) {
            inViewport = true
          }
        }
      },
      { threshold: 0.1 }
    )

    observer.observe(container)

    return () => observer.disconnect()
  })

  $effect(() => {
    if (!inViewport || !container || loaded) return
    loaded = true

    // Initialize Monaco
    const model = monaco.editor.createModel("", getLanguageFromPath(path))
    model.setValue(`// Loading ${path}...`)

    editor = monaco.editor.create(container, {
      model,
      theme: "vs-dark",
      automaticLayout: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      minimap: { enabled: false },
      scrollBeyondLastLine: false,
      renderLineHighlight: "all",
      bracketPairColorization: { enabled: true },
      padding: { top: 8, bottom: 8 },
    })

    // Override editor background to match terminal
    editor.updateOptions({
      scrollBeyondLastLine: false,
    })
  })

  // Update model when path changes
  $effect(() => {
    if (!editor || !loaded) return
    const lang = getLanguageFromPath(path)
    const model = monaco.editor.createModel(
      `// ${path}`,
      lang
    )
    editor.setModel(model)
  })
</script>

<div bind:this={container} class="w-full h-full bg-[#0a0a0a]">
  {#if !loaded}
    <div class="flex items-center justify-center h-full text-gray-500 text-sm">
      Loading editor…
    </div>
  {/if}
</div>

<style>
  :global(.monaco-editor) {
    background-color: #0a0a0a !important;
  }
  :global(.monaco-editor-background) {
    background-color: #0a0a0a !important;
  }
  :global(.monaco-editor .margin) {
    background-color: #0a0a0a !important;
  }
</style>
