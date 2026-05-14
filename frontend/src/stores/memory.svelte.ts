/**
 * memory.svelte.ts — Svelte 5 reactive memory store
 * Handles memory read/write and tracks memory paths and modification times.
 */
import { send, on } from "./ws"

class MemoryState {
  memory = $state<string>("")
  user = $state<string>("")
  paths = $state<string[]>([])
  mtimes = $state<Record<string, number>>({})
  loading = $state(false)
  error = $state<string | null>(null)

  async read(): Promise<void> {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("memory.read", (data: any) => {
        cleanup()
        this.memory = (data as any)?.memory ?? ""
        this.user = (data as any)?.user ?? ""
        this.paths = (data as any)?.paths ?? []
        this.mtimes = (data as any)?.mtimes ?? {}
        this.loading = false
        resolve()
      })
      const errCleanup = on("memory.read.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        this.loading = false
        resolve()
      })
      send({ protocol: "ui", method: "memory.read" })
      setTimeout(() => { this.loading = false; resolve() }, 5000)
    })
  }

  async write(memory: string, user: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("memory.write.ok", (_data: any) => {
        cleanup()
        this.memory = memory
        this.user = user
        resolve(true)
      })
      const errCleanup = on("memory.write.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "memory.write", params: { memory, user } })
    })
  }
}

export const memoryStore = new MemoryState()