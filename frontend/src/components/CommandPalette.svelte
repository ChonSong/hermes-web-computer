<script lang="ts">
  import { send } from "../stores/ws"
  import { onMount, createEventDispatcher } from "svelte"

  export let visible: boolean = false
  let query = $state("")
  let selectedIndex = $state(0)

  const commands = [
    { name: "New Terminal (Right)", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "split", direction: "h", content: "xterm" }}) },
    { name: "New Terminal (Below)", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "split", direction: "v", content: "xterm" }}) },
    { name: "Split Horizontal", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "swap", direction: "h" }}) },
    { name: "Split Vertical", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "swap", direction: "v" }}) },
    { name: "Close Tile", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "unmount" }}) },
    { name: "Toggle Fullscreen", action: () => send({ protocol: "ui", method: "layout.update", params: { op: "fullscreen" }}) },
  ]

  const filtered = $derived(query ? commands.filter(c => c.name.toLowerCase().includes(query.toLowerCase())) : commands)

  $effect(() => { if (!visible) { query = ""; selectedIndex = 0 } })
</script>

{#if visible}
  <div class="fixed inset-0 bg-black/50 flex items-start justify-center pt-20 z-50" on:click|self={() => visible = false}>
    <div class="w-96 bg-gray-900 rounded-lg border border-gray-700 shadow-xl">
      <input bind:value={query} placeholder="Type a command..." class="w-full p-3 bg-transparent text-white outline-none border-b border-gray-700" autofocus />
      <ul class="max-h-60 overflow-y-auto">
        {#each filtered as cmd, i}
          <li class="p-3 cursor-pointer hover:bg-gray-800" class:bg-gray-800={i === selectedIndex} on:click={() => { cmd.action(); visible = false }}>
            {cmd.name}
          </li>
        {/each}
      </ul>
    </div>
  </div>
{/if}
