/**
 * skills.svelte.ts — Svelte 5 reactive skills store
 * Handles skills listing, category selection, and skill content loading.
 */
import { send, on } from "./ws"

export interface Skill {
  name: string
  description: string
  category: string
  enabled: boolean
}

class SkillsState {
  skills = $state<Skill[]>([])
  selectedCategory = $state<string | null>(null)
  loading = $state(false)
  error = $state<string | null>(null)

  async refresh(category?: string): Promise<void> {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("skills.list", (data: any) => {
        cleanup()
        this.skills = (data?.skills ?? []) as Skill[]
        this.loading = false
        resolve()
      })
      const errCleanup = on("skills.list.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        this.loading = false
        resolve()
      })
      const params: Record<string, string> = {}
      if (category) {
        params.category = category
        this.selectedCategory = category
      }
      send({ protocol: "ui", method: "skills.list", params })
      setTimeout(() => { this.loading = false; resolve() }, 5000)
    })
  }

  async loadContent(name: string, file?: string): Promise<string | null> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("skills.content", (data: any) => {
        cleanup()
        resolve((data as any)?.content ?? null)
      })
      const errCleanup = on("skills.content.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(null)
      })
      const params: Record<string, string> = { name }
      if (file) params.file = file
      send({ protocol: "ui", method: "skills.content", params })
    })
  }
}

export const skillsStore = new SkillsState()