<script lang="ts">
  import { onMount, onDestroy } from "svelte"
  import { Terminal } from "@xterm/xterm"
  import { FitAddon } from "@xterm/addon-fit"
  import { send, ptyOutputs } from "../stores/ws"

  export let ptyId: string = ""

  let container: HTMLDivElement | undefined = $state()
  let term: Terminal | undefined = $state()
  let fitAddon: FitAddon | undefined = $state()
  let resizeObserver: ResizeObserver | undefined = $state()

  let outputBuffer = $derived(ptyId ? ($ptyOutputs.get(ptyId) || "") : "")
  let lastLength = $state(0)

  onMount(() => {
    if (!container) return

    term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
      theme: {
        background: "#0a0a0a",
        foreground: "#e0e0e0",
        cursor: "#60a5fa",
        selectionBackground: "#334155",
      },
    })

    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(container)
    fitAddon.fit()

    // Terminal input → WS
    term.onData((data: string) => {
      send({ protocol: "agent", method: "pty.write", params: { data } })
    })

    resizeObserver = new ResizeObserver(() => fitAddon?.fit())
    resizeObserver.observe(container)

    return () => {
      resizeObserver?.disconnect()
      term?.dispose()
    }
  })

  // Incremental PTY output rendering
  $effect(() => {
    if (term && outputBuffer.length > lastLength) {
      term.write(outputBuffer.slice(lastLength))
      lastLength = outputBuffer.length
    }
  })
</script>

<div bind:this={container} class="w-full h-full"></div>
