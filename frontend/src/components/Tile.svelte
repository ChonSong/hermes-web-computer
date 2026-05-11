<script lang="ts">
  import Terminal from "./Terminal.svelte"
  import Monaco from "./Monaco.svelte"
  import Browser from "./Browser.svelte"
  import { send, focus } from "../stores/ws"
  import type { LayoutTree } from "../stores/ws"

  let { node, depth = 0 }: { node: LayoutTree; depth?: number } = $props()

  function handleSplit(dir: string) {
    send({ protocol: "ui", method: "layout.update", params: {
      op: "split", target_id: node.id, direction: dir, content: "xterm"
    }})
  }

  function handleClose() {
    send({ protocol: "ui", method: "layout.update", params: {
      op: "unmount", target_id: node.id
    }})
  }
</script>

{#if depth >= 3}
  <div class="p-4 text-gray-500 text-center">Max tile depth</div>
{:else if node.type === 'split'}
  <div class="flex flex-col md:flex-row w-full h-full gap-1">
    {#each node.children ?? [] as child}
      <div class="flex-1 min-h-0 min-w-0">
        <Tile node={child} depth={depth + 1} />
      </div>
    {/each}
  </div>
{:else}
  <div
    class="w-full h-full border-2 rounded border-blue-500 cursor-pointer relative bg-gray-950"
    tabindex="0"
    onfocus={() => focus.set(node.id)}
    ondblclick={() => handleSplit('h')}
    onkeydown={(e) => {
      if (e.shiftKey && e.key === 'D') handleSplit(node.direction === 'h' ? 'v' : 'h')
      if (e.shiftKey && e.key === 'Q') handleClose()
      if (e.shiftKey && e.key === 'F') {
        send({ protocol: "ui", method: "layout.update", params: { op: "fullscreen", target_id: node.id }})
      }
    }}
  >
    {#if node.content === 'xterm'}
      <Terminal ptyId={node.pty_id} />
    {:else if node.content === 'editor'}
      <Monaco filePath={node.path} />
    {:else if node.content === 'browser'}
      <Browser id={node.id} browserId={node.browser_id ?? ''} />
    {:else if node.content === 'welcome'}
      <div class="flex items-center justify-center h-full text-gray-400">
        <div class="text-center">
          <p class="text-lg font-bold">Agent-OS v1.2</p>
          <p class="text-sm mt-2">Double-click to split</p>
          <p class="text-sm">Shift+Space to interrupt</p>
        </div>
      </div>
    {:else}
      <div class="flex items-center justify-center h-full text-gray-500">
        <p>{node.content || 'Empty tile'}</p>
      </div>
    {/if}
  </div>
{/if}
