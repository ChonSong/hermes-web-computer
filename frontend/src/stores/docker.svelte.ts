/**
 * docker.svelte.ts — Svelte 5 reactive docker store
 * Handles container listing, stats, logs, and lifecycle actions.
 */
import { send, on } from "./ws"

export interface Container {
  id: string
  name: string
  image: string
  state: "running" | "stopped" | "paused" | "restarting" | "exited" | "dead"
  status: string
  created: number
  ports: string
}

export interface ContainerStats {
  id: string
  cpu_percent: number
  mem_usage: string
  mem_limit: string
  mem_percent: number
  net_rx: string
  net_tx: string
  block_read: string
  block_write: string
}

class DockerState {
  containers = $state<Container[]>([])
  selectedId = $state<string | null>(null)
  stats = $state<Map<string, ContainerStats>>(new Map())
  logs = $state<Map<string, string>>(new Map())
  loading = $state(false)
  error = $state<string | null>(null)
  searchQuery = $state("")

  filteredContainers = $derived(
    this.containers.filter((c) => {
      const q = this.searchQuery.toLowerCase()
      if (!q) return true
      return (
        c.name.toLowerCase().includes(q) ||
        c.state.includes(q) ||
        c.image.toLowerCase().includes(q)
      )
    })
  )

  async refresh(): Promise<void> {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("docker.list.ok", (data: any) => {
        cleanup()
        this.containers = (data?.containers ?? []) as Container[]
        this.loading = false
        resolve()
      })
      const errCleanup = on("docker.list.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        this.loading = false
        resolve()
      })
      send({ protocol: "ui", method: "docker.list" })
      setTimeout(() => {
        this.loading = false
        resolve()
      }, 8000)
    })
  }

  async start(id: string): Promise<boolean> {
    return this.action(id, "docker.start", "docker.start.ok", "docker.start.error")
  }

  async stop(id: string): Promise<boolean> {
    return this.action(id, "docker.stop", "docker.stop.ok", "docker.stop.error")
  }

  async restart(id: string): Promise<boolean> {
    return this.action(id, "docker.restart", "docker.restart.ok", "docker.restart.error")
  }

  async remove(id: string): Promise<boolean> {
    return this.action(id, "docker.remove", "docker.remove.ok", "docker.remove.error")
  }

  private async action(id: string, method: string, okEvent: string, _errEvent: string): Promise<boolean> {
    return new Promise((resolve) => {
      const cleanup = on(okEvent, (_data: any) => {
        cleanup()
        resolve(true)
        this.refresh()
      })
      const errCleanup = on(`${method}.error`, (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method, params: { id } })
      setTimeout(() => { resolve(false) }, 8000)
    })
  }

  async fetchStats(id: string): Promise<void> {
    return new Promise<void>((resolve) => {
      const cleanup = on("docker.container_stats.ok", (data: any) => {
        cleanup()
        if (data?.id) {
          const stats = data as ContainerStats
          this.stats.set(stats.id, stats)
          this.stats = new Map(this.stats)
        }
        resolve()
      })
      const errCleanup = on("docker.container_stats.error", (_data: any) => {
        errCleanup()
        resolve()
      })
      send({ protocol: "ui", method: "docker.container_stats", params: { id } })
      setTimeout(() => { resolve() }, 5000)
    })
  }

  async fetchLogs(id: string, tail = 100): Promise<void> {
    return new Promise<void>((resolve) => {
      const cleanup = on("docker.container_logs.ok", (data: any) => {
        cleanup()
        if (data?.id) {
          this.logs.set(data.id, data.logs ?? "")
          this.logs = new Map(this.logs)
        }
        resolve()
      })
      const errCleanup = on("docker.container_logs.error", (_data: any) => {
        errCleanup()
        resolve()
      })
      send({ protocol: "ui", method: "docker.container_logs", params: { id, tail } })
      setTimeout(() => { resolve() }, 5000)
    })
  }

  select(id: string | null) {
    this.selectedId = id
    if (id) {
      this.fetchStats(id)
      this.fetchLogs(id)
    }
  }

  clearStats() {
    this.stats = new Map()
    this.logs = new Map()
  }
}

export const dockerStore = new DockerState()