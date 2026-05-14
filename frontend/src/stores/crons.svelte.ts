/**
 * crons.svelte.ts — Svelte 5 reactive crons store
 * Handles cron jobs listing, creation, update, and deletion.
 */
import { send, on } from "./ws"

export interface CronJob {
  id: string
  name: string
  schedule: string
  action: string
  enabled: boolean
  last_run?: number
  next_run?: number
  status?: string
}

class CronsState {
  jobs = $state<CronJob[]>([])
  loading = $state(false)
  error = $state<string | null>(null)

  async refresh(): Promise<void> {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("crons.list", (data: any) => {
        cleanup()
        this.jobs = (data?.jobs ?? []) as CronJob[]
        this.loading = false
        resolve()
      })
      const errCleanup = on("crons.list.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        this.loading = false
        resolve()
      })
      send({ protocol: "ui", method: "crons.list" })
      setTimeout(() => { this.loading = false; resolve() }, 5000)
    })
  }

  async create(name: string, schedule: string, action: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("crons.create.ok", (_data: any) => {
        cleanup()
        resolve(true)
      })
      const errCleanup = on("crons.create.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "crons.create", params: { name, schedule, action } })
    })
  }

  async update(id: string, updates: Partial<Pick<CronJob, "name" | "schedule" | "action" | "enabled">>): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("crons.update.ok", (_data: any) => {
        cleanup()
        resolve(true)
      })
      const errCleanup = on("crons.update.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "crons.update", params: { id, ...updates } })
    })
  }

  async delete(id: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("crons.delete.ok", (_data: any) => {
        cleanup()
        resolve(true)
      })
      const errCleanup = on("crons.delete.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "crons.delete", params: { id } })
    })
  }

  async toggle(id: string, enabled: boolean): Promise<boolean> {
    return this.update(id, { enabled })
  }
}

export const cronStore = new CronsState()