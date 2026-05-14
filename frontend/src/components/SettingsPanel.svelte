<script lang="ts">
  /**
   * SettingsPanel — Right panel tab for app settings and configuration.
   */
  let activeSection = $state<string | null>(null)

  const settingsSections = [
    { id: "general", label: "General", icon: "⚙️" },
    { id: "appearance", label: "Appearance", icon: "🎨" },
    { id: "shortcuts", label: "Shortcuts", icon: "⌨️" },
    { id: "connection", label: "Connection", icon: "🔌" },
    { id: "advanced", label: "Advanced", icon: "🔧" },
  ]
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <h2 class="text-white font-semibold text-base">Settings</h2>
  </div>

  <div class="flex-1 overflow-y-auto px-2 py-2">
    <div class="space-y-1">
      {#each settingsSections as section}
        <button
          class="w-full flex items-center gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors text-left
                 {activeSection === section.id ? 'bg-white/10' : ''}"
          onclick={() => activeSection = activeSection === section.id ? null : section.id}
        >
          <span class="text-base">{section.icon}</span>
          <span class="text-gray-200 text-sm">{section.label}</span>
        </button>
      {/each}
    </div>

    {#if activeSection}
      <div class="mt-4 px-2 py-3 bg-white/5 rounded-lg">
        <div class="text-xs text-gray-400 uppercase tracking-wider mb-2">{activeSection}</div>
        <div class="text-sm text-gray-400">No settings available for this section yet.</div>
      </div>
    {/if}
  </div>
</div>