<script lang="ts">
  /**
   * ProfilePanel — Right panel tab for browsing and managing agent profiles.
   */
  import { onMount } from "svelte"
  import { profileStore, type Profile } from "../stores/profiles.svelte"

  let loading = $derived(profileStore.loading)
  let profiles = $derived(profileStore.profiles)
  let activeProfile = $derived(profileStore.activeProfile)
  let error = $derived(profileStore.error)

  onMount(() => {
    profileStore.refresh()
    profileStore.getActive()
  })

  function formatTime(ts: number): string {
    return new Date(ts * 1000).toLocaleString()
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <h2 class="text-white font-semibold text-base">Profiles</h2>
  </div>

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if loading}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading profiles...</div>
  {:else}
    <div class="flex-1 overflow-y-auto px-2 py-2 space-y-1">
      {#if activeProfile}
        <div class="mb-3 px-2 py-2 rounded-lg bg-purple-500/20 border border-purple-500/30">
          <div class="text-xs text-purple-400 mb-1">Active Profile</div>
          <div class="text-white font-medium text-sm">{activeProfile.name}</div>
          <div class="text-gray-400 text-xs">{activeProfile.email}</div>
          <div class="text-gray-500 text-xs mt-1">{activeProfile.role}</div>
        </div>
      {/if}

      {#if profiles.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No profiles found</div>
      {:else}
        <div class="text-xs text-gray-500 uppercase tracking-wider px-2 pt-1">All Profiles</div>
        {#each profiles as profile (profile.id)}
          <div
            class="group flex items-center gap-2 px-2 py-2 rounded-lg transition-colors
                   {activeProfile?.id === profile.id ? 'bg-white/10' : 'hover:bg-white/5'}"
          >
            <div class="w-8 h-8 rounded-full bg-purple-500/30 flex items-center justify-center text-purple-300 text-sm font-medium">
              {profile.name.charAt(0).toUpperCase()}
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-gray-200 text-sm truncate">{profile.name}</div>
              <div class="text-gray-500 text-xs truncate">{profile.email}</div>
            </div>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>