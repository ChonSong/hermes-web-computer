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
  type: string
  content?: string
  direction?: string
  children?: LayoutTree[]
  path?: string
  pty_id?: string
  size?: number
}

export interface LayoutOp {
  op: string
  target_id?: string
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

export const focus = writable<string>("root")
export const ptyOutputs = writable<Map<string, string>>(new Map())

let socket: WebSocket | null = null
const handlers: Map<string, (data: unknown) => void> = new Map()

let reqId = 0
function nextId(): string {
  return `req_${++reqId}`
}

export function connect(url: string = "ws://localhost:3005/ws") {
  if (socket?.readyState === WebSocket.OPEN) return

  socket = new WebSocket(url)

  socket.onopen = () => {
    ws.set({ connected: true, lastError: null })
  }

  socket.onmessage = (ev) => {
    const event: Event = JSON.parse(ev.data)
    if (event.event === "layout.initial" || event.event === "layout.delta") {
      layout.update((l) => ({
        ...l,
        tree: event.data?.tree as LayoutTree,
        version: (event.data?.layout_version as number) || l.version + 1,
      }))
    }
    if (event.protocol === "agent" && event.event === "pty.output") {
      const data = event.data as { pty_id: string; data: string }
      ptyOutputs.update(map => {
        const prev = map.get(data.pty_id) || ""
        map.set(data.pty_id, prev + data.data)
        return map
      })
    }
    const handler = handlers.get(event.event)
    if (handler) handler(event.data)
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

export function sendOp(op: LayoutOp): string {
  return send({ protocol: "ui", method: "layout.update", params: op as any })
}

export function on(event: string, handler: (data: unknown) => void): () => void {
  handlers.set(event, handler)
  return () => handlers.delete(event)
}

// Don't auto-connect - let main.ts call connect()
