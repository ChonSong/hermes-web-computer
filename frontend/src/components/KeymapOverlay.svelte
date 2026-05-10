<script lang="ts">
  export let visible: boolean = false

  const shortcuts = [
    ["Shift+Space", "Universal interrupt", "Global"],
    ["Shift+←→↑↓", "Move focus", "Active tile"],
    ["Shift+D", "Swap split orientation", "Active tile"],
    ["Shift+Alt+←→↑↓", "Resize tile", "Active tile"],
    ["Shift+F", "Toggle fullscreen", "Active tile"],
    ["Shift+Q", "Close tile", "Active tile"],
    ["Ctrl+K", "Command palette", "Global"],
    ["Ctrl+?", "This overlay", "Global"],
    ["Double-click", "Split tile", "Any tile"],
  ]
</script>

{#if visible}
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" on:click|self={() => visible = false}>
    <div class="bg-gray-900 rounded-lg border border-gray-700 p-6 max-w-lg">
      <h2 class="text-lg text-white mb-4">Keyboard Shortcuts</h2>
      <table class="w-full text-sm">
        <thead><tr class="text-gray-400"><th class="text-left pb-2">Shortcut</th><th class="text-left pb-2">Action</th><th class="text-left pb-2">Scope</th></tr></thead>
        <tbody>
          {#each shortcuts as [key, action, scope]}
            <tr class="border-t border-gray-800">
              <td class="py-2 font-mono text-blue-400">{key}</td>
              <td class="py-2 text-gray-300">{action}</td>
              <td class="py-2 text-gray-500">{scope}</td>
            </tr>
          {/each}
        </tbody>
      </table>
      <p class="text-gray-500 text-sm mt-4 text-center">Press Escape to close</p>
    </div>
  </div>
{/if}
