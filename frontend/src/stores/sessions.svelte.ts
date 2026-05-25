/**
 * sessions.svelte.ts — Svelte 5 reactive session store
 * Mirrors hermes-webui session state in the HWC WS layer.
 * Handles streaming tokens, tool calls, and persistent message history.
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

  // Streaming accumulation — key by sessionId
  private _bufText = $state<Map<string, string>>(new Map())
  private _bufToolCalls = $state<Map<string, any[]>>(new Map())

  get active() {
    return this.sessions.find(s => s.session_id === this.activeId) ?? null
  }

  /** Append a message to a session's message list (in-place mutation for reactivity) */
  private _appendMsg(sid: string, role: SessionMessage["role"], content: string, toolCalls?: any[]) {
    this.sessions = this.sessions.map(s => {
      if (s.session_id !== sid) return s
      const msgs = [...(s.messages ?? []), {
        role,
        content,
        ...(toolCalls ? { tool_calls: toolCalls } : {}),
      } as SessionMessage]
      return { ...s, messages: msgs, updated_at: Date.now() }
    })
    // Update activeSession too
    if (this.activeId === sid) {
      this.activeSession = this.sessions.find(s => s.session_id === sid) ?? null
    }
  }

  /** Get current streaming buffer for a session */
  getBuf(sid: string): { text: string; toolCalls: any[] } {
    return {
      text: this._bufText.get(sid) ?? "",
      toolCalls: this._bufToolCalls.get(sid) ?? [],
    }
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
      setTimeout(() => { this.loading = false; resolve() }, 5000)
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
      const cleanup = on("session.delete.ok", (_data: any) => {
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

  async updateTitle(id: string, title: string): Promise<void> {
    this.sessions = this.sessions.map(s =>
      s.session_id === id ? { ...s, title } : s
    )
    send({ protocol: "ui", method: "session.update", params: { id, title } })
  }

  async archive(id: string, archived: boolean): Promise<void> {
    this.sessions = this.sessions.map(s =>
      s.session_id === id ? { ...s, archived } : s
    )
    send({ protocol: "ui", method: "session.update", params: { id, archived } })
  }

  async duplicate(id: string): Promise<Session | null> {
    const sess = this.sessions.find(s => s.session_id === id)
    if (!sess) return null
    return this.create(sess.workspace, sess.model)
  }

  select(id: string) {
    this.activeId = id
    this.activeSession = this.sessions.find(s => s.session_id === id) ?? null
    this.load(id)
  }

  /**
   * send — streaming-aware chat message.
   * Handles: optimistic user message, token accumulation, tool call cards,
   * tool result messages, and final reply assembly.
   */
  async send(content: string): Promise<void> {
    this.error = null
    const sid = this.activeId
    if (!sid) return

    // 1. Optimistic user message
    this._appendMsg(sid, "user", content)

    // 2. Clear buffers for this session
    this._bufText.set(sid, "")
    this._bufToolCalls.set(sid, [])

    return new Promise((resolve) => {
      let pendingText = ""
      let pendingToolCalls: any[] = []

      // Flush accumulated text as a completed assistant message
      const flush = () => {
        if (pendingText || pendingToolCalls.length > 0) {
          this._appendMsg(sid, "assistant", pendingText,
            pendingToolCalls.length > 0 ? pendingToolCalls : undefined)
          pendingText = ""
          pendingToolCalls = []
        }
      }

      const cleanupAll = () => {
        this._bufText.delete(sid)
        this._bufToolCalls.delete(sid)
      }

      // --- Token events ---
      const t1 = on("chat.token", (data: any) => {
        const tok = (data as any)?.content ?? ""
        pendingText += tok
        this._bufText.set(sid, pendingText)
      })

      // --- Reasoning events (flush text first) ---
      const t2 = on("chat.reasoning", (_data: any) => {
        flush() // flush text before showing reasoning
      })

      // --- Tool call events (flush text, buffer tool calls) ---
      const t3 = on("chat.tool_call", (data: any) => {
        flush()
        pendingToolCalls.push(data)
        const existing = this._bufToolCalls.get(sid) ?? []
        this._bufToolCalls.set(sid, [...existing, data])
      })

      // --- Tool result events ---
      const t4 = on("chat.tool_result", (data: any) => {
        const result = (data as any)?.result ?? ""
        this._appendMsg(sid, "tool", result)
      })

      // --- Final reply ---
      const t5 = on("chat.reply", (_data: any) => {
        flush()
        cleanupAll(); t1(); t2(); t3(); t4(); t5()
        resolve()
      })

      // --- Error ---
      const t6 = on("chat.error", (data: any) => {
        flush()
        this.error = (data as any)?.message ?? "Unknown error"
        cleanupAll(); t1(); t2(); t3(); t4(); t5(); t6()
        resolve()
      })

      send({ protocol: "agent", method: "chat.send", params: { message: content, session_id: sid } })
    })
  }
}

export const sessionStore = new SessionStore()
