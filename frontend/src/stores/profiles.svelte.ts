/**
 * profiles.svelte.ts — Svelte 5 reactive profile store
 * Handles profile listing, active profile retrieval, and profile state.
 */
import { send, on } from "./ws"

export interface Profile {
  id: string
  name: string
  email: string
  role: string
  created_at: number
}

class ProfileState {
  profiles = $state<Profile[]>([])
  activeProfile = $state<Profile | null>(null)
  loading = $state(false)
  error = $state<string | null>(null)

  async refresh(): Promise<void> {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("profiles.list", (data: any) => {
        cleanup()
        this.profiles = (data?.profiles ?? []) as Profile[]
        this.loading = false
        resolve()
      })
      const errCleanup = on("profiles.list.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        this.loading = false
        resolve()
      })
      send({ protocol: "ui", method: "profiles.list" })
      setTimeout(() => { this.loading = false; resolve() }, 5000)
    })
  }

  async getActive(): Promise<Profile | null> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("profiles.active", (data: any) => {
        cleanup()
        this.activeProfile = data as Profile
        resolve(data as Profile)
      })
      const errCleanup = on("profiles.active.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(null)
      })
      send({ protocol: "ui", method: "profiles.active" })
    })
  }
}

export const profileStore = new ProfileState()