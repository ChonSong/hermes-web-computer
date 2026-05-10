<script lang="ts">
  import { onMount } from "svelte"
  import { Terminal } from "@xterm/xterm"
  import { FitAddon } from "@xterm/addon-fit"
  import { send } from "../stores/ws"

  let container: HTMLDivElement

  onMount(() => {
    const term = new Terminal({
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

    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(container)
    fitAddon.fit()

    term.onData((data) => {
      send({ protocol: "agent", method: "pty.write", params: { data } })
    })

    term.writeln("Agent-OS v1.2 — Welcome")
    term.writeln("Shift+Space to interrupt")
    term.writeln("")

    const resizeObserver = new ResizeObserver(() => fitAddon.fit())
    resizeObserver.observe(container)

    return () => {
      resizeObserver.disconnect()
      term.dispose()
    }
  })
</script>

<div bind:this={container} class="w-full h-full"></div>
