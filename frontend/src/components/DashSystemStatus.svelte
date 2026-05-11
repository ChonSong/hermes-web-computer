<script lang="ts">
  /**
   * DashSystemStatus — System health & configuration tile
   * Migrated from agent-os StatusBar + ConfigPage concepts
   * Shows gateway status, resource usage, and quick config controls.
   */
  import { onMount } from "svelte"
  import { send, on, ws } from "../stores/ws"

  interface SysInfo {
    hostname: string
    os: string
    arch: string
    cpus: number
    total_mem_gb: number
    uptime: number
    node_version: string
    gateway_version: string
  }

  interface ResourceUsage {
    cpu_percent: number
    mem_used_gb: number
    mem_total_gb: number
    mem_percent: number
    disk_used_gb: number
    disk_total_gb: number
    disk_percent: number
  }

  interface ServiceStatus {
    name: string
    running: boolean
    pid: number | null
    uptime: number | null
  }

  let sysInfo = $state<SysInfo | null>(null)
  let resources = $state<ResourceUsage | null>(null)
  let services = $state<ServiceStatus[]>([])
  let loading = $state(true)
  let refreshInterval: ReturnType<typeof setInterval> | null = null

  function formatUptime(seconds: number): string {
    const d = Math.floor(seconds / 86400)
    const h = Math.floor((seconds % 86400) / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    if (d > 0) return `${d}d ${h}h`
    if (h > 0) return `${h}h ${m}m`
    return `${m}m`
  }

  function loadSysInfo() {
    send({ protocol: "ui", method: "system.info" })
    send({ protocol: "ui", method: "system.resources" })
    send({ protocol: "ui", method: "system.services" })
  }

  on("system.info.result", (data: any) => {
    if (data) sysInfo = data as SysInfo
    loading = false
  })

  on("system.resources.result", (data: any) => {
    if (data) resources = data as ResourceUsage
  })

  on("system.services.result", (data: any) => {
    if (data?.services) services = data.services as ServiceStatus[]
  })

  onMount(() => {
    loadSysInfo()
    refreshInterval = setInterval(loadSysInfo, 30000)
    return () => {
      if (refreshInterval) clearInterval(refreshInterval)
    }
  })
</script>

<div class="flex flex-col h-full overflow-hidden bg-gray-950">
  
  <div class="flex items-center justify-between px-4 py-2.5 border-b border-gray-800 shrink-0">
    <div class="flex items-center gap-2">
      <svg class="w-3.5 h-3.5 text-green-400" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M7 7h.01"/><path d="M17 7h.01"/><path d="M7 17h.01"/><path d="M17 17h.01"/><path d="M7 12h10"/></svg>
      <h2 class="text-xs font-semibold text-gray-200">System Status</h2>
    </div>
    <div class="flex items-center gap-1.5">
      <span class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[9px] {
        $ws.connected ? 'bg-green-500/10 text-green-400' : 'bg-red-500/10 text-red-400'
      }">
        <span class="w-1.5 h-1.5 rounded-full {$ws.connected ? 'bg-green-400' : 'bg-red-400'}" />
        {$ws.connected ? "Connected" : "Offline"}
      </span>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto p-3">
    {#if loading && !sysInfo}
      <div class="flex items-center justify-center h-32">
        <svg class="w-5 h-5 animate-spin text-gray-600" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>
      </div>
    {:else}
      <!-- System Info -->
      {#if sysInfo}
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2.5 mb-2">
          <div class="flex items-center gap-1.5 pb-1.5">
            <svg class="w-3 h-3 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="2" rx="2" ry="2"/><rect width="20" height="8" x="2" y="14" rx="2" ry="2"/><line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/></svg>
            <span class="text-[9px] font-bold uppercase tracking-wider text-gray-500">System</span>
          </div>
          <div class="grid grid-cols-2 gap-1 text-[10px]">
            <div class="text-gray-500">Hostname</div><div class="text-gray-300 font-mono">{sysInfo.hostname}</div>
            <div class="text-gray-500">OS</div><div class="text-gray-300">{sysInfo.os} ({sysInfo.arch})</div>
            <div class="text-gray-500">CPUs</div><div class="text-gray-300">{sysInfo.cpus} cores</div>
            <div class="text-gray-500">Memory</div><div class="text-gray-300">{sysInfo.total_mem_gb} GB</div>
            <div class="text-gray-500">Gateway</div><div class="text-gray-300 font-mono">{sysInfo.gateway_version}</div>
            <div class="text-gray-500">Uptime</div><div class="text-gray-300">{formatUptime(sysInfo.uptime)}</div>
          </div>
        </div>
      {/if}

      <!-- Resource Usage -->
      {#if resources}
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2.5 mb-2">
          <div class="flex items-center gap-1.5 pb-1.5">
            <svg class="w-3 h-3 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-2.48a2 2 0 0 0-1.93 1.46l-2.35 8.36a.25.25 0 0 1-.48 0L9.24 2.18a.25.25 0 0 0-.48 0l-2.35 8.36A2 2 0 0 1 4.49 12H2"/></svg>
            <span class="text-[9px] font-bold uppercase tracking-wider text-gray-500">Resources</span>
          </div>

          <!-- CPU -->
          <div class="mb-2">
            <div class="flex justify-between text-[9px] mb-0.5">
              <span class="text-gray-400">CPU</span>
              <span class="text-gray-500">{resources.cpu_percent.toFixed(1)}%</span>
            </div>
            <div class="h-1.5 bg-gray-800 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all {resources.cpu_percent > 80 ? 'bg-red-500' : resources.cpu_percent > 50 ? 'bg-amber-500' : 'bg-green-500'}"
                style="width: {Math.min(resources.cpu_percent, 100)}%"
              />
            </div>
          </div>

          <!-- Memory -->
          <div class="mb-2">
            <div class="flex justify-between text-[9px] mb-0.5">
              <span class="text-gray-400">Memory</span>
              <span class="text-gray-500">{resources.mem_used_gb.toFixed(1)} / {resources.mem_total_gb} GB ({resources.mem_percent.toFixed(0)}%)</span>
            </div>
            <div class="h-1.5 bg-gray-800 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all {resources.mem_percent > 80 ? 'bg-red-500' : resources.mem_percent > 50 ? 'bg-amber-500' : 'bg-green-500'}"
                style="width: {Math.min(resources.mem_percent, 100)}%"
              />
            </div>
          </div>

          <!-- Disk -->
          <div>
            <div class="flex justify-between text-[9px] mb-0.5">
              <span class="text-gray-400">Disk</span>
              <span class="text-gray-500">{resources.disk_used_gb.toFixed(1)} / {resources.disk_total_gb} GB ({resources.disk_percent.toFixed(0)}%)</span>
            </div>
            <div class="h-1.5 bg-gray-800 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full transition-all {resources.disk_percent > 80 ? 'bg-red-500' : resources.disk_percent > 50 ? 'bg-amber-500' : 'bg-green-500'}"
                style="width: {Math.min(resources.disk_percent, 100)}%"
              />
            </div>
          </div>
        </div>
      {/if}

      <!-- Services -->
      {#if services.length > 0}
        <div class="bg-gray-900/50 border border-gray-800 rounded p-2.5">
          <div class="flex items-center gap-1.5 pb-1.5">
            <svg class="w-3 h-3 text-gray-500" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z"/><path d="m9 12 2 2 4-4"/></svg>
            <span class="text-[9px] font-bold uppercase tracking-wider text-gray-500">Services</span>
          </div>
          <div class="space-y-1">
            {#each services as svc}
              <div class="flex items-center justify-between py-0.5 text-[10px]">
                <div class="flex items-center gap-1.5">
                  <span class="w-1.5 h-1.5 rounded-full {svc.running ? 'bg-green-400' : 'bg-gray-500'}" />
                  <span class="text-gray-300">{svc.name}</span>
                </div>
                <div class="text-gray-500">
                  {#if svc.running && svc.uptime}
                    {formatUptime(svc.uptime)}
                  {:else}
                    stopped
                  {/if}
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      {#if !sysInfo && !resources && services.length === 0}
        <div class="flex flex-col items-center text-gray-500 py-6">
          <svg class="w-6 h-6 mb-2 opacity-30" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M7 7h.01"/><path d="M17 7h.01"/><path d="M7 17h.01"/><path d="M17 17h.01"/><path d="M7 12h10"/></svg>
          <p class="text-[10px]">Waiting for system data...</p>
        </div>
      {/if}
    {/if}
  </div>
</div>
