import { writable } from "svelte/store"

export interface Envelope {
  protocol: "ui" | "agent" | "audio"
  method: string
  params?: Record<string, unknown>
  id: string
  ts: number
}

export interface Event {
  protocol: string
  event: string
  data?: Record<string, unknown>
  ts: number
}

export interface LayoutTree {
  id: string
  type: "leaf" | "split"
  content?: "xterm" | "monaco" | "welcome"
  direction?: "h" | "v"
  children?: LayoutTree[]
  path?: string
  pty_id?: string
  size?: number
}

export interface LayoutOp {
  op: string  // "split", "mount", "unmount", "resize", "swap", "fullscreen"
  target_id: string
  direction?: string
  content?: string
  pty_id?: string
  size?: number
}

export const ws = writable<{ connected: boolean; lastError: string | null }>({
  connected: false,
  lastError: null,
})

export const layout = writable<{ tree: LayoutTree | null; version: number }>({
  tree: null,
  version: 0,
})

// Focus tracking — which tile ID is currently focused
export const focus = writable<string>("root")

// PTY output routing — map from pty_id to output data
export const ptyOutputs = writable<Map<string, string>>(new Map())

let socket: WebSocket | null = null
const handlers: Map<string, (data: unknown) => void> = new Map()

let reqId = 0
function nextId(): string {
  return `req_${++reqId}`
}

export function connect(url: string = "ws://localhost:3001/ws") {
  if (socket?.readyState === WebSocket.OPEN) return

  socket = new WebSocket(url)

  socket.onopen = () => {
    ws.set({ connected: true, lastError: null })
  }

  socket.onmessage = (ev) => {
    try {
      const event: Event = JSON.parse(ev.data)
      if (event.event === "layout.initial" || event.event === "layout.delta") {
        layout.update((l) => ({
          ...l,
          tree: event.data?.tree as LayoutTree,
          version: (event.data?.layout_version as number) || l.version + 1,
        }))
      }
      // Route PTY output to the store
      if (event.protocol === "agent" && event.event === "pty.output") {
        const ptyData = event.data as { pty_id: string; data: string } | undefined
        if (ptyData?.pty_id) {
          ptyOutputs.update((map) => {
            const next = new Map(map)
            const existing = next.get(ptyData.pty_id) || ""
            next.set(ptyData.pty_id, existing + ptyData.data)
            return next
          })
        }
      }
      const handler = handlers.get(event.event)
      if (handler) handler(event.data)
    } catch (e) {
      console.error("WS message parse error:", e)
    }
  }

  socket.onclose = () => {
    ws.set({ connected: false, lastError: "Connection closed" })
    setTimeout(() => connect(url), 2000)
  }

  socket.onerror = () => {
    ws.set({ connected: false, lastError: "Connection error" })
  }
}

export function send(env: Omit<Envelope, "id" | "ts">): string {
  const id = nextId()
  const full: Envelope = { ...env, id, ts: Date.now() }
  socket?.send(JSON.stringify(full))
  return id
}

/** Send a layout mutation operation to the backend */
export function sendOp(op: LayoutOp): string {
  return send({
    protocol: "ui",
    method: "layout.op",
    params: {
      op: op.op,
      target_id: op.target_id,
      direction: op.direction,
      content: op.content,
      pty_id: op.pty_id,
      size: op.size,
    },
  })
}

/** Send a full layout tree update (after client-side mutations) */
export function sendLayoutUpdate(tree: LayoutTree): string {
  return send({
    protocol: "ui",
    method: "layout.update",
    params: { tree },
  })
}

export function on(event: string, handler: (data: unknown) => void): () => void {
  handlers.set(event, handler)
  return () => handlers.delete(event)
}

// Global keyboard interrupt handler
globalThis.addEventListener(
  "keydown",
  (e: KeyboardEvent) => {
    if (e.shiftKey && e.key === " " && !e.isComposing) {
      e.preventDefault()
      send({ protocol: "ui", method: "interrupt" })
    }
  },
  true
)

// Auto-connect on import
connect()
