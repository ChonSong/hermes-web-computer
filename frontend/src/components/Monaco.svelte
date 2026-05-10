<script lang="ts">
  import { onMount } from "svelte"

  let { path = "" }: { path?: string } = $props()
  let container: HTMLDivElement
  let editor: any = $state(null)

  onMount(async () => {
    const { default: monaco } = await import("monaco-editor")

    editor = monaco.editor.create(container, {
      value: "// Open a file to edit",
      language: getLanguageFromPath(path),
      theme: "vs-dark",
      automaticLayout: true,
      minimap: { enabled: false },
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
    })

    return () => {
      editor.dispose()
    }
  })

  function getLanguageFromPath(p: string): string {
    const ext = p.split(".").pop()?.toLowerCase()
    const langMap: Record<string, string> = {
      ts: "typescript", js: "javascript", py: "python", go: "go",
      rs: "rust", html: "html", css: "css", json: "json", yaml: "yaml",
      md: "markdown", sh: "shell", bash: "shell", sql: "sql",
    }
    return langMap[ext || ""] || "plaintext"
  }
</script>

<div bind:this={container} class="w-full h-full"></div>