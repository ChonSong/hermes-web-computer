<script lang="ts">
  import { onMount } from "svelte"
  import { Terminal } from "@xterm/xterm"
  import { FitAddon } from "@xterm/addon-fit"
  import { send, ptyOutputs } from "../stores/ws"

  let { ptyId = "" }: { ptyId?: string } = $props()
  let container: HTMLDivElement
  let term: Terminal
  let outputBuffer = $derived(ptyId ? ($ptyOutputs.get(ptyId) || "") : "")
  let lastLength = $state(0)

  onMount(() => {
    term = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: "monospace",
      theme: {
        background: "transparent",
        foreground: "#e0e0e0",
        cursor: "#60a5fa",
      },
    })
    const fitAddon = new FitAddon()
    term.loadAddon(fitAddon)
    term.open(container)
    fitAddon.fit()
    term.onData((data) => {
      send({ protocol: "agent", method: "pty.write", params: { data } })
    })
    term.writeln("Agent-OS v1.2")
    term.writeln("Shift+Space to interrupt")
    term.writeln("")
    const ro = new ResizeObserver(() => fitAddon.fit())
    ro.observe(container)
    return () => { ro.disconnect(); term.dispose() }
  })

  $effect(() => {
    if (term && outputBuffer.length > lastLength) {
      term.write(outputBuffer.slice(lastLength))
      lastLength = outputBuffer.length
    }
  })
</script>

<div bind:this={container} style="width: 100%; height: 100%;"></div>
