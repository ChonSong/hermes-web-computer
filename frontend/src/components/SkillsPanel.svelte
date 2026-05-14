<script lang="ts">
  /**
   * SkillsPanel — Right panel tab for browsing and managing skills.
   */
  import { onMount } from "svelte"
  import { skillsStore, type Skill } from "../stores/skills.svelte"

  let loading = $derived(skillsStore.loading)
  let skills = $derived(skillsStore.skills)
  let error = $derived(skillsStore.error)
  let selectedCategory = $derived(skillsStore.selectedCategory)

  // Group skills by category
  let categories = $derived(Array.from(new Set(skills.map(s => s.category)).values()).sort())

  onMount(() => {
    skillsStore.refresh()
  })

  function toggleCategory(cat: string) {
    if (selectedCategory === cat) {
      skillsStore.refresh(null)
    } else {
      skillsStore.refresh(cat)
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <h2 class="text-white font-semibold text-base">Skills</h2>
  </div>

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if loading}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading skills...</div>
  {:else}
    <!-- Category filter pills -->
    <div class="flex-none px-2 py-2 flex gap-1 flex-wrap">
      <button
        class="px-2 py-1 text-xs rounded-full transition-colors
               {!selectedCategory ? 'bg-purple-600 text-white' : 'bg-white/5 text-gray-400 hover:bg-white/10'}"
        onclick={() => skillsStore.refresh(null)}
      >
        All
      </button>
      {#each categories as cat}
        <button
          class="px-2 py-1 text-xs rounded-full transition-colors
                 {selectedCategory === cat ? 'bg-purple-600 text-white' : 'bg-white/5 text-gray-400 hover:bg-white/10'}"
          onclick={() => toggleCategory(cat)}
        >
          {cat}
        </button>
      {/each}
    </div>

    <!-- Skills list -->
    <div class="flex-1 overflow-y-auto px-2 py-2 space-y-1">
      {#if skills.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No skills found</div>
      {:else}
        {#each skills as skill (skill.name)}
          <div
            class="group flex items-start gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors"
          >
            <div class="w-6 h-6 rounded bg-green-500/20 flex items-center justify-center text-green-400 text-xs shrink-0 mt-0.5">
              ◆
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-gray-200 text-sm font-medium">{skill.name}</div>
              <div class="text-gray-500 text-xs mt-0.5 line-clamp-2">{skill.description}</div>
              <div class="flex items-center gap-2 mt-1">
                <span class="text-[10px] text-purple-400 bg-purple-500/10 px-1.5 py-0.5 rounded">{skill.category}</span>
                {#if skill.enabled}
                  <span class="text-[10px] text-green-400">enabled</span>
                {:else}
                  <span class="text-[10px] text-gray-500">disabled</span>
                {/if}
              </div>
            </div>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>