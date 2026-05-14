<script lang="ts">
  /**
   * CronPanel — Right panel tab for browsing and managing cron jobs.
   */
  import { onMount } from "svelte"
  import { cronStore, type CronJob } from "../stores/crons.svelte"

  let loading = $derived(cronStore.loading)
  let jobs = $derived(cronStore.jobs)
  let error = $derived(cronStore.error)

  onMount(() => {
    cronStore.refresh()
  })

  function formatTime(ts: number | undefined): string {
    if (!ts) return "never"
    return new Date(ts * 1000).toLocaleString()
  }

  async function handleToggle(job: CronJob) {
    await cronStore.toggle(job.id, !job.enabled)
    cronStore.refresh()
  }

  async function handleDelete(id: string) {
    if (confirm("Delete this cron job?")) {
      await cronStore.delete(id)
      cronStore.refresh()
    }
  }
</script>

<div class="flex flex-col h-full">
  <div class="flex-none px-4 py-3 border-b border-white/10">
    <h2 class="text-white font-semibold text-base">Crons</h2>
  </div>

  {#if error}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-red-400 text-sm">{error}</div>
  {:else if loading}
    <div class="flex-1 overflow-y-auto px-4 py-3 text-gray-500 text-sm">Loading cron jobs...</div>
  {:else}
    <div class="flex-1 overflow-y-auto px-2 py-2 space-y-1">
      {#if jobs.length === 0}
        <div class="text-center py-4 text-gray-500 text-sm">No cron jobs configured</div>
      {:else}
        {#each jobs as job (job.id)}
          <div
            class="group flex items-start gap-2 px-2 py-2 rounded-lg hover:bg-white/5 transition-colors"
          >
            <button
              class="w-6 h-6 rounded-full flex items-center justify-center text-xs shrink-0 mt-0.5 transition-colors
                     {job.enabled ? 'bg-green-500/20 text-green-400' : 'bg-gray-500/20 text-gray-500'}"
              onclick={() => handleToggle(job)}
              title={job.enabled ? "Disable" : "Enable"}
            >
              {job.enabled ? "●" : "○"}
            </button>
            <div class="flex-1 min-w-0">
              <div class="text-gray-200 text-sm font-medium">{job.name}</div>
              <div class="text-gray-500 text-xs font-mono mt-0.5">{job.schedule}</div>
              <div class="text-gray-600 text-xs mt-0.5 truncate">{job.action}</div>
              <div class="flex items-center gap-3 mt-1">
                <span class="text-[10px] text-gray-500">last: {formatTime(job.last_run)}</span>
                <span class="text-[10px] text-gray-500">next: {formatTime(job.next_run)}</span>
              </div>
            </div>
            <button
              class="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-red-500/20 text-gray-500 hover:text-red-400 shrink-0"
              onclick={() => handleDelete(job.id)}
              title="Delete"
            >
              ✕
            </button>
          </div>
        {/each}
      {/if}
    </div>
  {/if}
</div>