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

/** Image represents a Docker image. */
export interface DockerImage {
  id: string
  repository: string
  tag: string
  size: string
  created: string
}

/** ComposeProject represents a docker compose project. */
export interface ComposeProject {
  name: string
  path: string
  services: number
  status: string
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
  images = $state<DockerImage[]>([])
  projects = $state<ComposeProject[]>([])
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

  // create runs a new container from an image.
  async create(image: string, name: string, ports: string[], envVars: string[], volumes: string[]): Promise<{ id?: string; error?: string }> {
    return new Promise((resolve) => {
      const cleanup = on("docker.create.ok", (_data: any) => {
        cleanup()
        this.refresh()
        resolve({ id: _data?.id })
      })
      const errCleanup = on("docker.create.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve({ error: this.error })
      })
      send({ protocol: "ui", method: "docker.create", params: { image, name, ports, env_vars: envVars, volumes } })
      setTimeout(() => { resolve({ error: "timeout" }) }, 30000)
    })
  }

  // listImages fetches the list of Docker images.
  async listImages(): Promise<void> {
    return new Promise<void>((resolve) => {
      const cleanup = on("docker.images.ok", (data: any) => {
        cleanup()
        this.images = (data?.images ?? []) as DockerImage[]
        resolve()
      })
      const errCleanup = on("docker.images.error", (_data: any) => {
        errCleanup()
        resolve()
      })
      send({ protocol: "ui", method: "docker.images" })
      setTimeout(() => { resolve() }, 10000)
    })
  }

  // removeImage removes a Docker image.
  async removeImage(id: string, force = false): Promise<boolean> {
    return new Promise((resolve) => {
      const cleanup = on("docker.image.remove.ok", (_data: any) => {
        cleanup()
        this.listImages()
        resolve(true)
      })
      const errCleanup = on("docker.image.remove.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "docker.image.remove", params: { id, force } })
      setTimeout(() => { resolve(false) }, 15000)
    })
  }

  // pullImage pulls a Docker image from a registry.
  async pullImage(image: string): Promise<{ error?: string }> {
    return new Promise((resolve) => {
      const cleanup = on("docker.image.pull.ok", (_data: any) => {
        cleanup()
        this.listImages()
        resolve({})
      })
      const errCleanup = on("docker.image.pull.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve({ error: this.error })
      })
      send({ protocol: "ui", method: "docker.image.pull", params: { image } })
      setTimeout(() => { resolve({ error: "timeout" }) }, 120000)
    })
  }

  // listProjects fetches the list of compose projects.
  async listProjects(): Promise<void> {
    return new Promise<void>((resolve) => {
      const cleanup = on("docker.compose.ls.ok", (data: any) => {
        cleanup()
        this.projects = (data?.projects ?? []) as ComposeProject[]
        resolve()
      })
      const errCleanup = on("docker.compose.ls.error", (_data: any) => {
        errCleanup()
        resolve()
      })
      send({ protocol: "ui", method: "docker.compose.ls" })
      setTimeout(() => { resolve() }, 10000)
    })
  }

  // composeUp starts a compose project.
  async composeUp(path: string): Promise<boolean> {
    return new Promise((resolve) => {
      const cleanup = on("docker.compose.up.ok", (_data: any) => {
        cleanup()
        this.listProjects()
        resolve(true)
      })
      const errCleanup = on("docker.compose.up.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "docker.compose.up", params: { path } })
      setTimeout(() => { resolve(false) }, 60000)
    })
  }

  // composeDown stops and removes a compose project.
  async composeDown(path: string, removeVolumes = false): Promise<boolean> {
    return new Promise((resolve) => {
      const cleanup = on("docker.compose.down.ok", (_data: any) => {
        cleanup()
        this.listProjects()
        resolve(true)
      })
      const errCleanup = on("docker.compose.down.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "docker.compose.down", params: { path, remove_volumes: removeVolumes } })
      setTimeout(() => { resolve(false) }, 60000)
    })
  }

  // composeStop stops a compose project.
  async composeStop(path: string): Promise<boolean> {
    return new Promise((resolve) => {
      const cleanup = on("docker.compose.stop.ok", (_data: any) => {
        cleanup()
        this.listProjects()
        resolve(true)
      })
      const errCleanup = on("docker.compose.stop.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "docker.compose.stop", params: { path } })
      setTimeout(() => { resolve(false) }, 60000)
    })
  }
}

export const dockerStore = new DockerState()