/**
 * sessions.svelte.ts — Svelte 5 reactive session store
 * Mirrors hermes-webui session state in the HWC WS layer.
 */
import { send, on } from "./ws"

export interface SessionMessage {
  role: "user" | "assistant" | "system" | "tool"
  content: string
  tool_calls?: Array<{
    id: string
    type: "function"
    function: { name: string; arguments: string }
  }>
  tool_call_id?: string
  name?: string
}

export interface Session {
  session_id: string
  title: string
  workspace: string
  model: string
  pinned: boolean
  archived: boolean
  project_id?: string
  created_at: number
  updated_at: number
  message_count?: number
  messages?: SessionMessage[]
}

// Reactive state using Svelte 5 runes pattern
class SessionStore {
  sessions = $state<Session[]>([])
  activeId = $state<string | null>(null)
  activeSession = $state<Session | null>(null)
  loading = $state(false)
  error = $state<string | null>(null)

  get active() {
    return this.sessions.find(s => s.session_id === this.activeId) ?? null
  }

  async refresh() {
    this.loading = true
    this.error = null
    return new Promise<void>((resolve) => {
      const cleanup = on("session.list", (data: any) => {
        cleanup()
        this.sessions = (data?.sessions ?? []) as Session[]
        this.loading = false
        resolve()
      })
      send({ protocol: "ui", method: "session.list" })
      // Timeout fallback
      setTimeout(() => {
        this.loading = false
        resolve()
      }, 5000)
    })
  }

  async create(workspace?: string, model?: string): Promise<Session | null> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("session.new.ok", (data: any) => {
        cleanup()
        const sess = data as Session
        this.sessions = [sess, ...this.sessions]
        this.activeId = sess.session_id
        this.activeSession = sess
        resolve(sess)
      })
      const errCleanup = on("session.new.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(null)
      })
      send({ protocol: "ui", method: "session.new", params: { workspace, model } })
    })
  }

  async load(id: string): Promise<Session | null> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("session.get", (data: any) => {
        cleanup()
        const sess = data as Session
        this.activeId = id
        this.activeSession = sess
        // Update in list too
        this.sessions = this.sessions.map(s => s.session_id === id ? sess : s)
        resolve(sess)
      })
      const errCleanup = on("session.get.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(null)
      })
      send({ protocol: "ui", method: "session.get", params: { id } })
    })
  }

  async delete(id: string): Promise<boolean> {
    this.error = null
    return new Promise((resolve) => {
      const cleanup = on("session.delete.ok", (data: any) => {
        cleanup()
        this.sessions = this.sessions.filter(s => s.session_id !== id)
        if (this.activeId === id) {
          this.activeId = this.sessions[0]?.session_id ?? null
          this.activeSession = this.sessions[0] ?? null
        }
        resolve(true)
      })
      const errCleanup = on("session.delete.error", (data: any) => {
        errCleanup()
        this.error = (data as any)?.message ?? "Unknown error"
        resolve(false)
      })
      send({ protocol: "ui", method: "session.delete", params: { id } })
    })
  }

  async pin(id: string, pinned: boolean): Promise<void> {
    this.sessions = this.sessions.map(s =>
      s.session_id === id ? { ...s, pinned } : s
    )
    send({ protocol: "ui", method: "session.update", params: { id, pinned } })
  }

  select(id: string) {
    this.activeId = id
    this.activeSession = this.sessions.find(s => s.session_id === id) ?? null
    this.load(id)
  }
}

export const sessionStore = new SessionStore()
